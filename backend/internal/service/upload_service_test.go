package service

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestUploadRejectsFakeExtension(t *testing.T) {
	dir := t.TempDir()
	svc := NewUploadService(dir, 5<<20)
	// 内容是文本，扩展名伪装 .png
	_, err := svc.Upload("evil.png", bytes.NewReader([]byte("#!/bin/sh\nrm -rf /")))
	if err == nil {
		t.Fatal("magic-number mismatch must be rejected")
	}
}

func TestUploadGeneratesThumb(t *testing.T) {
	dir := t.TempDir()
	svc := NewUploadService(dir, 5<<20)
	res, err := svc.Upload("photo.png", bytes.NewReader(makePNG(t, 2000, 1000)))
	if err != nil {
		t.Fatal(err)
	}
	if res.URL == "" || res.ThumbURL == "" {
		t.Fatalf("url/thumb empty: %+v", res)
	}
	// 缩略图必须存在且宽度 ≤ 800
	f, err := os.Open(filepath.Join(dir, filepath.Base(res.ThumbURL)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width > 800 {
		t.Fatalf("thumb width = %d, want <= 800", cfg.Width)
	}
}

func TestUploadRejectsOversize(t *testing.T) {
	dir := t.TempDir()
	svc := NewUploadService(dir, 4) // 4 字节上限
	_, err := svc.Upload("big.png", bytes.NewReader(makePNG(t, 10, 10)))
	if err == nil {
		t.Fatal("oversize must be rejected")
	}
}

func TestUploadExactMaxSize(t *testing.T) {
	dir := t.TempDir()
	data := makePNG(t, 10, 10)
	svc := NewUploadService(dir, int64(len(data))) // 上限 == 文件长度
	if _, err := svc.Upload("exact.png", bytes.NewReader(data)); err != nil {
		t.Fatalf("exact-size upload must succeed, got: %v", err)
	}
}

func TestUploadJpegExtension(t *testing.T) {
	dir := t.TempDir()
	svc := NewUploadService(dir, 5<<20)
	// 最小合法 jpeg：FF D8 FF（头）+ E0 段 + SOS + 尾
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00,
		0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00,
		0xFF, 0xD9}
	res, err := svc.Upload("photo.jpeg", bytes.NewReader(jpegData))
	if err != nil {
		t.Fatalf(".jpeg upload must succeed, got: %v", err)
	}
	if res.URL == "" || res.ThumbURL == "" {
		t.Fatalf("url/thumb empty: %+v", res)
	}
}

func TestUploadWebpThumbFallback(t *testing.T) {
	dir := t.TempDir()
	svc := NewUploadService(dir, 5<<20)
	// 合法 webp 头：RIFF + 大小 + WEBP
	webp := append([]byte("RIFF"), 0x00, 0x00, 0x00, 0x00)
	webp = append(webp, []byte("WEBP")...)
	webp = append(webp, make([]byte, 16)...)
	res, err := svc.Upload("pic.webp", bytes.NewReader(webp))
	if err != nil {
		t.Fatalf("webp upload must succeed, got: %v", err)
	}
	if res.ThumbURL != res.URL {
		t.Fatalf("webp thumb must fall back to original, got thumb=%s url=%s", res.ThumbURL, res.URL)
	}
}

func TestUploadShortPNG(t *testing.T) {
	dir := t.TempDir()
	svc := NewUploadService(dir, 5<<20)
	// 13 字节：png 魔数 + 5 字节填充——不得 panic
	short := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 5)...)
	_, err := svc.Upload("short.png", bytes.NewReader(short))
	// 内容不完整 decode 失败也 OK，但不能 panic
	_ = err
}
