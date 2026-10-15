package service

import (
	"context"
	"encoding/json"
	"strconv"

	"gorm.io/gorm"

	"shiguangji/internal/errs"
	"shiguangji/internal/model"
)

// SocialLink 社交链接（site_settings.social_links JSON 项，spec §4.8）。
type SocialLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Icon string `json:"icon"`
}

// SitePublic 前台站点信息；刻意不含 sensitive_words 等敏感配置。
type SitePublic struct {
	SiteName       string       `json:"site_name"`
	SiteDesc       string       `json:"site_desc"`
	Avatar         string       `json:"avatar"`
	WechatQR       string       `json:"wechat_qr"`
	AlipayQR       string       `json:"alipay_qr"`
	DonateText     string       `json:"donate_text"`
	SocialLinks    []SocialLink `json:"social_links"`
	PageSize       int          `json:"page_size"`
	CommentEnabled bool         `json:"comment_enabled"`
	PostCount      int64        `json:"post_count"`
	ViewCount      int64        `json:"view_count"`
	LikeCount      int64        `json:"like_count"`
}

// SiteService 站点设置业务逻辑。
type SiteService struct{ db *gorm.DB }

// NewSiteService 构造函数。
func NewSiteService(db *gorm.DB) *SiteService { return &SiteService{db: db} }

// GetPublic 读全部配置键（缺失以 §4.8 默认值兜底）并附站点统计（仅已发布）。
func (s *SiteService) GetPublic(ctx context.Context) (*SitePublic, error) {
	var rows []model.SiteSetting
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	kv := make(map[string]string, len(rows))
	for _, r := range rows {
		kv[r.SettingKey] = r.SettingValue
	}

	vo := &SitePublic{
		SiteName:       kvDefault(kv, "site_name", "时光集"),
		SiteDesc:       kvDefault(kv, "site_desc", "记录技术与生活"),
		Avatar:         kv["avatar"],
		WechatQR:       kv["wechat_qr"],
		AlipayQR:       kv["alipay_qr"],
		DonateText:     kvDefault(kv, "donate_text", "写得不错，请我喝杯咖啡 ☕"),
		SocialLinks:    parseSocialLinks(kv["social_links"]),
		PageSize:       10,
		CommentEnabled: true,
	}
	if n, err := strconv.Atoi(kvDefault(kv, "page_size", "10")); err == nil && n > 0 && n <= 50 {
		vo.PageSize = n
	}
	if v, ok := kv["comment_enabled"]; ok {
		vo.CommentEnabled = v != "false"
	}

	// 统计扫进独立结构体：直接 Scan 进 vo 会被 GORM 把 SocialLinks 当作关联解析
	var stats struct {
		PostCount int64
		ViewCount int64
		LikeCount int64
	}
	err := s.db.WithContext(ctx).Model(&model.Post{}).
		Where("status = ?", model.StatusPublished).
		Select("COUNT(*) AS post_count, COALESCE(SUM(views),0) AS view_count, COALESCE(SUM(like_count),0) AS like_count").
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	vo.PostCount, vo.ViewCount, vo.LikeCount = stats.PostCount, stats.ViewCount, stats.LikeCount
	return vo, nil
}

// kvDefault 取值，空串或缺失时回默认。
func kvDefault(kv map[string]string, key, def string) string {
	if v, ok := kv[key]; ok && v != "" {
		return v
	}
	return def
}

// parseSocialLinks 解析 JSON；空值/坏值一律空切片（前台 v-for 安全）。
func parseSocialLinks(raw string) []SocialLink {
	if raw == "" {
		return []SocialLink{}
	}
	var links []SocialLink
	if err := json.Unmarshal([]byte(raw), &links); err != nil || links == nil {
		return []SocialLink{}
	}
	return links
}

// SiteAdmin 后台站点设置 VO：比公开版多 sensitive_words。
type SiteAdmin struct {
	SiteName       string       `json:"site_name"`
	SiteDesc       string       `json:"site_desc"`
	Avatar         string       `json:"avatar"`
	WechatQR       string       `json:"wechat_qr"`
	AlipayQR       string       `json:"alipay_qr"`
	DonateText     string       `json:"donate_text"`
	SocialLinks    []SocialLink `json:"social_links"`
	PageSize       int          `json:"page_size"`
	CommentEnabled bool         `json:"comment_enabled"`
	SensitiveWords string       `json:"sensitive_words"`
}

// AdminGet 后台读取全部配置（含敏感词表），缺失键以默认值兜底。
func (s *SiteService) AdminGet(ctx context.Context) (*SiteAdmin, error) {
	var rows []model.SiteSetting
	if err := s.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	kv := make(map[string]string, len(rows))
	for _, r := range rows {
		kv[r.SettingKey] = r.SettingValue
	}
	vo := &SiteAdmin{
		SiteName:       kvDefault(kv, "site_name", "时光集"),
		SiteDesc:       kvDefault(kv, "site_desc", "记录技术与生活"),
		Avatar:         kv["avatar"],
		WechatQR:       kv["wechat_qr"],
		AlipayQR:       kv["alipay_qr"],
		DonateText:     kvDefault(kv, "donate_text", "写得不错，请我喝杯咖啡 ☕"),
		SocialLinks:    parseSocialLinks(kv["social_links"]),
		PageSize:       10,
		CommentEnabled: true,
		SensitiveWords: kv["sensitive_words"],
	}
	if n, err := strconv.Atoi(kvDefault(kv, "page_size", "10")); err == nil && n > 0 && n <= 50 {
		vo.PageSize = n
	}
	if v, ok := kv["comment_enabled"]; ok {
		vo.CommentEnabled = v != "false"
	}
	return vo, nil
}

// SiteSaveInput 保存入参；SocialLinks 由 handler 反序列化后传入。
type SiteSaveInput struct {
	SiteName       string
	SiteDesc       string
	Avatar         string
	WechatQR       string
	AlipayQR       string
	DonateText     string
	SocialLinks    []SocialLink
	PageSize       int
	CommentEnabled bool
	SensitiveWords string
}

// Save 校验后事务 upsert 全部键（FirstOrCreate+Assign：存在即更新，缺失即插入）。
func (s *SiteService) Save(ctx context.Context, in *SiteSaveInput) error {
	if in.SiteName == "" {
		return errs.New(40001, "站点名称不能为空")
	}
	if in.PageSize < 1 || in.PageSize > 50 {
		return errs.New(40001, "每页条数需在 1-50")
	}
	links, err := json.Marshal(in.SocialLinks)
	if err != nil {
		return err
	}
	kv := map[string]string{
		"site_name":       in.SiteName,
		"site_desc":       in.SiteDesc,
		"avatar":          in.Avatar,
		"wechat_qr":       in.WechatQR,
		"alipay_qr":       in.AlipayQR,
		"donate_text":     in.DonateText,
		"social_links":    string(links),
		"page_size":       strconv.Itoa(in.PageSize),
		"comment_enabled": strconv.FormatBool(in.CommentEnabled),
		"sensitive_words": in.SensitiveWords,
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for k, v := range kv {
			row := model.SiteSetting{SettingKey: k, SettingValue: v}
			if err := tx.Where("setting_key = ?", k).
				Assign("setting_value", v).
				FirstOrCreate(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
