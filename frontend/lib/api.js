export const API_BASE = process.env.NEXT_PUBLIC_API_URL || '/api/v1'

/**
 * 把后端返回的错误体转成可读信息。
 * Go 端错误统一为 {"error": "..."}，也兼容纯文本响应。
 */
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
 * 统一的 JSON 请求入口：自动补 API_BASE、带 cookie、解析错误体。
 */
export async function apiFetch(path, { headers, ...rest } = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    credentials: 'same-origin',
    ...rest,
    headers: { 'Content-Type': 'application/json', ...(headers || {}) },
  })

  if (!res.ok) {
    const text = await res.text().catch(() => '')
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
