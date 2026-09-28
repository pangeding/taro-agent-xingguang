import { API_BASE, extractError } from './api'

/**
 * 解析一个 SSE 数据块（以空行分隔）。
 * 返回 null 表示该块没有 data 行或无法解析。
 */
function decodeBlock(block) {
  let event = 'message'
  const dataLines = []

  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
    } else if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).replace(/^ /, ''))
    }
  }

  if (!dataLines.length) return null

  const raw = dataLines.join('\n')
  try {
    return { event, data: JSON.parse(raw) }
  } catch {
    console.warn('[sse] 跳过无法解析的数据块:', raw)
    return null
  }
}

/**
 * 消费一个 SSE 流。
 *
 * @param {string} path  以 / 开头的 API 路径，例如 `/conversations/1/tarot`
 * @param {object} opts
 * @param {object}   [opts.body]    请求体，会被 JSON 序列化
 * @param {string}   [opts.method]  默认 POST
 * @param {AbortSignal} [opts.signal]
 * @param {(event: string, data: any) => void} opts.onEvent
 *
 * 抛出 AbortError 表示调用方主动取消；其余错误已是可读文案。
 */
export async function streamSSE(path, { method = 'POST', body, signal, onEvent } = {}) {
  const response = await fetch(`${API_BASE}${path}`, {
    method,
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    signal,
    body: body === undefined ? undefined : JSON.stringify(body),
  })

  if (!response.ok) {
    const text = await response.text().catch(() => '')
    throw new Error(extractError(text, response.status))
  }
  if (!response.body) {
    throw new Error('当前环境不支持流式响应')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  const dispatch = (block) => {
    const decoded = decodeBlock(block)
    if (decoded) onEvent(decoded.event, decoded.data)
  }

  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const blocks = buffer.split('\n\n')
      buffer = blocks.pop() || ''
      for (const block of blocks) dispatch(block)
    }

    // 流结束时可能残留一个没有结尾空行的数据块
    buffer += decoder.decode()
    if (buffer.trim()) dispatch(buffer)
  } finally {
    try {
      await reader.cancel()
    } catch {
      // 流已正常结束或已被取消
    }
  }
}
