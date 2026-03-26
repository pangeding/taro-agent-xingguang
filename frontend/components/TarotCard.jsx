'use client'

import { useState, useEffect } from 'react'
import { RotateCw, AlertCircle, Sparkles } from 'lucide-react'
import axios from 'axios'

const TarotCard = ({ card, position, spreadType }) => {
  const [isFlipped, setIsFlipped] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [interpretation, setInterpretation] = useState(card.interpretation)

  // 位置描述映射
  const positionDescriptions = {
    single: ['当前问题核心'],
    three: ['过去', '现在', '未来'],
  }

  const positionName = positionDescriptions[spreadType]?.[position] || `位置 ${position + 1}`

  // 检查解读是否已完成
  useEffect(() => {
    if (card.interpretation && !interpretation) {
      setInterpretation(card.interpretation)
    }
  }, [card.interpretation, interpretation])

  // 轮询获取解读结果
  useEffect(() => {
    if (!interpretation) {
      const interval = setInterval(async () => {
        try {
          // 这里应该调用API获取更新的解读
          // 暂时使用模拟
          setIsLoading(false)
        } catch (error) {
          console.error('获取解读失败:', error)
        }
      }, 5000)

      return () => clearInterval(interval)
    }
  }, [interpretation])

  const handleRefresh = async () => {
    setIsLoading(true)
    try {
      // 这里可以调用API重新生成解读
      await new Promise(resolve => setTimeout(resolve, 1000))
      setInterpretation('新的解读内容...')
    } catch (error) {
      console.error('刷新解读失败:', error)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="bg-white rounded-2xl shadow-lg overflow-hidden border border-mystic-200">
      {/* 牌面信息头部 */}
      <div className="p-6 border-b border-mystic-100">
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center space-x-3">
              <div className="flex-shrink-0">
                <div className="relative">
                  <div className={`w-12 h-16 rounded-lg ${card.is_reversed ? 'bg-red-100' : 'bg-primary-100'} flex items-center justify-center`}>
                    <div className={`text-lg font-bold ${card.is_reversed ? 'text-red-600' : 'text-primary-600'}`}>
                      {position + 1}
                    </div>
                  </div>
                  {card.is_reversed && (
                    <div className="absolute -top-1 -right-1">
                      <div className="w-6 h-6 bg-red-500 text-white rounded-full flex items-center justify-center">
                        <RotateCw className="w-3 h-3" />
                      </div>
                    </div>
                  )}
                </div>
              </div>
              <div>
                <div className="flex items-center space-x-2">
                  <h3 className="text-xl font-bold text-mystic-900">{card.name}</h3>
                  <span className={`px-2 py-0.5 text-xs font-medium rounded-full ${card.is_reversed ? 'bg-red-100 text-red-700' : 'bg-green-100 text-green-700'}`}>
                    {card.is_reversed ? '逆位' : '正位'}
                  </span>
                </div>
                <p className="text-mystic-600 mt-1">
                  {positionName} · {card.is_reversed ? '挑战与反思' : '机遇与启示'}
                </p>
              </div>
            </div>
          </div>
          <button
            onClick={() => setIsFlipped(!isFlipped)}
            className="p-2 text-mystic-500 hover:text-primary-600 hover:bg-mystic-50 rounded-lg transition-colors"
            title={isFlipped ? "查看解读" : "查看牌面信息"}
          >
            <RotateCw className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* 牌面内容 */}
      <div className="p-6">
        {isFlipped ? (
          // 牌面基础信息
          <div className="space-y-4">
            <div className="bg-mystic-50 rounded-xl p-4">
              <h4 className="font-medium text-mystic-900 mb-2">牌面含义</h4>
              <p className="text-mystic-700">
                {card.is_reversed ? '逆位表示挑战、阻碍或需要反思的方面' : '正位表示机遇、积极的发展方向'}
              </p>
            </div>
            <div>
              <h4 className="font-medium text-mystic-900 mb-2">象征意义</h4>
              <p className="text-mystic-700">
                这张牌在{positionName}的位置上，代表着你在当前问题中{' '}
                {positionName === '过去' ? '已经经历' : positionName === '现在' ? '正在面对' : '将要面对'}{' '}
                的情况。{card.is_reversed ? '逆位提醒你需要特别注意可能存在的挑战。' : '正位预示着积极的发展趋势。'}
              </p>
            </div>
          </div>
        ) : (
          // AI解读
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Sparkles className="w-5 h-5 text-primary-500" />
                <h4 className="font-medium text-mystic-900">AI深度解读</h4>
              </div>
              <button
                onClick={handleRefresh}
                disabled={isLoading}
                className="text-sm text-primary-600 hover:text-primary-700 flex items-center space-x-1"
              >
                <RotateCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
                <span>刷新解读</span>
              </button>
            </div>

            {interpretation ? (
              <div className="prose prose-sm max-w-none">
                <p className="text-mystic-700 leading-relaxed whitespace-pre-line">
                  {interpretation}
                </p>
              </div>
            ) : (
              <div className="space-y-4">
                {/* 加载状态 */}
                <div className="animate-pulse space-y-3">
                  <div className="h-4 bg-mystic-200 rounded"></div>
                  <div className="h-4 bg-mystic-200 rounded"></div>
                  <div className="h-4 bg-mystic-200 rounded w-3/4"></div>
                </div>

                {/* 提示信息 */}
                <div className="bg-yellow-50 border border-yellow-200 rounded-xl p-4">
                  <div className="flex items-start space-x-3">
                    <AlertCircle className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5" />
                    <div>
                      <p className="text-sm text-yellow-800">
                        AI正在深度解读这张牌的含义...
                      </p>
                      <p className="text-sm text-yellow-700 mt-1">
                        解读需要一些时间，请耐心等待。系统会自动更新解读结果。
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* 建议部分 */}
            {interpretation && (
              <div className="mt-6 pt-6 border-t border-mystic-100">
                <h4 className="font-medium text-mystic-900 mb-3">给你的建议</h4>
                <div className="bg-primary-50 rounded-xl p-4">
                  <p className="text-primary-800">
                    {card.is_reversed
                      ? '面对逆位的挑战，建议保持耐心与反思，寻找问题的根源。'
                      : '把握正位的机遇，积极行动，但也要保持谨慎与平衡。'}
                  </p>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

export default TarotCard