'use client'

import { useState, useEffect, useRef } from 'react'
import { Plus, Trash2, Menu } from 'lucide-react'
import Markdown from '../../components/Markdown'
import { apiFetch } from '../../lib/api'
import { streamSSE } from '../../lib/sse'
import AuthGate from '../../components/AuthGate'

function CardStrip({ cards }) {
  if (!cards || !cards.length) return null
  return (
    <div className="flex flex-wrap gap-2">
      {cards.map((c, i) => (
        <div
          key={i}
          className="flex flex-col items-center bg-mystic-800 text-mystic-50 rounded-lg px-3 py-2 min-w-[72px] shadow"
        >
          <span className="text-lg">🎴</span>
          <span className="text-sm font-semibold">{c.name}</span>
          <span className="text-xs opacity-80">{c.is_reversed ? '逆位' : '正位'}</span>
          {c.position_name ? <span className="text-[10px] opacity-60">{c.position_name}</span> : null}
        </div>
      ))}
    </div>
  )
}

// 页面主体。AuthGate 保证只有已登录用户才会走到这里。
function ChatContent() {
  const [ready, setReady] = useState(false)
  const [conversations, setConversations] = useState([])
  const [currentConversation, setCurrentConversation] = useState(null)
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isStreaming, setIsStreaming] = useState(false)
  const [streamContent, setStreamContent] = useState('')
  const [streamCards, setStreamCards] = useState([])
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [error, setError] = useState('')
  const messagesEndRef = useRef(null)
  const abortControllerRef = useRef(null)

  // 身份由 AuthGate 保证（已登录才会渲染本页面），不再需要 /user/init。
  useEffect(() => () => abortControllerRef.current?.abort(), [])

  useEffect(() => {
    scrollToBottom()
  }, [messages, streamContent])

  useEffect(() => {
    if (!ready) return
    const bootstrap = async () => {
      const list = await loadConversations()
      if (list && list.length > 0) {
        loadConversation(list[0].id)
      } else {
        createConversation()
      }
    }
    bootstrap()
  }, [ready])

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const loadConversations = async () => {
    try {
      // 只列 chat 频道：占卜页写入的 reading 会话不应出现在聊天侧边栏
      const data = await apiFetch('/conversations?channel=chat')
      setConversations(Array.isArray(data) ? data : [])
      return data
    } catch (err) {
      // 401/403 时 apiFetch 已跳转登录页，错误地再弹一条提示只会闪一下
      if (err.handled) return null
      console.error('Failed to load conversations:', err)
      setError('加载会话列表失败，请稍后重试。')
      return null
    }
  }

  const loadConversation = async (id) => {
    try {
      const data = await apiFetch(`/conversations/${id}`)
      if (data) {
        setCurrentConversation(data)
        setMessages(data.messages || [])
      }
    } catch (err) {
      if (err.handled) return
      console.error('Failed to load conversation:', err)
    }
  }

  const createConversation = async () => {
    try {
      const data = await apiFetch('/conversations', {
        method: 'POST',
        body: JSON.stringify({ channel: 'chat' }),
      })
      setConversations((prev) => [data, ...prev])
      setCurrentConversation(data)
      setMessages([])
      setError('')
      return data
    } catch (err) {
      if (err.handled) return null
      console.error('Failed to create conversation:', err)
      setError('创建会话失败，请稍后重试。')
      return null
    }
  }

  const deleteConversation = async (id) => {
    try {
      await apiFetch(`/conversations/${id}`, { method: 'DELETE' })
      if (currentConversation?.id === id) {
        setCurrentConversation(null)
        setMessages([])
      }
      loadConversations()
    } catch (err) {
      if (err.handled) return
      console.error('Failed to delete conversation:', err)
    }
  }

  const sendMessage = async (content) => {
    if (!content.trim() || isStreaming) return

    let conversation = currentConversation
    if (!conversation) {
      conversation = await createConversation()
      if (!conversation) return
    }

    const userMessage = {
      id: Date.now(),
      role: 'user',
      content: content.trim(),
      type: 'text',
      created_at: new Date().toISOString(),
    }
    setMessages((prev) => [...prev, userMessage])
    setInput('')
    setIsStreaming(true)
    setStreamContent('')
    setError('')

    abortControllerRef.current?.abort()
    const controller = new AbortController()
    abortControllerRef.current = controller

    let fullContent = ''
    let finished = false
    let errorMessage = null
    let drawnCards = []

    const appendAssistant = (text) => {
      setMessages((prev) => [
        ...prev,
        {
          id: Date.now() + Math.random(),
          role: 'assistant',
          content: text,
          type: 'text',
          created_at: new Date().toISOString(),
        },
      ])
    }

    const handleEvent = (eventName, data) => {
      if (!data || typeof data !== 'object') return
      if (eventName === 'error' || data.error) {
        errorMessage = data.error || '服务出错了'
        return
      }
      if (eventName === 'card_drawn' || (data.name && data.is_reversed !== undefined)) {
        drawnCards = [...drawnCards, data]
        setStreamCards(drawnCards)
        return
      }
      if (data.delta) {
        fullContent += data.delta
        setStreamContent(fullContent)
      }
      if (data.done) {
        finished = true
      }
    }

    try {
      await streamSSE(`/conversations/${conversation.id}/messages`, {
        body: { content: content.trim() },
        signal: controller.signal,
        onEvent: handleEvent,
      })
    } catch (err) {
      if (err.name === 'AbortError') {
        return
      }
      // 会话失效时 streamSSE 已跳转登录页，不要往消息流里塞一条「登录状态已失效」
      if (err.handled) {
        setError('登录状态已失效，请重新登录。')
        return
      }
      console.error('Stream error:', err)
      errorMessage = err.message || '回复出现错误，请重试。'
    } finally {
      setIsStreaming(false)
      setStreamContent('')
      setStreamCards([])

      if (errorMessage) {
        if (fullContent) appendAssistant(fullContent)
        appendAssistant(`抱歉，${errorMessage}`)
      } else if (finished) {
        // 牌面与 assistant 消息都已落库（messages.reading_id → reading_cards），
        // 回读服务端，刷新或切换会话后牌面依然在。
        await loadConversation(conversation.id)
        loadConversations()
      } else if (fullContent) {
        appendAssistant(fullContent)
      }
    }
  }

  const handleSend = () => {
    sendMessage(input)
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className="flex h-[calc(100vh-8rem)] bg-mystic-50 rounded-xl overflow-hidden border border-mystic-200">
      {/* 侧边栏 */}
      {sidebarOpen && (
        <div className="w-64 bg-white border-r border-mystic-200 flex flex-col">
          <div className="p-4 border-b border-mystic-200">
            <button
              onClick={createConversation}
              className="w-full flex items-center justify-center space-x-2 py-2 px-4 bg-gradient-mystic text-white rounded-lg hover:opacity-90"
            >
              <Plus className="w-4 h-4" />
              <span>新对话</span>
            </button>
          </div>

          <div className="flex-1 overflow-y-auto">
            {conversations.map((conv) => (
              <div
                key={conv.id}
                className={`flex items-center justify-between p-3 cursor-pointer hover:bg-mystic-100 ${
                  currentConversation?.id === conv.id ? 'bg-mystic-100 border-r-2 border-primary-500' : ''
                }`}
                onClick={() => loadConversation(conv.id)}
              >
                <span className="truncate flex-1 text-sm text-mystic-700">{conv.title}</span>
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    deleteConversation(conv.id)
                  }}
                  className="text-mystic-400 hover:text-red-500"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 主内容区 */}
      <div className="flex-1 flex flex-col">
        {/* 顶部栏 */}
        <div className="flex items-center p-4 bg-white border-b border-mystic-200">
          <button onClick={() => setSidebarOpen(!sidebarOpen)} className="text-mystic-600 hover:text-mystic-800">
            <Menu className="w-5 h-5" />
          </button>
          <h1 className="ml-3 text-lg font-semibold text-mystic-900">
            {currentConversation?.title || '新对话'}
          </h1>
        </div>

        {/* 消息列表 */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {error && (
            <div className="flex items-start justify-between gap-3 bg-red-50 border border-red-200 text-red-700 text-sm px-4 py-2 rounded-lg">
              <span>{error}</span>
              <button onClick={() => setError('')} className="text-red-400 hover:text-red-600">
                ✕
              </button>
            </div>
          )}

          {messages.length === 0 && !isStreaming && (
            <div className="flex flex-col items-center justify-center h-full text-center">
              <div className="text-6xl mb-4">🔮</div>
              <h2 className="text-2xl font-bold text-mystic-900 mb-2">开始你的塔罗对话</h2>
              <p className="text-mystic-600 max-w-md">
                向星语塔罗师提问，或在对话中使用"抽牌"功能进行塔罗占卜
              </p>
            </div>
          )}

          {messages.map((msg) => {
            const hasCards = Array.isArray(msg.cards) && msg.cards.length > 0
            return (
              <div
                key={msg.id}
                className={`flex flex-col ${msg.role === 'user' ? 'items-end' : 'items-start'}`}
              >
                {hasCards && (
                  <div className="mb-2 max-w-2xl">
                    <CardStrip cards={msg.cards} />
                  </div>
                )}
                <div className={`max-w-2xl p-4 rounded-xl ${
                  msg.role === 'user'
                    ? 'bg-gradient-mystic text-white'
                    : 'bg-white shadow-md text-mystic-800'
                }`}>
                  <div className="wysiwyg">
                    <Markdown content={msg.content} />
                  </div>
                </div>
              </div>
            )
          })}

          {isStreaming && streamCards.length > 0 && (
            <div className="flex justify-start">
              <CardStrip cards={streamCards} />
            </div>
          )}

          {isStreaming && streamContent && (
            <div className="flex justify-start">
              <div className="max-w-2xl p-4 rounded-xl bg-white shadow-md text-mystic-800">
              <div className="wysiwyg">
                <Markdown content={streamContent} />
              </div>
              </div>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* 输入框 */}
        <div className="p-4 bg-white border-t border-mystic-200">
          <div className="flex space-x-2">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="输入你的问题... (Enter发送, Shift+Enter换行)"
              disabled={isStreaming}
              className="flex-1 px-4 py-3 border border-mystic-300 rounded-xl focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
              rows={1}
            />
            <button
              onClick={handleSend}
              disabled={isStreaming || !input.trim()}
              className="px-6 py-3 bg-gradient-mystic text-white font-semibold rounded-xl hover:opacity-90 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              发送
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default function ChatPage() {
  return (
    <AuthGate>
      <ChatContent />
    </AuthGate>
  )
}
