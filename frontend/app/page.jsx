'use client'

import { useState } from 'react'
import axios from 'axios'
import { Sparkles, Clock, Star, HelpCircle, RotateCcw } from 'lucide-react'
import TarotCard from '../components/TarotCard'
import ReadingHistory from '../components/ReadingHistory'

export default function Home() {
  const [question, setQuestion] = useState('')
  const [spreadType, setSpreadType] = useState('single')
  const [isLoading, setIsLoading] = useState(false)
  const [currentReading, setCurrentReading] = useState(null)
  const [sessionId, setSessionId] = useState(null)
  const [error, setError] = useState('')

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!question.trim()) {
      setError('请输入你的问题')
      return
    }

    setIsLoading(true)
    setError('')

    try {
      const response = await axios.post('http://localhost:8000/api/v1/readings/', {
        question,
        spread_type: spreadType,
        session_id: sessionId,
      })

      setCurrentReading(response.data)
      setSessionId(response.data.session_id)
      setIsLoading(false)
    } catch (err) {
      console.error('占卜失败:', err)
      setError('占卜失败，请稍后重试')
      setIsLoading(false)
    }
  }

  const handleNewReading = () => {
    setCurrentReading(null)
    setQuestion('')
  }

  const exampleQuestions = [
    '我最近的工作运势如何？',
    '应该如何改善人际关系？',
    '下一步的人生方向是什么？',
    '这段感情的未来发展会怎样？',
    '近期有什么需要注意的事项？',
  ]

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
        {/* 左侧：占卜表单 */}
        <div className="lg:col-span-2 space-y-6">
          {/* 占卜卡片 */}
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
                onClick={handleNewReading}
                className="flex items-center space-x-2 text-mystic-600 hover:text-primary-600"
              >
                <RotateCcw className="w-5 h-5" />
                <span>重新开始</span>
              </button>
            </div>

            {!currentReading ? (
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
                    disabled={isLoading}
                  />
                  {error && <p className="mt-2 text-red-500 text-sm">{error}</p>}

                  {/* 示例问题 */}
                  <div className="mt-4">
                    <p className="text-sm text-mystic-500 mb-2">不知道问什么？试试这些问题：</p>
                    <div className="flex flex-wrap gap-2">
                      {exampleQuestions.map((q, i) => (
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
                    {[
                      { id: 'single', name: '单张牌阵', desc: '快速洞察问题核心' },
                      { id: 'three', name: '三张牌阵', desc: '过去、现在、未来' },
                    ].map((spread) => (
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
                  disabled={isLoading}
                  className="w-full py-4 bg-gradient-mystic text-white font-semibold rounded-xl hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isLoading ? (
                    <div className="flex items-center justify-center space-x-2">
                      <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
                      <span>正在连接AI进行占卜...</span>
                    </div>
                  ) : (
                    '开始占卜'
                  )}
                </button>
              </form>
            ) : (
              // 占卜结果显示
              <div className="space-y-6">
                <div className="bg-gradient-to-r from-primary-50 to-mystic-50 p-6 rounded-xl">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="text-lg font-semibold text-mystic-900">你的问题</h3>
                      <p className="text-mystic-700 mt-1">{currentReading.question}</p>
                    </div>
                    <div className="flex items-center space-x-2 text-mystic-500">
                      <Clock className="w-5 h-5" />
                      <span>{new Date(currentReading.created_at).toLocaleString()}</span>
                    </div>
                  </div>
                </div>

                {/* 抽牌结果 */}
                <div>
                  <h3 className="text-lg font-semibold text-mystic-900 mb-4">抽牌结果</h3>
                  <div className="grid gap-6">
                    {currentReading.cards.map((card, index) => (
                      <TarotCard
                        key={index}
                        card={card}
                        position={index}
                        spreadType={currentReading.spread_type}
                      />
                    ))}
                  </div>
                </div>

                {/* AI解读状态 */}
                <div className="bg-blue-50 border border-blue-200 rounded-xl p-4">
                  <div className="flex items-center space-x-3">
                    <div className="p-2 bg-blue-100 rounded-lg">
                      <Sparkles className="w-5 h-5 text-blue-600" />
                    </div>
                    <div>
                      <p className="font-medium text-blue-900">AI正在深度解读中...</p>
                      <p className="text-sm text-blue-700 mt-1">
                        塔罗牌解读需要一些时间，请耐心等待。解读完成后会自动更新。
                      </p>
                    </div>
                  </div>
                </div>
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
              {[
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
              ].map((feature, i) => (
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
          <ReadingHistory sessionId={sessionId} />
        </div>
      </div>
    </div>
  )
}