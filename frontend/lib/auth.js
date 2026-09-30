import { apiFetch } from './api'

/**
 * 认证 API 封装。
 *
 * 会话由后端以 HttpOnly Cookie 维持，前端拿不到也不需要 token，
 * 因此这里没有任何本地存储逻辑——这也是选 Session Cookie 而非 JWT 的收益之一。
 */

/** 返回 { user } 或抛错。401 会被 apiFetch 转成跳登录页。 */
export async function getMe() {
  return apiFetch('/auth/me')
}

/** 登录。成功后 Cookie 由后端下发，前端只需刷新视图状态。 */
export async function login(username, password) {
  return apiFetch('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
    // 登录接口的 401 表示「口令错误」，不是会话失效，
    // 不能让 apiFetch 把它吞掉并跳转。
    skipAuthRedirect: true,
  })
}

/** 登出。后端删除会话并让 Cookie 过期。 */
export async function logout() {
  try {
    await apiFetch('/auth/logout', { method: 'POST' })
  } catch {
    // 会话可能已失效，登出失败不应阻塞用户离开
  }
}

/** 修改口令。成功后除当前会话外的其他会话都会失效。 */
export async function changePassword(oldPassword, newPassword) {
  return apiFetch('/auth/password', {
    method: 'POST',
    body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }),
  })
}

export const ROLE_ADMIN = 'admin'

export function isAdmin(user) {
  return user?.role === ROLE_ADMIN
}
