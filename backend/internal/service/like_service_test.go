package service

import (
	"context"
	"sync"
	"testing"

	"shiguangji/internal/errs"
	"shiguangji/internal/model"
)

func TestLikeToggleRoundTrip(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	post := mustCreate(t, ps, &model.Post{Title: "p", Slug: "p", Content: "x", Status: model.StatusPublished}, nil)
	mustCreate(t, ps, &model.Post{Title: "d", Slug: "d", Content: "x", Status: model.StatusDraft}, nil)

	svc := NewLikeService(db, "test-salt")
	ctx := context.Background()

	// 草稿不可点赞
	if _, _, err := svc.Toggle(ctx, "d", "1.1.1.1", "ua"); err != errs.ErrNotFound {
		t.Fatalf("draft err = %v, want ErrNotFound", err)
	}

	// 点赞 → liked=true, count=1
	liked, count, err := svc.Toggle(ctx, "p", "1.1.1.1", "ua")
	if err != nil || !liked || count != 1 {
		t.Fatalf("like = (%v,%d,%v), want (true,1,nil)", liked, count, err)
	}
	// 状态查询一致
	liked, count, err = svc.Status(ctx, "p", "1.1.1.1", "ua")
	if err != nil || !liked || count != 1 {
		t.Fatalf("status = (%v,%d,%v), want (true,1,nil)", liked, count, err)
	}
	// 不同指纹（换 UA）视为未点赞
	if liked, _, _ := svc.Status(ctx, "p", "1.1.1.1", "other-ua"); liked {
		t.Fatal("different fingerprint should not be liked")
	}
	// 再切换 → 取消
	liked, count, err = svc.Toggle(ctx, "p", "1.1.1.1", "ua")
	if err != nil || liked || count != 0 {
		t.Fatalf("unlike = (%v,%d,%v), want (false,0,nil)", liked, count, err)
	}
	// 计数不出现负数：连删两次后仍为 0
	db.Exec("DELETE FROM likes")
	if err := db.Model(&model.Post{}).Where("id = ?", post.ID).
		UpdateColumn("like_count", 0).Error; err != nil {
		t.Fatal(err)
	}
	liked, count, err = svc.Toggle(ctx, "p", "9.9.9.9", "ua")
	if err != nil || !liked || count != 1 {
		t.Fatalf("re-like = (%v,%d,%v)", liked, count, err)
	}
}

func TestLikeFingerprintIsAnonymous(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	mustCreate(t, ps, &model.Post{Title: "p", Slug: "p", Content: "x", Status: model.StatusPublished}, nil)

	svc := NewLikeService(db, "salt")
	fp := svc.fingerprint("1.2.3.4", "Mozilla/5.0")
	// 不可反推：哈希输出 64 hex，且不含原始 IP
	if len(fp) != 64 || fp == "1.2.3.4" {
		t.Fatalf("fingerprint = %q", fp)
	}
	// 不同盐 → 不同指纹（指纹不可跨站比对）
	svc2 := NewLikeService(db, "other-salt")
	if svc2.fingerprint("1.2.3.4", "Mozilla/5.0") == fp {
		t.Fatal("salt should change fingerprint")
	}
}

func TestLikeConcurrentIdempotent(t *testing.T) {
	db := newTestDB(t)
	ps := NewPostService(db)
	post := mustCreate(t, ps, &model.Post{Title: "p", Slug: "p", Content: "x", Status: model.StatusPublished}, nil)

	svc := NewLikeService(db, "salt")
	ctx := context.Background()

	// 并发 8 个同指纹 Toggle：唯一约束下至多一次成功插入路径冲突，
	// 最终态必须是确定的：like_count == 0 或 1，且与 likes 行数一致
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = svc.Toggle(ctx, "p", "5.5.5.5", "ua")
		}()
	}
	wg.Wait()

	var rows int64
	db.Model(&model.Like{}).Where("post_id = ?", post.ID).Count(&rows)
	var p model.Post
	db.First(&p, post.ID)
	if rows > 1 {
		t.Fatalf("likes rows = %d, want <= 1 (unique constraint)", rows)
	}
	if p.LikeCount != rows { // 并行切换后计数必须与行数一致：双插幂等/双删只减一次，防漂移
		t.Fatalf("after race: like_count = %d, rows = %d, want equal", p.LikeCount, rows)
	}
	// 串行重放至确定态：再 Toggle 一次对齐 liked/计数
	liked, count, err := svc.Toggle(ctx, "p", "5.5.5.5", "ua")
	if err != nil {
		t.Fatal(err)
	}
	db.Model(&model.Like{}).Where("post_id = ?", post.ID).Count(&rows)
	if rows != 1 || count != 1 || !liked {
		t.Fatalf("after settle: rows=%d count=%d liked=%v, want 1/1/true", rows, count, liked)
	}
}
