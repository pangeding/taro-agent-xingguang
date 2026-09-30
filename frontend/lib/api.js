export const API_BASE = process.env.NEXT_PUBLIC_API_URL || '/api/v1'

/** 是否处于浏览器环境（这些模块也会被 Next 的服务端预渲染阶段导入）。 */
const isBrowser = typeof window !== 'undefined'

/**
 * 登录后要跳回的路径参数名。登录页读取它，其余地方写入它。
 */
export const NEXT_PARAM = 'next'

/**
 * 重定向到登录页，并带上当前路径以便登录后跳回。
 *
 * 关键：当前已在 /login 时不再跳转，否则登录页自身的 401（比如探测会话）
 * 会触发无限重定向。
 */
export function redirectToLogin() {
  if (!isBrowser) return
  if (window.location.pathname === '/login') return

  const next = window.location.pathname + window.location.search
  const target = `/login?${NEXT_PARAM}=${encodeURIComponent(next)}`
  window.location.replace(target)
}

/**
 * 强制改密重定向。后端在 APP_ENV=prod 且 must_change_password=true 时，
 * 除白名单外的所有接口都返回 403 password_change_required。
 */
export function redirectToChangePassword() {
  if (!isBrowser) return
  if (window.location.pathname === '/account/password') return
  window.location.replace('/account/password')
}

/** 把后端返回的错误体转成可读信息。 */
export function extractError(text, status) {
  if (text) {
    try {
      const parsed = JSON.parse(text)
      if (parsed && typeof parsed === 'object' && parsed.error) {
        return String(parsed.error)
      }
    } catch {
      const trimmed = text.trim()
      if (trimmed && trimmed.length <= 200) return trimmed
    }
  }
  return `请求失败 (${status})`
}

/**
 * 统一处理认证类失败。
 *
 * 返回 true 表示「已经处理掉了，调用方不要再展示错误」——
 * 因为页面马上要跳走，此时弹一条红色错误只会闪一下。
 */
function handleAuthFailure(status, text) {
  if (status === 401) {
    redirectToLogin()
    return true
  }
  if (status === 403) {
    try {
      const parsed = JSON.parse(text)
      if (parsed?.error === 'password_change_required') {
        redirectToChangePassword()
        return true
      }
    } catch {
      // 非 JSON 的 403 交由调用方展示
    }
  }
  return false
}

/**
 * 统一的 JSON 请求入口：自动补 API_BASE、带 cookie、解析错误体、处理认证失效。
 *
 * skipAuthRedirect：登录接口自身要传 true。
 * 否则「口令错误」返回的 401 会被当成会话失效处理，
 * 错误文案被替换成「登录状态已失效」，用户看不到真正的原因。
 */
export async function apiFetch(path, { headers, skipAuthRedirect = false, ...rest } = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    credentials: 'same-origin',
    ...rest,
    headers: { 'Content-Type': 'application/json', ...(headers || {}) },
  })

  if (!res.ok) {
    const text = await res.text().catch(() => '')
    if (!skipAuthRedirect && handleAuthFailure(res.status, text)) {
      const err = new Error('登录状态已失效')
      err.handled = true
      throw err
    }
    throw new Error(extractError(text, res.status))
  }
  if (res.status === 204) return null

  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

/**
 * 带认证失效处理的 SSE 请求。
 *
 * 与 apiFetch 分开是因为 SSE 需要在拿到 Response 之后继续读 body，
 * 不能在同一个函数里把流消费掉。
 */
export async function authAwareFetch(path, init = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    credentials: 'same-origin',
    ...init,
  })

  if (!res.ok) {
    const text = await res.text().catch(() => '')
    if (handleAuthFailure(res.status, text)) {
      const err = new Error('登录状态已失效')
      err.handled = true
      throw err
    }
    throw new Error(extractError(text, res.status))
  }
  return res
}
