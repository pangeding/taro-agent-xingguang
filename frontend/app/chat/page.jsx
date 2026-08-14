'use client'

import { useState, useEffect, useRef, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { Plus, Trash2, Menu } from 'lucide-react'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000/api/v1'

function SimpleMarkdown({ content }) {
  if (!content) return null

  const parts = content.split(/(```[\s\S]*?```|`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*|#{1,3} .+?)(?:\n|$)/g)

  return (
    <div className="markdown-content">
      {parts.map((part, i) => {
        if (part.startsWith('```')) {
          const code = part.replace(/^```\w*\n?|\n?```$/g, '')
          return (
            <pre key={i} className="bg-mystic-800 text-mystic-100 p-3 rounded-lg my-2 overflow-x-auto text-sm">
              <code>{code}</code>
            </pre>
          )
        }
        if (part.startsWith('`') && part.endsWith('`')) {
          return <code key={i} className="bg-mystic-100 text-mystic-800 px-1 py-0.5 rounded text-sm">{part.slice(1, -1)}</code>
        }
        if (part.startsWith('**') && part.endsWith('**')) {
          return <strong key={i} className="font-bold">{part.slice(2, -2)}</strong>
        }
        if (part.startsWith('*') && part.endsWith('*') && !part.startsWith('**')) {
          return <em key={i}>{part.slice(1, -1)}</em>
        }
        if (part.startsWith('### ')) {
          return <h3 key={i} className="text-lg font-bold mt-3 mb-1">{part.slice(4)}</h3>
        }
        if (part.startsWith('## ')) {
          return <h2 key={i} className="text-xl font-bold mt-3 mb-2">{part.slice(3)}</h2>
        }
        if (part.startsWith('# ')) {
          return <h1 key={i} className="text-2xl font-bold mt-3 mb-2">{part.slice(2)}</h1>
        }
        return <span key={i}>{part}</span>
      })}
    </div>
  )
}

export default function ChatPage() {
  const [userId, setUserId] = useState(null)
  const [conversations, setConversations] = useState([])
  const [currentConversation, setCurrentConversation] = useState(null)
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isStreaming, setIsStreaming] = useState(false)
  const [streamContent, setStreamContent] = useState('')
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const messagesEndRef = useRef(null)
  const abortControllerRef = useRef(null)

  useEffect(() => {
    initUser()
  }, [])

  useEffect(() => {
    scrollToBottom()
  }, [messages, streamContent])

  useEffect(() => {
    if (userId) {
      loadConversations()
    }
  }, [userId])

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  const getHeaders = () => ({
    'Content-Type': 'application/json',
    'X-User-Id': userId || '',
  })

  const initUser = async () => {
    let storedId = localStorage.getItem('user_id')
    if (!storedId) {
      try {
        const res = await fetch(`${API_BASE}/user/init`, { method: 'POST' })
        const data = await res.json()
        storedId = data.user_id
        localStorage.setItem('user_id', storedId)
        document.cookie = `user_id=${storedId}; max-age=31536000; path=/`
      } catch (err) {
        console.error('Failed to init user:', err)
        return
      }
    }
    setUserId(storedId)
  }

  const loadConversations = async () => {
    try {
      const res = await fetch(`${API_BASE}/conversations/`, {
        headers: getHeaders(),
      })
      if (res.ok) {
        const data = await res.json()
        setConversations(data)
      }
    } catch (err) {
      console.error('Failed to load conversations:', err)
    }
  }

  const loadConversation = async (id) => {
    try {
      const res = await fetch(`${API_BASE}/conversations/${id}`, {
        headers: getHeaders(),
      })
      if (res.ok) {
        const data = await res.json()
        setCurrentConversation(data)
        setMessages(data.messages || [])
      }
    } catch (err) {
      console.error('Failed to load conversation:', err)
    }
  }

  const createConversation = async () => {
    try {
      const res = await fetch(`${API_BASE}/conversations/`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({}),
      })
      if (res.ok) {
        const data = await res.json()
        loadConversations()
        setCurrentConversation(data)
        setMessages([])
      }
    } catch (err) {
      console.error('Failed to create conversation:', err)
    }
  }

  const deleteConversation = async (id) => {
    try {
      const res = await fetch(`${API_BASE}/conversations/${id}`, {
        method: 'DELETE',
        headers: getHeaders(),
      })
      if (res.ok) {
        if (currentConversation?.id === id) {
          setCurrentConversation(null)
          setMessages([])
        }
        loadConversations()
      }
    } catch (err) {
      console.error('Failed to delete conversation:', err)
    }
  }

  const sendMessage = async (content) => {
    if (!currentConversation || !content.trim() || isStreaming) return

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

    try {
      const response = await fetch(`${API_BASE}/conversations/${currentConversation.id}/messages`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ content: content.trim() }),
      })

      if (!response.ok) throw new Error('Failed to send message')

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let fullContent = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const dataStr = line.slice(6)
            try {
              const data = JSON.parse(dataStr)
              if (data.delta) {
                fullContent += data.delta
                setStreamContent(fullContent)
              }
              if (data.done) {
                const assistantMessage = {
                  id: Date.now(),
                  role: 'assistant',
                  content: data.full_text || fullContent,
                  type: 'text',
                  created_at: new Date().toISOString(),
                }
                setMessages((prev) => [...prev, assistantMessage])
                setStreamContent('')
                setIsStreaming(false)
                loadConversations()
              }
            } catch (e) {
              // ignore JSON parse errors
            }
          }
        }
      }
    } catch (err) {
      console.error('Stream error:', err)
      setIsStreaming(false)
      setStreamContent('')
      setMessages((prev) => [
        ...prev,
        {
          id: Date.now(),
          role: 'assistant',
          content: '抱歉，回复出现错误，请重试。',
          type: 'text',
          created_at: new Date().toISOString(),
        },
      ])
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
    <div className="flex h-screen bg-mystic-50">
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
          {messages.length === 0 && !isStreaming && (
            <div className="flex flex-col items-center justify-center h-full text-center">
              <div className="text-6xl mb-4">🔮</div>
              <h2 className="text-2xl font-bold text-mystic-900 mb-2">开始你的塔罗对话</h2>
              <p className="text-mystic-600 max-w-md">
                向星语塔罗师提问，或在对话中使用"抽牌"功能进行塔罗占卜
              </p>
            </div>
          )}

          {messages.map((msg) => (
            <div
              key={msg.id}
              className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div className={`max-w-2xl p-4 rounded-xl ${
                msg.role === 'user'
                  ? 'bg-gradient-mystic text-white'
                  : 'bg-white shadow-md text-mystic-800'
              }`}>
              <div className="wysiwyg">
                <SimpleMarkdown content={msg.content} />
              </div>
              </div>
            </div>
          ))}

          {isStreaming && streamContent && (
            <div className="flex justify-start">
              <div className="max-w-2xl p-4 rounded-xl bg-white shadow-md text-mystic-800">
              <div className="wysiwyg">
                <SimpleMarkdown content={streamContent} />
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
              disabled={isStreaming || !currentConversation}
              className="flex-1 px-4 py-3 border border-mystic-300 rounded-xl focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
              rows={1}
            />
            <button
              onClick={handleSend}
              disabled={isStreaming || !input.trim() || !currentConversation}
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
