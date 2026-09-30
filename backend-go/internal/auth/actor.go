// Package auth 提供身份与口令的纯函数能力：Actor 归属过滤、口令哈希与强度校验、会话 token。
//
// 该包不依赖 gin，也不碰数据库，便于被 service / middleware / scripts 三处复用。
package auth

import (
	"gorm.io/gorm"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// Actor 是本次请求的调用者。
//
// All 只在「管理员 + 请求显式传 scope=all」时为 true。
// 这是刻意的设计：管理员默认也只看得到自己的数据，
// 否则管理员在正常浏览自己的历史时会看到全站数据（见技术文档 §3.4）。
type Actor struct {
	UserID uint
	Role   string
	All    bool
}

// IsAdmin 返回调用者是否为管理员。
func (a Actor) IsAdmin() bool { return a.Role == RoleAdmin }

// VisibleTo 给查询加上归属过滤。col 是目标表的归属列名（如 "owner_id"）。
//
// 规则：默认只可见自己的数据；仅当 Role==admin 且 All==true 时才不过滤。
//
// 注意 owner_id 的默认值是 0，而 users.id 自增从 1 起，
// 因此「未归属」的行对任何真实用户都不可见——这是迁移未完成时的安全兜底。
//
// 刻意不提供「无条件不过滤」的 SystemActor 之类的便利函数：
// 那类 API 一旦存在，就总有人会在业务代码里顺手用它绕过隔离。
// 确实需要跨用户查询时，走 Actor{All: true} 并在调用点显式写明理由。
func (a Actor) VisibleTo(q *gorm.DB, col string) *gorm.DB {
	if a.IsAdmin() && a.All {
		return q
	}
	return q.Where(col+" = ?", a.UserID)
}
