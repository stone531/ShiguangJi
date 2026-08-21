package handler

import (
	"github.com/gin-gonic/gin"

	"shiguangji/internal/errs"
	"shiguangji/internal/service"
)

// UploadHandler 上传处理器。
type UploadHandler struct {
	svc *service.UploadService
}

// NewUploadHandler 构造。
func NewUploadHandler(svc *service.UploadService) *UploadHandler {
	return &UploadHandler{svc: svc}
}

// Upload multipart 字段 "file"：FormFile → Open（defer Close）→ service.Upload。
func (h *UploadHandler) Upload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		Fail(c, errs.New(40001, "缺少文件字段 file"))
		return
	}
	f, err := fh.Open()
	if err != nil {
		Fail(c, err)
		return
	}
	defer f.Close() // 资源关闭义务

	res, err := h.svc.Upload(fh.Filename, f)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, res)
}
