'use client'

import { useState, useEffect, useCallback } from 'react'
import { History, Calendar, Eye, Trash2, Loader2 } from 'lucide-react'
import { apiFetch } from '../lib/api'

// 最多扫描多少个会话 / 展示多少条占卜记录
const MAX_CONVERSATIONS = 10
const MAX_READINGS = 20

// 后端写入的用户提问前缀，见 conversation_service.go 的 StreamTarotReading
const QUESTION_PREFIX = /^🎴\s*塔罗占卜[:：]\s*/

/**
 * 从某个会话的消息流里配对出占卜记录：
 * 一条 type=reading 的用户消息（提问）后面紧跟一条带 reading_id 的助手消息（解读）。
 */
function collectReadings(messages, conversation) {
  const out = []
  let pendingQuestion = ''

  for (const msg of messages || []) {
    if (msg.type !== 'reading') continue

    if (msg.role === 'user') {
      pendingQuestion = (msg.content || '').replace(QUESTION_PREFIX, '').trim()
    } else if (msg.role === 'assistant' && msg.reading_id) {
      out.push({
        readingId: msg.reading_id,
        conversationId: conversation.id,
        conversationTitle: conversation.title,
        question: pendingQuestion || conversation.title || '塔罗占卜',
        content: msg.content || '',
        created_at: msg.created_at,
      })
      pendingQuestion = ''
    }
  }

  return out
}

function formatDate(dateString) {
  if (!dateString) return ''
  const date = new Date(dateString)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * 真实占卜历史：数据来自 GET /conversations + GET /conversations/:id/messages，
 * 不再有任何模拟数据。
 */
export default function ReadingHistory({ refreshKey = 0, onSelect, disabled = false }) {
  const [readings, setReadings] = useState([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')

  const fetchReadings = useCallback(async () => {
    setIsLoading(true)
    setError('')
    try {
      const conversations = await apiFetch('/conversations?channel=reading')
      const recent = (conversations || []).slice(0, MAX_CONVERSATIONS)

      const batches = await Promise.all(
        recent.map(async (conv) => {
          try {
            const messages = await apiFetch(`/conversations/${conv.id}/messages`)
            return collectReadings(messages, conv)
          } catch {
            // 单个会话失败不影响其余记录
            return []
          }
        }),
      )

      setReadings(
        batches
          .flat()
          .sort((a, b) => new Date(b.created_at || 0) - new Date(a.created_at || 0))
          .slice(0, MAX_READINGS),
      )
    } catch (err) {
      // 401/403 时 apiFetch 已跳转登录页，没必要再渲染一条马上会消失的错误
      if (!err.handled) setError(err.message || '加载占卜历史失败')
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchReadings()
  }, [fetchReadings, refreshKey])

  const handleView = (reading) => {
    if (disabled || !onSelect) return
    onSelect(reading)
  }

  // 后端没有单条占卜的删除接口，只能删除其所在的整个会话
  const handleDelete = async (reading) => {
    const confirmed = window.confirm(
      `该占卜记录属于对话「${reading.conversationTitle}」。\n` +
        '删除会移除这个对话及其中的全部聊天消息，确定继续吗？',
    )
    if (!confirmed) return

    setError('')
    try {
      await apiFetch(`/conversations/${reading.conversationId}`, { method: 'DELETE' })
      setReadings((prev) => prev.filter((r) => r.conversationId !== reading.conversationId))
    } catch (err) {
      if (!err.handled) setError(err.message || '删除失败，请稍后重试')
    }
  }

  return (
    <div className="bg-white rounded-2xl shadow-xl p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center space-x-3">
          <History className="w-6 h-6 text-mystic-600" />
          <h3 className="text-lg font-semibold text-mystic-900">占卜历史</h3>
        </div>
        <button
          onClick={fetchReadings}
          className="text-sm text-primary-600 hover:text-primary-700 disabled:opacity-40"
          disabled={isLoading}
        >
          {isLoading ? '刷新中…' : '刷新'}
        </button>
      </div>

      {error && (
        <div className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-8 text-mystic-500">
          <Loader2 className="w-5 h-5 animate-spin mr-2" />
          <span className="text-sm">加载中…</span>
        </div>
      ) : readings.length === 0 ? (
        <div className="text-center py-8">
          <Calendar className="w-12 h-12 text-mystic-300 mx-auto mb-3" />
          <p className="text-mystic-500">开始你的第一次占卜，历史记录将在这里显示</p>
        </div>
      ) : (
        <div className="space-y-3">
          {readings.map((reading) => (
            <div
              key={`${reading.conversationId}-${reading.readingId}`}
              className={`group border border-mystic-200 rounded-xl p-4 transition-colors ${
                disabled ? 'opacity-60' : 'hover:bg-mystic-50 cursor-pointer'
              }`}
              onClick={() => handleView(reading)}
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-mystic-900 truncate">
                    {reading.question}
                  </p>
                  <div className="flex items-center space-x-3 mt-2">
                    <span className="text-xs text-mystic-500">
                      {formatDate(reading.created_at)}
                    </span>
                    <span className="text-xs text-mystic-400 truncate">
                      {reading.conversationTitle}
                    </span>
                  </div>
                </div>
                <div className="flex items-center space-x-1 ml-2 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      handleView(reading)
                    }}
                    disabled={disabled}
                    className="p-1.5 text-mystic-500 hover:text-primary-600 hover:bg-mystic-100 rounded-lg disabled:opacity-40"
                    title="查看详情"
                  >
                    <Eye className="w-4 h-4" />
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      handleDelete(reading)
                    }}
                    className="p-1.5 text-mystic-500 hover:text-red-600 hover:bg-red-50 rounded-lg"
                    title="删除该对话"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {!isLoading && readings.length > 0 && (
        <div className="mt-4 pt-4 border-t border-mystic-100">
          <p className="text-sm text-mystic-500 text-center">
            共 {readings.length} 条占卜记录
          </p>
        </div>
      )}
    </div>
  )
}
