package auth

import "errors"

// 认证相关的哨兵错误。
//
// 集中放在 auth 包而不是各自声明：middleware 需要 errors.Is 判断
// service 返回的是哪一种失败，两处各写一份 var 会导致比对永远不成立。
var (
	// ErrInvalidCredential 统一表示「用户名或密码错误」。
	// 刻意不区分「用户不存在」与「口令错误」，避免用户名枚举。
	ErrInvalidCredential = errors.New("用户名或密码错误")
	// ErrAccountDisabled 表示账号被禁用。
	ErrAccountDisabled = errors.New("账号已禁用")
	// ErrAccountLocked 表示连续失败次数过多被临时锁定。
	ErrAccountLocked = errors.New("尝试次数过多，请稍后再试")
	// ErrUnauthenticated 表示未登录或会话无效/过期。
	ErrUnauthenticated = errors.New("未登录")
	// ErrOldPasswordWrong 表示改密时原密码不正确。
	ErrOldPasswordWrong = errors.New("原密码错误")
)
