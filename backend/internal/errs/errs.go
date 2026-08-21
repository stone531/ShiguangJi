package errs

// BizError 全局业务错误；Code 为业务码：40001 参数 / 40004 不存在 / 20001/20002 token。
type BizError struct {
	Code int
	Msg  string
}

func (e *BizError) Error() string { return e.Msg }

// New 构造业务错误。
func New(code int, msg string) *BizError { return &BizError{Code: code, Msg: msg} }

var (
	ErrParam        = New(40001, "参数错误")
	ErrNotFound     = New(40004, "资源不存在")
	ErrUnauthorized = New(20001, "token 校验失败")
	ErrTokenExpired = New(20002, "token 已过期")
	ErrRateLimited  = New(42901, "请求过于频繁")
)
