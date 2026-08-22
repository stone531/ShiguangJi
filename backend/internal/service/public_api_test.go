package service

import (
	"context"
	"testing"
	"time"

	"shiguangji/internal/model"
)

func mustCreate(t *testing.T, svc *PostService, p *model.Post, tagIDs []uint) *model.Post {
	t.Helper()
	created, err := svc.Create(context.Background(), p, tagIDs)
	if err != nil {
		t.Fatalf("create %q: %v", p.Slug, err)
	}
	return created
}

func TestSiteGetPublicDefaultsAndStats(t *testing.T) {
	db := newTestDB(t)
	svc := NewSiteService(db)

	// 空配置表：全部默认值兜底
	vo, err := svc.GetPublic(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if vo.SiteName != "时光集" || vo.PageSize != 10 || !vo.CommentEnabled {
		t.Fatalf("defaults wrong: %+v", vo)
	}
	if vo.SocialLinks == nil || len(vo.SocialLinks) != 0 {
		t.Fatalf("social_links = %v, want empty slice", vo.SocialLinks)
	}

	// 自定义配置覆盖 + 统计仅计已发布
	db.Create(&model.SiteSetting{SettingKey: "site_name", SettingValue: "我的站"})
	db.Create(&model.SiteSetting{SettingKey: "page_size", SettingValue: "20"})
	db.Create(&model.SiteSetting{SettingKey: "comment_enabled", SettingValue: "false"})
	db.Create(&model.SiteSetting{SettingKey: "social_links", SettingValue: `[{"name":"GitHub","url":"https://github.com","icon":"github"}]`})
	ps := NewPostService(db)
	pub := mustCreate(t, ps, &model.Post{Title: "p", Slug: "p", Content: "x", Status: model.StatusPublished}, nil)
	db.Model(&pub).UpdateColumn("views", 7)
	mustCreate(t, ps, &model.Post{Title: "d", Slug: "d", Content: "x", Status: model.StatusDraft}, nil)

	vo, err = svc.GetPublic(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if vo.SiteName != "我的站" || vo.PageSize != 20 || vo.CommentEnabled {
		t.Fatalf("overrides wrong: %+v", vo)
	}
	if len(vo.SocialLinks) != 1 || vo.SocialLinks[0].Name != "GitHub" {
		t.Fatalf("social_links = %+v", vo.SocialLinks)
	}
	if vo.PostCount != 1 || vo.ViewCount != 7 || vo.LikeCount != 0 {
		t.Fatalf("stats = post:%d view:%d like:%d", vo.PostCount, vo.ViewCount, vo.LikeCount)
	}
}

func TestTaxonomyCountsOnlyPublished(t *testing.T) {
	db := newTestDB(t)
	catA := model.Category{Name: "tech"}
	catB := model.Category{Name: "life"}
	db.Create(&catA)
	db.Create(&catB)
	tagGo := model.Tag{Name: "go"}
	tagDead := model.Tag{Name: "dead"}
	db.Create(&tagGo)
	db.Create(&tagDead)

	ps := NewPostService(db)
	mustCreate(t, ps, &model.Post{Title: "a1", Slug: "a1", Content: "x", Status: model.StatusPublished, CategoryID: &catA.ID}, []uint{tagGo.ID})
	mustCreate(t, ps, &model.Post{Title: "a2", Slug: "a2", Content: "x", Status: model.StatusPublished, CategoryID: &catA.ID}, nil)
	mustCreate(t, ps, &model.Post{Title: "a3", Slug: "a3", Content: "x", Status: model.StatusDraft, CategoryID: &catA.ID}, nil) // 草稿不计
	mustCreate(t, ps, &model.Post{Title: "b1", Slug: "b1", Content: "x", Status: model.StatusPublished, CategoryID: &catB.ID}, nil)
	mustCreate(t, ps, &model.Post{Title: "dead", Slug: "dead", Content: "x", Status: model.StatusDraft}, []uint{tagDead.ID}) // 草稿的标签不出现

	svc := NewTaxonomyService(db)
	cats, err := svc.ListCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 2 || cats[0].Name != "tech" || cats[0].PostCount != 2 || cats[1].PostCount != 1 {
		t.Fatalf("categories = %+v", cats)
	}
	tags, err := svc.ListTags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0].Name != "go" || tags[0].PostCount != 1 {
		t.Fatalf("tags = %+v", tags)
	}
}

func TestListPublicFilters(t *testing.T) {
	db := newTestDB(t)
	cat := model.Category{Name: "tech"}
	db.Create(&cat)
	tagGo := model.Tag{Name: "golang"}
	db.Create(&tagGo)

	ps := NewPostService(db)
	video := model.TypeVideo
	mustCreate(t, ps, &model.Post{Title: "go 视频", Slug: "v", Content: "x", Status: model.StatusPublished, Type: model.TypeVideo, VideoURL: "https://player.bilibili.com/player.html?bvid=xxx", IsFeatured: true}, []uint{tagGo.ID})
	mustCreate(t, ps, &model.Post{Title: "生活随笔", Slug: "life", Content: "x", Status: model.StatusPublished, CategoryID: &cat.ID}, nil)
	mustCreate(t, ps, &model.Post{Title: "草稿 go", Slug: "draft", Content: "x", Status: model.StatusDraft}, []uint{tagGo.ID})

	ctx := context.Background()
	featured := true

	if _, total, _ := ps.ListPublic(ctx, PublicListFilter{Page: 1, Size: 10, Type: &video}); total != 1 {
		t.Fatalf("type filter total = %d, want 1", total)
	}
	if _, total, _ := ps.ListPublic(ctx, PublicListFilter{Page: 1, Size: 10, Category: "tech"}); total != 1 {
		t.Fatalf("category filter total = %d, want 1", total)
	}
	if _, total, _ := ps.ListPublic(ctx, PublicListFilter{Page: 1, Size: 10, Tag: "golang"}); total != 1 {
		t.Fatalf("tag filter total = %d, want 1", total)
	}
	if _, total, _ := ps.ListPublic(ctx, PublicListFilter{Page: 1, Size: 10, Featured: &featured}); total != 1 {
		t.Fatalf("featured filter total = %d, want 1", total)
	}
	// keyword 命中标题与标签名，但草稿不可见
	list, total, _ := ps.ListPublic(ctx, PublicListFilter{Page: 1, Size: 10, Keyword: "go"})
	if total != 1 || len(list) != 1 || list[0].Slug != "v" {
		t.Fatalf("keyword filter = total:%d list:%+v", total, list)
	}
}

func TestArchiveGroupsByMonth(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	t1 := time.Date(2026, 7, 1, 10, 0, 0, 0, time.Local)
	t2 := time.Date(2026, 8, 2, 10, 0, 0, 0, time.Local)
	t3 := time.Date(2026, 8, 5, 10, 0, 0, 0, time.Local)
	mustCreate(t, ps, &model.Post{Title: "old", Slug: "old", Content: "x", Status: model.StatusPublished, PublishedAt: &t1}, nil)
	mustCreate(t, ps, &model.Post{Title: "mid", Slug: "mid", Content: "x", Status: model.StatusPublished, PublishedAt: &t2}, nil)
	mustCreate(t, ps, &model.Post{Title: "new", Slug: "new", Content: "x", Status: model.StatusPublished, PublishedAt: &t3}, nil)
	mustCreate(t, ps, &model.Post{Title: "draft", Slug: "dr", Content: "x", Status: model.StatusDraft}, nil) // 无发布时间，不出现

	groups, err := ps.Archive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(groups))
	}
	if groups[0].Year != 2026 || groups[0].Month != 8 || len(groups[0].Posts) != 2 {
		t.Fatalf("first group = %+v", groups[0])
	}
	if groups[0].Posts[0].Slug != "new" {
		t.Fatalf("group not ordered desc: %+v", groups[0].Posts)
	}
	if groups[1].Month != 7 || len(groups[1].Posts) != 1 {
		t.Fatalf("second group = %+v", groups[1])
	}
}

func TestGetBySlugPublicNeighbors(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	t1 := time.Date(2026, 7, 1, 10, 0, 0, 0, time.Local)
	t2 := time.Date(2026, 8, 1, 10, 0, 0, 0, time.Local)
	t3 := time.Date(2026, 8, 9, 10, 0, 0, 0, time.Local)
	t4 := time.Date(2026, 8, 20, 10, 0, 0, 0, time.Local)
	mustCreate(t, ps, &model.Post{Title: "p1", Slug: "p1", Content: "x", Status: model.StatusPublished, PublishedAt: &t1}, nil)
	mustCreate(t, ps, &model.Post{Title: "p2", Slug: "p2", Content: "x", Status: model.StatusPublished, PublishedAt: &t2}, nil)
	mustCreate(t, ps, &model.Post{Title: "p3-draft", Slug: "p3d", Content: "x", Status: model.StatusDraft, PublishedAt: &t3}, nil) // 夹在中间但草稿
	mustCreate(t, ps, &model.Post{Title: "p4", Slug: "p4", Content: "x", Status: model.StatusPublished, PublishedAt: &t4}, nil)

	p, prev, next, err := ps.GetBySlugPublic(context.Background(), "p2")
	if err != nil {
		t.Fatal(err)
	}
	if p.Views != 1 {
		t.Fatalf("views = %d, want 1", p.Views)
	}
	if prev == nil || prev.Slug != "p1" {
		t.Fatalf("prev = %+v, want p1", prev)
	}
	if next == nil || next.Slug != "p4" {
		t.Fatalf("next = %+v, want p4 (draft skipped)", next)
	}
}
