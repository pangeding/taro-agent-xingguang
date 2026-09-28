'use client'

import { useState, useEffect, useRef, useCallback } from 'react'
import { Sparkles, Clock, Star, HelpCircle, RotateCcw, Loader2 } from 'lucide-react'
import TarotCard from '../components/TarotCard'
import ReadingHistory from '../components/ReadingHistory'
import Markdown from '../components/Markdown'
import { apiFetch } from '../lib/api'
import { streamSSE } from '../lib/sse'

const EXAMPLE_QUESTIONS = [
  '我最近的工作运势如何？',
  '应该如何改善人际关系？',
  '下一步的人生方向是什么？',
  '这段感情的未来发展会怎样？',
  '近期有什么需要注意的事项？',
]

const SPREADS = [
  { id: 'single', name: '单张牌阵', desc: '快速洞察问题核心' },
  { id: 'three', name: '三张牌阵', desc: '过去、现在、未来' },
]

const FEATURES = [
  {
    icon: Sparkles,
    title: 'AI智能解读',
    desc: '结合牌面含义与用户问题，提供个性化深度解读',
  },
  {
    icon: Clock,
    title: '多种牌阵',
    desc: '支持单张、三张等多种经典牌阵选择',
  },
  {
    icon: Star,
    title: '专业准确',
    desc: '基于传统塔罗牌知识与现代AI技术',
  },
]

export default function Home() {
  const [ready, setReady] = useState(false)
  const [conversationId, setConversationId] = useState(null)

  const [question, setQuestion] = useState('')
  const [spreadType, setSpreadType] = useState('single')

  // idle → drawing（已抽牌，正在解读）→ streaming（逐字输出）→ done
  const [phase, setPhase] = useState('idle')
  const [liveCards, setLiveCards] = useState([])
  const [streamText, setStreamText] = useState('')
  const [reading, setReading] = useState(null)
  const [error, setError] = useState('')

  const [historyVersion, setHistoryVersion] = useState(0)
  const abortRef = useRef(null)

  const isBusy = phase === 'drawing' || phase === 'streaming'
  const hasResult = Boolean(reading) || liveCards.length > 0 || isBusy || Boolean(streamText)

  // 初始化：建立用户身份，并复用最近一个会话
  useEffect(() => {
    let cancelled = false

    ;(async () => {
      try {
        await apiFetch('/user/init', { method: 'POST' })
        const list = await apiFetch('/conversations')
        if (!cancelled && Array.isArray(list) && list.length > 0) {
          setConversationId(list[0].id)
        }
      } catch (err) {
        if (!cancelled) {
          setError(err.message || '无法连接到服务，请确认后端已启动后刷新页面')
        }
      } finally {
        if (!cancelled) setReady(true)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [])

  // 离开页面时中断进行中的流
  useEffect(() => () => abortRef.current?.abort(), [])

  const ensureConversation = useCallback(async () => {
    if (conversationId) return conversationId
    const conv = await apiFetch('/conversations', {
      method: 'POST',
      body: JSON.stringify({}),
    })
    if (!conv?.id) throw new Error('创建会话失败')
    setConversationId(conv.id)
    return conv.id
  }, [conversationId])

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (isBusy) return
    if (!question.trim()) {
      setError('请输入你的问题')
      return
    }

    setError('')
    setReading(null)
    setLiveCards([])
    setStreamText('')
    setPhase('drawing')

    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller

    let readingId = null
    let collected = ''
    let failure = ''

    try {
      const convId = await ensureConversation()

      await streamSSE(`/conversations/${convId}/tarot`, {
        body: { question: question.trim(), spread_type: spreadType },
        signal: controller.signal,
        onEvent: (event, data) => {
          if (!data || typeof data !== 'object') return

          if (event === 'error' || data.error) {
            failure = data.error || '解读服务出错了'
            return
          }

          // 逐张抽牌：立即渲染，不等解读完成
          if (
            event === 'card_drawn' ||
            (data.name !== undefined && data.is_reversed !== undefined)
          ) {
            setLiveCards((prev) => [...prev, data])
            return
          }

          // 逐字流式解读
          if (data.delta) {
            collected += data.delta
            setStreamText(collected)
            setPhase('streaming')
            return
          }

          if (data.done) {
            if (typeof data.full_text === 'string' && data.full_text) {
              collected = data.full_text
              setStreamText(data.full_text)
            }
            if (data.reading_id) readingId = data.reading_id
          }
        },
      })

      if (failure) throw new Error(failure)

      setPhase('done')
      setHistoryVersion((v) => v + 1)

      // 流式阶段只有综合解读，再取一次完整记录以获得每张牌的解读与真实牌义
      if (readingId) {
        try {
          const full = await apiFetch(`/readings/${readingId}`)
          if (full) setReading(full)
        } catch (err) {
          console.warn('加载完整牌面信息失败:', err)
        }
      }
    } catch (err) {
      if (err.name === 'AbortError') {
        setPhase('idle')
        return
      }
      setError(err.message || '占卜失败，请稍后重试')
      setPhase('idle')
    }
  }

  const handleReset = () => {
    abortRef.current?.abort()
    abortRef.current = null
    setReading(null)
    setLiveCards([])
    setStreamText('')
    setQuestion('')
    setError('')
    setPhase('idle')
  }

  const handleSelectHistory = async (entry) => {
    if (isBusy) return
    abortRef.current?.abort()
    setError('')
    setLiveCards([])
    setStreamText(entry.content || '')
    setPhase('done')
    setReading({
      reading_id: entry.readingId,
      question: entry.question,
      created_at: entry.created_at,
      cards: [],
    })

    try {
      const full = await apiFetch(`/readings/${entry.readingId}`)
      if (full) setReading(full)
    } catch (err) {
      setError(`加载该占卜记录失败：${err.message}`)
    }
  }

  const displayCards = reading?.cards?.length
    ? reading.cards
    : liveCards.map((c, i) => ({ ...c, position: c.position ?? i }))
  const displaySpreadType = reading?.spread_type || spreadType
  const displayQuestion = reading?.question || question

  return (
    <div className="space-y-8">
      {/* 英雄区域 */}
      <div className="text-center space-y-4">
        <div className="inline-flex items-center space-x-2 bg-gradient-mystic text-white px-4 py-2 rounded-full">
          <Sparkles className="w-5 h-5" />
          <span className="font-medium">AI智能解读</span>
        </div>
        <h1 className="text-4xl md:text-5xl font-bold text-mystic-900">
          探索塔罗牌的智慧
        </h1>
        <p className="text-xl text-mystic-600 max-w-3xl mx-auto">
          结合传统塔罗牌智慧与现代人工智能技术，为您提供个性化、深入的塔罗牌解读体验
        </p>
      </div>

      <div className="grid lg:grid-cols-3 gap-8">
        {/* 左侧：占卜表单 / 结果 */}
        <div className="lg:col-span-2 space-y-6">
          <div className="bg-white rounded-2xl shadow-xl p-6 card-hover">
            <div className="flex items-center justify-between mb-6">
              <div className="flex items-center space-x-3">
                <div className="p-2 bg-primary-100 rounded-lg">
                  <HelpCircle className="w-6 h-6 text-primary-600" />
                </div>
                <div>
                  <h2 className="text-2xl font-bold text-mystic-900">开始你的占卜</h2>
                  <p className="text-mystic-500">提出你的问题，让塔罗牌为你指引方向</p>
                </div>
              </div>
              <button
                onClick={handleReset}
                className="flex items-center space-x-2 text-mystic-600 hover:text-primary-600"
              >
                <RotateCcw className="w-5 h-5" />
                <span>重新开始</span>
              </button>
            </div>

            {error && (
              <div className="mb-6 flex items-start justify-between gap-3 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                <span>{error}</span>
                <button
                  onClick={() => setError('')}
                  className="text-red-400 hover:text-red-600"
                  aria-label="关闭提示"
                >
                  ✕
                </button>
              </div>
            )}

            {!hasResult ? (
              <form onSubmit={handleSubmit} className="space-y-6">
                {/* 问题输入 */}
                <div>
                  <label className="block text-sm font-medium text-mystic-700 mb-2">
                    你的问题
                  </label>
                  <textarea
                    value={question}
                    onChange={(e) => setQuestion(e.target.value)}
                    placeholder="例如：我最近的工作运势如何？"
                    className="w-full h-32 px-4 py-3 border border-mystic-300 rounded-xl focus:ring-2 focus:ring-primary-500 focus:border-transparent resize-none"
                    disabled={isBusy}
                  />

                  {/* 示例问题 */}
                  <div className="mt-4">
                    <p className="text-sm text-mystic-500 mb-2">不知道问什么？试试这些问题：</p>
                    <div className="flex flex-wrap gap-2">
                      {EXAMPLE_QUESTIONS.map((q, i) => (
                        <button
                          key={i}
                          type="button"
                          onClick={() => setQuestion(q)}
                          className="px-3 py-1.5 text-sm bg-mystic-100 hover:bg-mystic-200 text-mystic-700 rounded-lg transition-colors"
                        >
                          {q}
                        </button>
                      ))}
                    </div>
                  </div>
                </div>

                {/* 牌阵选择 */}
                <div>
                  <label className="block text-sm font-medium text-mystic-700 mb-2">
                    选择牌阵
                  </label>
                  <div className="grid grid-cols-2 gap-4">
                    {SPREADS.map((spread) => (
                      <button
                        key={spread.id}
                        type="button"
                        onClick={() => setSpreadType(spread.id)}
                        className={`p-4 border-2 rounded-xl text-left transition-all ${
                          spreadType === spread.id
                            ? 'border-primary-500 bg-primary-50'
                            : 'border-mystic-200 hover:border-mystic-300'
                        }`}
                      >
                        <div className="font-medium text-mystic-900">{spread.name}</div>
                        <div className="text-sm text-mystic-500 mt-1">{spread.desc}</div>
                      </button>
                    ))}
                  </div>
                </div>

                {/* 提交按钮 */}
                <button
                  type="submit"
                  disabled={isBusy || !ready}
                  className="w-full py-4 bg-gradient-mystic text-white font-semibold rounded-xl hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {!ready ? (
                    <div className="flex items-center justify-center space-x-2">
                      <Loader2 className="w-5 h-5 animate-spin" />
                      <span>正在初始化…</span>
                    </div>
                  ) : (
                    '开始占卜'
                  )}
                </button>
              </form>
            ) : (
              // 占卜结果
              <div className="space-y-6">
                <div className="bg-gradient-to-r from-primary-50 to-mystic-50 p-6 rounded-xl">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0">
                      <h3 className="text-lg font-semibold text-mystic-900">你的问题</h3>
                      <p className="text-mystic-700 mt-1 break-words">
                        {displayQuestion || '塔罗占卜'}
                      </p>
                    </div>
                    {reading?.created_at && (
                      <div className="flex items-center space-x-2 text-mystic-500 shrink-0">
                        <Clock className="w-5 h-5" />
                        <span className="text-sm">
                          {new Date(reading.created_at).toLocaleString()}
                        </span>
                      </div>
                    )}
                  </div>
                </div>

                {/* 抽牌结果 */}
                {displayCards.length > 0 ? (
                  <div>
                    <h3 className="text-lg font-semibold text-mystic-900 mb-4">抽牌结果</h3>
                    <div className="grid gap-6">
                      {displayCards.map((card, index) => (
                        <TarotCard
                          key={`${card.card_id ?? 'live'}-${card.position ?? index}`}
                          card={card}
                          position={card.position ?? index}
                          spreadType={displaySpreadType}
                          readingId={reading?.reading_id ?? null}
                        />
                      ))}
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-center gap-3 py-10 text-mystic-500">
                    <Loader2 className="w-5 h-5 animate-spin" />
                    <span>正在抽牌…</span>
                  </div>
                )}

                {/* 解读进度：牌已抽出、逐张解读进行中 */}
                {phase === 'drawing' && liveCards.length > 0 && (
                  <div className="flex items-start gap-3 rounded-xl border border-primary-200 bg-primary-50 p-4">
                    <Loader2 className="w-5 h-5 animate-spin text-primary-600 flex-shrink-0 mt-0.5" />
                    <div>
                      <p className="font-medium text-primary-900">牌已抽出，正在逐张解读…</p>
                      <p className="text-sm text-primary-700 mt-1">
                        {displaySpreadType === 'three'
                          ? '三张牌阵需要依次解读三张牌，大约需要 20-40 秒。'
                          : '大约需要 10-20 秒。'}
                      </p>
                    </div>
                  </div>
                )}

                {/* 整体解读（流式） */}
                {streamText && (
                  <div className="bg-mystic-50 rounded-xl p-6">
                    <div className="flex items-center gap-2 mb-3">
                      <Sparkles className="w-5 h-5 text-primary-500" />
                      <h3 className="text-lg font-semibold text-mystic-900">整体解读</h3>
                      {isBusy && <Loader2 className="w-4 h-4 animate-spin text-mystic-400" />}
                    </div>
                    <div className="text-mystic-700 leading-relaxed">
                      <Markdown content={streamText} />
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* 右侧：信息面板 */}
        <div className="space-y-6">
          {/* 功能特点 */}
          <div className="bg-white rounded-2xl shadow-xl p-6">
            <h3 className="text-lg font-semibold text-mystic-900 mb-4">功能特点</h3>
            <div className="space-y-4">
              {FEATURES.map((feature, i) => (
                <div key={i} className="flex items-start space-x-3">
                  <div className="p-2 bg-primary-100 rounded-lg">
                    <feature.icon className="w-5 h-5 text-primary-600" />
                  </div>
                  <div>
                    <div className="font-medium text-mystic-900">{feature.title}</div>
                    <div className="text-sm text-mystic-500 mt-1">{feature.desc}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* 占卜历史 */}
          <ReadingHistory
            refreshKey={historyVersion}
            onSelect={handleSelectHistory}
            disabled={isBusy}
          />
        </div>
      </div>
    </div>
  )
}
