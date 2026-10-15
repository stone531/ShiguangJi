package service

import (
	"context"
	"testing"

	"shiguangji/internal/model"
)

func TestSiteAdminGetSaveRoundTrip(t *testing.T) {
	db := newTestDB(t)
	svc := NewSiteService(db)
	ctx := context.Background()

	// 初始：默认值 + 敏感词空
	vo, err := svc.AdminGet(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if vo.SiteName != "时光集" || vo.PageSize != 10 || !vo.CommentEnabled || vo.SensitiveWords != "" {
		t.Fatalf("initial admin vo = %+v", vo)
	}

	// 保存全量配置
	err = svc.Save(ctx, &SiteSaveInput{
		SiteName: "我的站", SiteDesc: "描述", Avatar: "/uploads/a.png",
		SocialLinks: []SocialLink{{Name: "GitHub", URL: "https://github.com", Icon: "github"}},
		PageSize:    20, CommentEnabled: false,
		SensitiveWords: "傻逼\n垃圾",
	})
	if err != nil {
		t.Fatal(err)
	}
	vo, err = svc.AdminGet(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if vo.SiteName != "我的站" || vo.PageSize != 20 || vo.CommentEnabled ||
		len(vo.SocialLinks) != 1 || vo.SensitiveWords != "傻逼\n垃圾" {
		t.Fatalf("after save vo = %+v", vo)
	}
	// 公开端点读到保存值，且永不暴露敏感词
	pub, err := svc.GetPublic(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if pub.SiteName != "我的站" || pub.PageSize != 20 || pub.CommentEnabled {
		t.Fatalf("public vo = %+v", pub)
	}

	// 校验：站名空 / 分页越界
	if err := svc.Save(ctx, &SiteSaveInput{SiteName: "", PageSize: 10}); err == nil {
		t.Fatal("empty site_name should fail")
	}
	if err := svc.Save(ctx, &SiteSaveInput{SiteName: "x", PageSize: 99}); err == nil {
		t.Fatal("page_size 99 should fail")
	}

	// 敏感词改动即时影响评论过滤
	ps := NewPostService(db)
	mustCreate(t, ps, &model.Post{Title: "p", Slug: "p", Content: "x", Status: model.StatusPublished}, nil)
	commentSvc := NewCommentService(db)
	defer commentSvc.Close()
	if _, err := commentSvc.Create(ctx, "p", "甲", "", "你就是个垃圾", "1.1.1.1", "ua"); err == nil {
		t.Fatal("new sensitive word should block comment immediately")
	}
}

func TestAdminProfile(t *testing.T) {
	db := newTestDB(t)
	admin := model.Admin{Username: "admin", PasswordHash: "x", Nickname: "旧名", Avatar: ""}
	db.Create(&admin)

	svc := NewAuthService(db, "secret", 0)
	ctx := context.Background()

	p, err := svc.GetProfile(ctx, admin.ID)
	if err != nil || p.Username != "admin" || p.Nickname != "旧名" {
		t.Fatalf("profile = %+v err = %v", p, err)
	}
	if err := svc.UpdateProfile(ctx, admin.ID, "新名", "/uploads/me.png"); err != nil {
		t.Fatal(err)
	}
	p, _ = svc.GetProfile(ctx, admin.ID)
	if p.Nickname != "新名" || p.Avatar != "/uploads/me.png" {
		t.Fatalf("after update = %+v", p)
	}
	// 昵称越界
	if err := svc.UpdateProfile(ctx, admin.ID, "", ""); err == nil {
		t.Fatal("empty nickname should fail")
	}
	// 不存在的管理员
	if _, err := svc.GetProfile(ctx, 999); err == nil {
		t.Fatal("missing admin should fail")
	}
}

func TestStatsSixMetrics(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	p1 := mustCreate(t, ps, &model.Post{Title: "p1", Slug: "p1", Content: "x", Status: model.StatusPublished}, nil)
	p2 := mustCreate(t, ps, &model.Post{Title: "p2", Slug: "p2", Content: "x", Status: model.StatusDraft}, nil)
	db.Model(&p1).UpdateColumns(map[string]interface{}{"views": 100, "like_count": 5})
	db.Model(&p2).UpdateColumn("views", 30)
	// 评论：1 已通过 + 2 待审
	db.Create(&model.Comment{PostID: p1.ID, Nickname: "a", Content: "x", Status: model.CommentApproved})
	db.Create(&model.Comment{PostID: p1.ID, Nickname: "b", Content: "x", Status: model.CommentPending})
	db.Create(&model.Comment{PostID: p2.ID, Nickname: "c", Content: "x", Status: model.CommentPending})

	svc := NewStatsService(db)
	vo, err := svc.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if vo.PostTotal != 2 || vo.PostPublished != 1 {
		t.Fatalf("post stats = %+v", vo)
	}
	if vo.CommentTotal != 3 || vo.CommentPending != 2 {
		t.Fatalf("comment stats = %+v", vo)
	}
	if vo.ViewTotal != 130 || vo.LikeTotal != 5 {
		t.Fatalf("view/like stats = %+v", vo)
	}
}

func TestTaxonomyAdminCRUDAndGuards(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	svc := NewTaxonomyService(db)
	ctx := context.Background()

	// 分类新建/更新/重名
	cat, err := svc.SaveCategory(ctx, 0, "技术", 1)
	if err != nil || cat.ID == 0 {
		t.Fatalf("create category: %+v %v", cat, err)
	}
	if _, err := svc.SaveCategory(ctx, 0, "技术", 2); err == nil {
		t.Fatal("duplicate category name should fail")
	}
	if _, err := svc.SaveCategory(ctx, cat.ID, "Tech", 3); err != nil {
		t.Fatal(err)
	}
	var got model.Category
	db.First(&got, cat.ID)
	if got.Name != "Tech" || got.Sort != 3 {
		t.Fatalf("updated = %+v", got)
	}

	// 删除守护：其下有文章（含草稿）→ 拒绝
	draft := mustCreate(t, ps, &model.Post{Title: "d", Slug: "d", Content: "x", Status: model.StatusDraft, CategoryID: &cat.ID}, nil)
	if err := svc.DeleteCategory(ctx, cat.ID); err == nil {
		t.Fatal("category with posts should refuse delete")
	}
	db.Model(&draft).Update("category_id", nil)
	if err := svc.DeleteCategory(ctx, cat.ID); err != nil {
		t.Fatalf("empty category delete: %v", err)
	}

	// 标签：新建/删除守护/合并
	tagGo, _ := svc.SaveTag(ctx, 0, "go")
	tagGolang, _ := svc.SaveTag(ctx, 0, "golang")
	published := mustCreate(t, ps, &model.Post{Title: "p", Slug: "p", Content: "x", Status: model.StatusPublished}, []uint{tagGo.ID, tagGolang.ID})

	// 同时挂两个标签的文章：合并后只留 target 一条关联
	if err := svc.MergeTag(ctx, tagGo.ID, tagGolang.ID); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&model.PostTag{}).Where("post_id = ? AND tag_id = ?", published.ID, tagGolang.ID).Count(&n)
	if n != 1 {
		t.Fatalf("after merge target rows = %d, want 1", n)
	}
	var dead int64
	db.Model(&model.Tag{}).Where("id = ?", tagGo.ID).Count(&dead)
	if dead != 0 {
		t.Fatal("source tag should be deleted after merge")
	}
	// 合并到自身 → 拒绝
	if err := svc.MergeTag(ctx, tagGolang.ID, tagGolang.ID); err == nil {
		t.Fatal("merge to self should fail")
	}
	// 被引用的标签不可直接删
	if err := svc.DeleteTag(ctx, tagGolang.ID); err == nil {
		t.Fatal("in-use tag delete should refuse")
	}
}
