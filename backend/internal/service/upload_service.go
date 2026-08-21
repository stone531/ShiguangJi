// Package service 上传业务：魔数校验 + MD5 命名 + 同步缩略图。
package service

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"

	"shiguangji/internal/errs"
)

const thumbMaxWidth = 800

// UploadResult 上传结果：原图与缩略图的相对 URL。
type UploadResult struct {
	URL      string `json:"url"`
	ThumbURL string `json:"thumb_url"`
}

// ThumbGenerator 缩略图生成接口：Upload 依赖它，测试可替换。
type ThumbGenerator interface {
	Generate(src image.Image) (thumbBytes []byte, ext string, err error)
}

// UploadService 上传服务。
type UploadService struct {
	dir      string
	maxBytes int64
	thumbGen ThumbGenerator
}

// NewUploadService 构造；maxBytes 为单文件大小上限。
func NewUploadService(dir string, maxBytes int64) *UploadService {
	return &UploadService{dir: dir, maxBytes: maxBytes, thumbGen: defaultThumbGen{}}
}

// defaultThumbGen 用 imaging 库生成 PNG 缩略图（宽 ≤ 800，等比缩放）。
type defaultThumbGen struct{}

// Generate 实现 ThumbGenerator。
func (defaultThumbGen) Generate(src image.Image) ([]byte, string, error) {
	b := src.Bounds()
	var thumb *image.NRGBA
	if b.Dx() > thumbMaxWidth {
		// 宽超限时等比缩到 800（imaging.Fit 的高度参数传 0 不生效，手动算等比高度）
		h := b.Dy() * thumbMaxWidth / b.Dx()
		if h < 1 {
			h = 1
		}
		thumb = imaging.Resize(src, thumbMaxWidth, h, imaging.Lanczos)
	} else {
		thumb = imaging.Clone(src)
	}
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, thumb, imaging.PNG); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), ".png", nil
}

var allowedExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}

// sniff 检测真实格式（魔数）：jpg/png/gif/webp 四种文件头。
func sniff(head []byte) (string, bool) {
	switch {
	case bytes.HasPrefix(head, []byte("\xFF\xD8\xFF")):
		return ".jpg", true
	case bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n")):
		return ".png", true
	case bytes.HasPrefix(head, []byte("GIF87a")), bytes.HasPrefix(head, []byte("GIF89a")):
		return ".gif", true
	case len(head) >= 12 && bytes.HasPrefix(head, []byte("RIFF")) && string(head[8:12]) == "WEBP":
		return ".webp", true
	}
	return "", false
}

// Upload 校验→MD5 命名→落盘→同步生成缩略图。
// 注意：不启动任何 goroutine——个人站上传频率低，同步生成（通常 <100ms）避免异步生命周期复杂度。
func (s *UploadService) Upload(filename string, r io.Reader) (*UploadResult, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExt[ext] {
		return nil, errs.New(40001, "仅支持 jpg/png/webp/gif")
	}

	// io.LimitReader 限长读入内存，防超大请求 OOM
	data, err := io.ReadAll(io.LimitReader(r, s.maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > s.maxBytes {
		return nil, errs.New(40001, "文件超过大小限制")
	}

	// 魔数校验：空文件/超短文件防护——先判长度再取头部
	if len(data) < 12 {
		return nil, errs.New(40001, "文件内容不是有效图片")
	}
	// 直接传整个 data：bytes.HasPrefix 对短切片安全，webp 分支自带 len>=12 保护
	realExt, ok := sniff(data)
	if !ok {
		return nil, errs.New(40001, "文件内容不是有效图片")
	}
	// jpeg 归一：.jpeg 与 .jpg 视为同一格式
	if ext != realExt && !(ext == ".jpeg" && realExt == ".jpg") {
		return nil, errs.New(40001, "扩展名与文件内容不符")
	}

	sum := md5.Sum(data)
	name := hex.EncodeToString(sum[:]) + realExt
	dst := filepath.Join(s.dir, name)
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return nil, err
	}

	res := &UploadResult{URL: "/uploads/" + name}

	// gif 为动图，imaging 解码只取首帧且逐帧重编码代价高——直接用原图兜底；
	// webp 同理：imaging 库不支持 webp 解码，thumb 兜底为原图（见下方分支）。
	needThumb := ext != ".gif" && ext != ".webp"
	if needThumb {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err == nil {
			if tb, thumbExt, err := s.thumbGen.Generate(img); err == nil {
				thumbName := hex.EncodeToString(sum[:]) + "_thumb" + thumbExt
				if err := os.WriteFile(filepath.Join(s.dir, thumbName), tb, 0o644); err == nil {
					res.ThumbURL = "/uploads/" + thumbName
				}
			}
		}
		// 缩略图失败不阻断上传：原图仍可用
	}
	if res.ThumbURL == "" {
		res.ThumbURL = res.URL // 兜底：gif/webp 或生成失败时用原图
	}
	return res, nil
}
