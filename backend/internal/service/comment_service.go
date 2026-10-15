package service

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"

	"shiguangji/internal/errs"
	"shiguangji/internal/middleware"
	"shiguangji/internal/model"
)

var linkRe = regexp.MustCompile(`https?://`)

var (
	multiSpaceRe   = regexp.MustCompile(`[ \t]{2,}`)
	multiNewlineRe = regexp.MustCompile(`\n{3,}`)
)

// CommentNode 前台评论树节点（仅已通过；email/ip 永不输出）。
type CommentNode struct {
	ID        uint          `json:"id"`
	Nickname  string        `json:"nickname"`
	Content   string        `json:"content"`
	IsAdmin   bool          `json:"is_admin"`
	CreatedAt time.Time     `json:"created_at"`
	Children  []CommentNode `json:"children"`
}

// AdminCommentVO 后台评论列表项（含 email/ip/post_title，供审核参考）。
type AdminCommentVO struct {
	ID        uint      `json:"id"`
	PostID    uint      `json:"post_id"`
	PostTitle string    `json:"post_title"`
	PostSlug  string    `json:"post_slug"`
	ParentID  uint      `json:"parent_id"`
	Nickname  string    `json:"nickname"`
	Email     string    `json:"email"`
	Content   string    `json:"content"`
	IsAdmin   bool      `json:"is_admin"`
	Status    int8      `json:"status"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

// CommentService 评论业务：四层防垃圾（spec §5.4）+ 内容净化（§5.5）。
type CommentService struct {
	db          *gorm.DB
	perPostFreq *middleware.IPLimiter // 同 IP 每文章 1 条/分钟
	perDay      *middleware.IPLimiter // 同 IP 每天 10 条
}

// NewCommentService 构造；内部限流器随服务单例存在（router 装配一次）。
func NewCommentService(db *gorm.DB) *CommentService {
	return &CommentService{
		db:          db,
		perPostFreq: middleware.NewIPLimiter(1, time.Minute),
		perDay:      middleware.NewIPLimiter(10, 24*time.Hour),
	}
}

// Close 释放限流器后台 goroutine（优雅关停时调用）。
func (s *CommentService) Close() {
	s.perPostFreq.Close()
	s.perDay.Close()
}

// SanitizeCommentText 净化：剔除不可见格式字符/控制字符（保留换行与普通空白）、压缩连续空白与换行（≤2）。
func SanitizeCommentText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\n', ' ', '\t':
			b.WriteRune(r)
		default:
			if unicode.IsControl(r) || isInvisible(r) {
				continue
			}
			b.WriteRune(r)
		}
	}
	out := multiSpaceRe.ReplaceAllString(b.String(), " ")
	out = multiNewlineRe.ReplaceAllString(out, "\n\n")
	return strings.TrimSpace(out)
}

// isInvisible Cf 格式字符（零宽空格/连接符、BOM、软连字符、词连接符等）：
// 渲染不可见，还可插入词中拆散敏感词匹配，一律剔除（按 Unicode 类别判断，覆盖整个 Cf 区）。
func isInvisible(r rune) bool {
	return unicode.Is(unicode.Cf, r)
}

// loadSensitiveFilter 每次现查词表现建 trie（单行 SELECT；后台改词表即时生效）。
func (s *CommentService) loadSensitiveFilter(ctx context.Context) *SensitiveFilter {
	var row model.SiteSetting
	err := s.db.WithContext(ctx).
		Where("setting_key = ?", "sensitive_words").First(&row).Error
	if err != nil {
		return &SensitiveFilter{root: &sensitiveNode{children: map[rune]*sensitiveNode{}}}
	}
	return NewSensitiveFilter(row.SettingValue)
}

// Create 访客评论：净化 → 长度校验 → 敏感词/链接数 → 限流 → status=0 待审。
func (s *CommentService) Create(ctx context.Context, slug, nickname, email, content, ip, ua string) (*model.Comment, error) {
	var post model.Post
	if err := s.db.WithContext(ctx).
		Where("slug = ? AND status = ?", slug, model.StatusPublished).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}

	nickname = SanitizeCommentText(nickname)
	content = SanitizeCommentText(content)
	email = strings.TrimSpace(email)
	nickRunes, contentRunes := []rune(nickname), []rune(content)
	if len(nickRunes) < 1 || len(nickRunes) > 20 {
		return nil, errs.New(40001, "昵称长度需在 1-20 字符")
	}
	if len(contentRunes) < 1 || len(contentRunes) > 500 {
		return nil, errs.New(40001, "评论内容长度需在 1-500 字符")
	}
	if len(email) > 100 || (email != "" && !strings.Contains(email, "@")) {
		return nil, errs.New(40001, "邮箱格式不正确")
	}

	if s.loadSensitiveFilter(ctx).Hit(content) {
		return nil, errs.ErrCommentBlocked
	}
	if linkCount := len(linkRe.FindAllString(content, -1)); linkCount > 2 {
		return nil, errs.ErrCommentBlocked
	}

	if !s.perPostFreq.Allow(ip+"|"+strconv.FormatUint(uint64(post.ID), 10)) || !s.perDay.Allow(ip) {
		return nil, errs.ErrRateLimited
	}

	cm := &model.Comment{
		PostID: post.ID, ParentID: 0, Nickname: nickname, Email: email,
		Content: content, Status: model.CommentPending, IP: ip, UserAgent: ua,
	}
	if err := s.db.WithContext(ctx).Create(cm).Error; err != nil {
		return nil, err
	}
	return cm, nil
}

// ListPublic 前台评论树：顶级分页（时间正序），子回复挂在 children。
func (s *CommentService) ListPublic(ctx context.Context, slug string, page, size int) ([]CommentNode, int64, error) {
	var post model.Post
	if err := s.db.WithContext(ctx).
		Where("slug = ? AND status = ?", slug, model.StatusPublished).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errs.ErrNotFound
		}
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 20
	}

	var total int64
	q := s.db.WithContext(ctx).Model(&model.Comment{}).
		Where("post_id = ? AND status = ? AND parent_id = 0", post.ID, model.CommentApproved)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tops []model.Comment
	if err := q.Order("created_at ASC").
		Offset((page - 1) * size).Limit(size).Find(&tops).Error; err != nil {
		return nil, 0, err
	}
	if len(tops) == 0 {
		return []CommentNode{}, total, nil
	}

	ids := make([]uint, 0, len(tops))
	for _, t := range tops {
		ids = append(ids, t.ID)
	}
	var children []model.Comment
	if err := s.db.WithContext(ctx).
		Where("parent_id IN ? AND status = ?", ids, model.CommentApproved).
		Order("created_at ASC").Find(&children).Error; err != nil {
		return nil, 0, err
	}

	nodes := make([]CommentNode, 0, len(tops))
	for _, t := range tops {
		nodes = append(nodes, CommentNode{
			ID: t.ID, Nickname: t.Nickname, Content: t.Content,
			IsAdmin: t.IsAdmin, CreatedAt: t.CreatedAt, Children: []CommentNode{},
		})
	}
	for _, ch := range children {
		for i := range nodes {
			if nodes[i].ID == ch.ParentID {
				nodes[i].Children = append(nodes[i].Children, CommentNode{
					ID: ch.ID, Nickname: ch.Nickname, Content: ch.Content,
					IsAdmin: ch.IsAdmin, CreatedAt: ch.CreatedAt, Children: []CommentNode{},
				})
			}
		}
	}
	return nodes, total, nil
}

// ListAdmin 后台评论列表（联文章标题，按状态筛选，新到旧）。
func (s *CommentService) ListAdmin(ctx context.Context, page, size int, status *int8) ([]AdminCommentVO, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	q := s.db.WithContext(ctx).Model(&model.Comment{})
	if status != nil {
		q = q.Where("comments.status = ?", *status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []AdminCommentVO
	err := q.Select("comments.*, posts.title AS post_title, posts.slug AS post_slug").
		Joins("LEFT JOIN posts ON posts.id = comments.post_id").
		Order("comments.created_at DESC").
		Offset((page - 1) * size).Limit(size).Scan(&list).Error
	return list, total, err
}

// Review 审核：1=通过 / 2=驳回；随后重算文章 comment_count（仅 status=1 计数）。
func (s *CommentService) Review(ctx context.Context, id uint, status int8) error {
	if status != model.CommentApproved && status != model.CommentRejected {
		return errs.ErrParam
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cm model.Comment
		if err := tx.First(&cm, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if cm.Status == status {
			return nil
		}
		if err := tx.Model(&cm).Update("status", status).Error; err != nil {
			return err
		}
		return recountComments(tx, cm.PostID)
	})
}

// Reply 博主回复：挂到目标所在顶级评论下，is_admin=1 且自动通过。
func (s *CommentService) Reply(ctx context.Context, id, adminID uint, content string) (*model.Comment, error) {
	content = SanitizeCommentText(content)
	if n := len([]rune(content)); n < 1 || n > 500 {
		return nil, errs.New(40001, "回复内容长度需在 1-500 字符")
	}
	var admin model.Admin
	if err := s.db.WithContext(ctx).First(&admin, adminID).Error; err != nil {
		return nil, err
	}
	var cm *model.Comment
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target model.Comment
		if err := tx.First(&target, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		parentID := target.ID
		if target.ParentID != 0 { // 回复「回复」仍挂其顶级
			parentID = target.ParentID
		}
		cm = &model.Comment{
			PostID: target.PostID, ParentID: parentID, Nickname: admin.Nickname,
			Content: content, IsAdmin: true, Status: model.CommentApproved,
		}
		if err := tx.Create(cm).Error; err != nil {
			return err
		}
		return recountComments(tx, target.PostID)
	})
	if err != nil {
		return nil, err
	}
	return cm, nil
}

// Delete 软删（保留审计），随后重算计数。
func (s *CommentService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cm model.Comment
		if err := tx.First(&cm, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrNotFound
			}
			return err
		}
		if err := tx.Delete(&model.Comment{}, id).Error; err != nil {
			return err
		}
		return recountComments(tx, cm.PostID)
	})
}

// recountComments 重算 posts.comment_count（仅已通过且未软删）。
func recountComments(tx *gorm.DB, postID uint) error {
	return tx.Model(&model.Post{}).Where("id = ?", postID).
		UpdateColumn("comment_count", gorm.Expr(
			"(SELECT COUNT(*) FROM comments WHERE comments.post_id = ? AND comments.status = 1 AND comments.deleted_at IS NULL)",
			postID,
		)).Error
}
