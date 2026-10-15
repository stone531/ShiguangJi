package service

import (
	"context"
	"strings"
	"testing"

	"shiguangji/internal/errs"
	"shiguangji/internal/model"
)

func TestSanitizeCommentText(t *testing.T) {
	cases := []struct{ in, want string }{
		{"正常评论", "正常评论"},
		{"零\u200b宽\u200c字\u200d符\ufeff", "零宽字符"}, // 零宽剔除（含 BOM）
		{"带控\x00制\x07字符", "带控制字符"},               // 控制字符剔除
		{"软­连字⁠符", "软连字符"},                       // Cf 格式字符剔除（U+00AD/U+2060）
		{"多   余  空白\t\t压缩", "多 余 空白 压缩"},         // 连续空白压缩
		{"行1\n\n\n\n行2", "行1\n\n行2"},             // 连续换行 ≤2
		{"  首尾空白  \n  ", "首尾空白"},                 // TrimSpace
		{"emoji 😊 保留", "emoji 😊 保留"},             //
	}
	for _, c := range cases {
		if got := SanitizeCommentText(c.in); got != c.want {
			t.Errorf("SanitizeCommentText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSensitiveFilterObfuscation(t *testing.T) {
	f := NewSensitiveFilter("傻逼\n煞笔\nfuck")
	cases := []struct {
		in   string
		want bool
	}{
		{"你真傻逼", true},       // 直接命中
		{"你真傻 逼", true},      // 插空格
		{"你真傻.逼", true},      // 插标点
		{"你真傻⁠逼", true},      // 插词连接符 U+2060（Cf 格式字符）
		{"你真傻­逼", true},      // 插软连字符 U+00AD（Cf 格式字符）
		{"你真傻Ｘ逼不该命中", false}, // 中间多了字符，不构成连续命中
		{"全角ＦＵＣＫ", true},     // 全角 + 大写
		{"f u c k", true},    // 逐字母插空格
		{"正常内容", false},
	}
	for _, c := range cases {
		if got := f.Hit(c.in); got != c.want {
			t.Errorf("Hit(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestCommentCreatePipeline(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	mustCreate(t, ps, &model.Post{Title: "p", Slug: "p", Content: "x", Status: model.StatusPublished}, nil)
	mustCreate(t, ps, &model.Post{Title: "draft", Slug: "draft", Content: "x", Status: model.StatusDraft}, nil)
	db.Create(&model.SiteSetting{SettingKey: "sensitive_words", SettingValue: "傻逼\n煞笔"})

	svc := NewCommentService(db)
	defer svc.Close()
	ctx := context.Background()

	// 草稿文章不可评论
	if _, err := svc.Create(ctx, "draft", "小明", "", "好文", "1.1.1.1", "ua"); err != errs.ErrNotFound {
		t.Fatalf("draft err = %v, want ErrNotFound", err)
	}
	// 长度越界（昵称 >20 rune）
	if _, err := svc.Create(ctx, "p", strings.Repeat("名", 21), "", "内容", "1.1.1.2", "ua"); err == nil || err.(*errs.BizError).Code != 40001 {
		t.Fatalf("long nickname err = %v, want 40001", err)
	}
	// 内容 >500 rune
	if _, err := svc.Create(ctx, "p", "小明", "", strings.Repeat("字", 501), "1.1.1.3", "ua"); err == nil || err.(*errs.BizError).Code != 40001 {
		t.Fatalf("long content err = %v, want 40001", err)
	}
	// 邮箱不含 @
	if _, err := svc.Create(ctx, "p", "小明", "not-an-email", "内容", "1.1.1.4", "ua"); err == nil || err.(*errs.BizError).Code != 40001 {
		t.Fatalf("bad email err = %v, want 40001", err)
	}
	// 变形敏感词 → 40006
	if _, err := svc.Create(ctx, "p", "小明", "", "你真傻.逼啊", "1.1.1.5", "ua"); err != errs.ErrCommentBlocked {
		t.Fatalf("sensitive err = %v, want ErrCommentBlocked", err)
	}
	// 链接 >2 → 40006
	if _, err := svc.Create(ctx, "p", "小明", "", "看 http://a.com 和 https://b.com 还有 http://c.com", "1.1.1.6", "ua"); err != errs.ErrCommentBlocked {
		t.Fatalf("links err = %v, want ErrCommentBlocked", err)
	}
	// 正常落库：status=0 待审、净化生效
	cm, err := svc.Create(ctx, "p", "  小明  ", "", "零​宽\n\n\n\n多行", "2.2.2.2", "ua")
	if err != nil {
		t.Fatal(err)
	}
	if cm.Status != model.CommentPending {
		t.Fatalf("status = %d, want pending", cm.Status)
	}
	if cm.Nickname != "小明" || cm.Content != "零宽\n\n多行" {
		t.Fatalf("sanitized = %q / %q", cm.Nickname, cm.Content)
	}
	// 同 IP 同文章 1 条/分钟：第二条被限流
	if _, err := svc.Create(ctx, "p", "小明", "", "再来一条", "2.2.2.2", "ua"); err != errs.ErrRateLimited {
		t.Fatalf("rate limit err = %v, want ErrRateLimited", err)
	}
	// 不同 IP 不受影响
	if _, err := svc.Create(ctx, "p", "小红", "", "我也是", "3.3.3.3", "ua"); err != nil {
		t.Fatal(err)
	}
	// 同 IP 每日 10 条：同一 IP 对 10 篇不同文章各发一条（绕开每文章 1 条/分钟），
	// 第 11 条被日限额拦截
	for i := 0; i < 10; i++ {
		slug := "daily-" + string(rune('a'+i))
		mustCreate(t, ps, &model.Post{Title: slug, Slug: slug, Content: "x", Status: model.StatusPublished}, nil)
		if _, err := svc.Create(ctx, slug, "路人", "", "第"+string(rune('A'+i))+"条", "4.4.4.4", "ua"); err != nil {
			t.Fatalf("daily warmup %d: %v", i, err)
		}
	}
	mustCreate(t, ps, &model.Post{Title: "x", Slug: "daily-x", Content: "x", Status: model.StatusPublished}, nil)
	if _, err := svc.Create(ctx, "daily-x", "路人", "", "超限", "4.4.4.4", "ua"); err != errs.ErrRateLimited {
		t.Fatalf("daily limit err = %v, want ErrRateLimited", err)
	}
}

func TestCommentTreeAndAdminList(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	post := mustCreate(t, ps, &model.Post{Title: "标题", Slug: "p", Content: "x", Status: model.StatusPublished}, nil)

	// 直接落库：顶级两条已通过、一条待审；子回复一条已通过、一条被驳回
	top1 := model.Comment{PostID: post.ID, Nickname: "甲", Content: "顶1", Status: model.CommentApproved}
	top2 := model.Comment{PostID: post.ID, Nickname: "乙", Content: "顶2", Status: model.CommentApproved}
	pending := model.Comment{PostID: post.ID, Nickname: "丙", Content: "待审", Status: model.CommentPending}
	db.Create(&top1)
	db.Create(&top2)
	db.Create(&pending)
	db.Create(&model.Comment{PostID: post.ID, ParentID: top1.ID, Nickname: "丁", Content: "子回复", Status: model.CommentApproved})
	db.Create(&model.Comment{PostID: post.ID, ParentID: top1.ID, Nickname: "戊", Content: "被驳回的子回复", Status: model.CommentRejected})

	svc := NewCommentService(db)
	defer svc.Close()
	ctx := context.Background()

	nodes, total, err := svc.ListPublic(ctx, "p", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(nodes) != 2 {
		t.Fatalf("public tree total = %d len = %d, want 2/2（待审不出现）", total, len(nodes))
	}
	if nodes[0].Nickname != "甲" || len(nodes[0].Children) != 1 || nodes[0].Children[0].Content != "子回复" {
		t.Fatalf("tree = %+v", nodes[0])
	}
	if len(nodes[1].Children) != 0 {
		t.Fatalf("top2 children = %+v, want empty", nodes[1].Children)
	}

	// 后台列表：全量 5 条、联表带出文章标题；状态筛选
	list, total, err := svc.ListAdmin(ctx, 1, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 || len(list) != 5 {
		t.Fatalf("admin total = %d len = %d, want 5", total, len(list))
	}
	if list[0].PostTitle != "标题" || list[0].PostSlug != "p" {
		t.Fatalf("joined post = %q/%q", list[0].PostTitle, list[0].PostSlug)
	}
	pendingStatus := int8(model.CommentPending)
	_, total, _ = svc.ListAdmin(ctx, 1, 10, &pendingStatus)
	if total != 1 {
		t.Fatalf("pending filter total = %d, want 1", total)
	}
}

func TestCommentReviewSyncsCount(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	post := mustCreate(t, ps, &model.Post{Title: "p", Slug: "p", Content: "x", Status: model.StatusPublished}, nil)

	cm := model.Comment{PostID: post.ID, Nickname: "甲", Content: "待审", Status: model.CommentPending}
	db.Create(&cm)

	svc := NewCommentService(db)
	defer svc.Close()
	ctx := context.Background()

	// 通过 → comment_count=1
	if err := svc.Review(ctx, cm.ID, model.CommentApproved); err != nil {
		t.Fatal(err)
	}
	var p model.Post
	db.First(&p, post.ID)
	if p.CommentCount != 1 {
		t.Fatalf("comment_count = %d, want 1 after approve", p.CommentCount)
	}
	// 重复审核幂等
	if err := svc.Review(ctx, cm.ID, model.CommentApproved); err != nil {
		t.Fatal(err)
	}
	db.First(&p, post.ID)
	if p.CommentCount != 1 {
		t.Fatalf("comment_count = %d after idempotent review", p.CommentCount)
	}
	// 驳回 → 归零
	if err := svc.Review(ctx, cm.ID, model.CommentRejected); err != nil {
		t.Fatal(err)
	}
	db.First(&p, post.ID)
	if p.CommentCount != 0 {
		t.Fatalf("comment_count = %d, want 0 after reject", p.CommentCount)
	}
	// 非法状态值
	if err := svc.Review(ctx, cm.ID, 9); err != errs.ErrParam {
		t.Fatalf("invalid status err = %v, want ErrParam", err)
	}
}

func TestCommentReplyAttachesToTopLevel(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	post := mustCreate(t, ps, &model.Post{Title: "p", Slug: "p", Content: "x", Status: model.StatusPublished}, nil)
	admin := model.Admin{Username: "admin", PasswordHash: "x", Nickname: "博主"}
	db.Create(&admin)

	top := model.Comment{PostID: post.ID, Nickname: "甲", Content: "顶", Status: model.CommentApproved}
	db.Create(&top)
	child := model.Comment{PostID: post.ID, ParentID: top.ID, Nickname: "乙", Content: "子", Status: model.CommentApproved}
	db.Create(&child)

	svc := NewCommentService(db)
	defer svc.Close()
	ctx := context.Background()

	// 回复「子回复」→ 仍挂顶级 top 下
	reply, err := svc.Reply(ctx, child.ID, admin.ID, "博主回复")
	if err != nil {
		t.Fatal(err)
	}
	if reply.ParentID != top.ID {
		t.Fatalf("reply parent_id = %d, want top %d", reply.ParentID, top.ID)
	}
	if !reply.IsAdmin || reply.Nickname != "博主" || reply.Status != model.CommentApproved {
		t.Fatalf("reply = %+v", reply)
	}
	// 回复直接通过：顶级 + 子 + 回复 = 3 条已通过
	var p model.Post
	db.First(&p, post.ID)
	if p.CommentCount != 3 {
		t.Fatalf("comment_count = %d, want 3", p.CommentCount)
	}
	// 软删回复后回落
	if err := svc.Delete(ctx, reply.ID); err != nil {
		t.Fatal(err)
	}
	db.First(&p, post.ID)
	if p.CommentCount != 2 {
		t.Fatalf("comment_count = %d after delete, want 2", p.CommentCount)
	}
}
