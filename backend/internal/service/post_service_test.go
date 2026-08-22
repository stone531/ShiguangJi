package service

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"shiguangji/internal/model"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// 内存库跟随连接：连接池开新连接会拿到空库，限制单连接
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	err = db.AutoMigrate(
		&model.Post{}, &model.Category{}, &model.Tag{}, &model.PostTag{},
		&model.Comment{}, &model.Like{}, &model.SiteSetting{}, &model.Admin{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreatePostAutoSummary(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	p, err := svc.Create(context.Background(), &model.Post{
		Title: "hello", Slug: "hello-world", Content: strings.Repeat("字", 200), Status: model.StatusPublished,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(p.Summary)) != 120 {
		t.Fatalf("summary runes = %d, want 120", len([]rune(p.Summary)))
	}
}

func TestCreatePostDuplicateSlug(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	_, err := svc.Create(context.Background(), &model.Post{Title: "a", Slug: "dup", Content: "x"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(context.Background(), &model.Post{Title: "b", Slug: "dup", Content: "x"}, nil)
	if err == nil {
		t.Fatal("duplicate slug must fail")
	}
}

func TestPublishSetsPublishedAt(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	p, _ := svc.Create(context.Background(), &model.Post{Title: "a", Slug: "s1", Content: "x"}, nil)
	if err := svc.SetStatus(context.Background(), p.ID, model.StatusPublished); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.GetByID(context.Background(), p.ID)
	if got.PublishedAt == nil {
		t.Fatal("published_at must be set on first publish")
	}
}

func TestVideoPostRequiresWhitelistedDomain(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	_, err := svc.Create(context.Background(), &model.Post{
		Title: "v", Slug: "v1", Content: "x", Type: model.TypeVideo,
		VideoURL: "https://evil.example.com/embed/x",
	}, nil)
	if err == nil {
		t.Fatal("non-whitelisted video domain must be rejected")
	}
}

func mustTags(t *testing.T, db *gorm.DB, names ...string) []uint {
	t.Helper()
	var ids []uint
	for _, n := range names {
		tag := model.Tag{Name: n}
		if err := db.Create(&tag).Error; err != nil {
			t.Fatal(err)
		}
		ids = append(ids, tag.ID)
	}
	return ids
}

func TestCreateWithTags(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	tagIDs := mustTags(t, db, "go", "gin")
	p, err := svc.Create(context.Background(), &model.Post{
		Title: "t", Slug: "with-tags", Content: "x", Status: model.StatusPublished,
	}, tagIDs)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("tags = %d, want 2", len(got.Tags))
	}
}

func TestUpdatePostChangesTitle(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	tagIDs := mustTags(t, db, "a", "b", "c")
	p, _ := svc.Create(context.Background(), &model.Post{
		Title: "old", Slug: "old-slug", Content: "x", Status: model.StatusPublished,
	}, tagIDs[:2])
	newTags := mustTags(t, db, "d")
	err := svc.Update(context.Background(), p.ID, &model.Post{
		Title: "new", Slug: "old-slug", Content: "y", Status: model.StatusPublished,
	}, append(newTags, tagIDs[0]))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := svc.GetByID(context.Background(), p.ID)
	if got.Title != "new" {
		t.Fatalf("title = %q, want new", got.Title)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("tags = %d, want 2", len(got.Tags))
	}
}

func TestUpdatePostSlugConflict(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	a, _ := svc.Create(context.Background(), &model.Post{Title: "a", Slug: "slug-a", Content: "x"}, nil)
	b, _ := svc.Create(context.Background(), &model.Post{Title: "b", Slug: "slug-b", Content: "x"}, nil)
	err := svc.Update(context.Background(), b.ID, &model.Post{Title: "b", Slug: "slug-a", Content: "x"}, nil)
	if err == nil {
		t.Fatal("slug conflict must fail")
	}
	_ = a
}

func TestListOnlyPublished(t *testing.T) {
	db := newTestDB(t)
	svc := NewPostService(db)
	svc.Create(context.Background(), &model.Post{Title: "pub", Slug: "p1", Content: "x", Status: model.StatusPublished}, nil)
	svc.Create(context.Background(), &model.Post{Title: "draft", Slug: "d1", Content: "x", Status: model.StatusDraft}, nil)
	list, total, err := svc.ListPublic(context.Background(), PublicListFilter{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Slug != "p1" {
		t.Fatalf("total=%d list=%v", total, list)
	}
}
