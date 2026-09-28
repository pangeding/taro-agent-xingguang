'use client'

import { useState, useEffect } from 'react'
import { RotateCw, AlertCircle, Sparkles, Loader2 } from 'lucide-react'
import Markdown from './Markdown'
import { apiFetch } from '../lib/api'

// 位置语义用显式映射表，不再依赖展示文案做判断
const POSITION_NAMES = {
  single: ['当前问题核心'],
  three: ['过去', '现在', '未来'],
}

const POSITION_TENSE = {
  过去: '已经经历',
  现在: '正在面对',
  未来: '将要面对',
}

function resolvePositionName(spreadType, position) {
  return POSITION_NAMES[spreadType]?.[position] ?? `第 ${position + 1} 张`
}

const TarotCard = ({ card, position, spreadType, readingId }) => {
  const [isFlipped, setIsFlipped] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [interpretation, setInterpretation] = useState(card.interpretation || '')
  const [refreshError, setRefreshError] = useState('')

  const positionName = resolvePositionName(spreadType, position)
  const tense = POSITION_TENSE[positionName]
  const meaning = card.is_reversed ? card.meaning_reversed : card.meaning_upright
  const hasCardDetail = Boolean(meaning || card.description || card.keywords)

  const keywordList = (card.keywords || '')
    .split(/[,，、]/)
    .map((k) => k.trim())
    .filter(Boolean)

  // 父组件加载到完整数据后（GET /readings/:id）以父级数据为准
  useEffect(() => {
    if (card.interpretation) {
      setInterpretation(card.interpretation)
    }
  }, [card.interpretation])

  // 重新拉取该条占卜记录，取服务端最新解读（不再用字面量覆盖）
  const handleRefresh = async () => {
    if (!readingId || isLoading) return
    setIsLoading(true)
    setRefreshError('')
    try {
      const data = await apiFetch(`/readings/${readingId}`)
      const fresh = data?.cards?.find((c) => c.position === position)
      if (fresh?.interpretation) {
        setInterpretation(fresh.interpretation)
      } else {
        setRefreshError('服务端暂无可用的解读内容')
      }
    } catch (err) {
      setRefreshError(err.message || '刷新失败，请稍后重试')
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
                  {positionName}
                  {tense ? ` · ${tense}` : ''}
                  {' · '}
                  {card.is_reversed ? '挑战与反思' : '机遇与启示'}
                </p>
              </div>
            </div>
          </div>
          <button
            onClick={() => setIsFlipped(!isFlipped)}
            className="p-2 text-mystic-500 hover:text-primary-600 hover:bg-mystic-50 rounded-lg transition-colors"
            title={isFlipped ? '查看AI解读' : '查看牌面信息'}
          >
            <RotateCw className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* 牌面内容 */}
      <div className="p-6">
        {isFlipped ? (
          // 牌面基础信息（取自数据库中的真实牌义）
          <div className="space-y-4">
            {!hasCardDetail ? (
              <p className="text-sm text-mystic-500">牌面资料加载中…</p>
            ) : (
              <>
                <div className="bg-mystic-50 rounded-xl p-4">
                  <h4 className="font-medium text-mystic-900 mb-2">
                    {card.is_reversed ? '逆位含义' : '正位含义'}
                  </h4>
                  <p className="text-mystic-700">{meaning || '暂无'}</p>
                  {keywordList.length > 0 && (
                    <div className="mt-3 flex flex-wrap gap-2">
                      {keywordList.map((k) => (
                        <span key={k} className="px-2 py-0.5 text-xs bg-white text-mystic-600 rounded-full border border-mystic-200">
                          {k}
                        </span>
                      ))}
                    </div>
                  )}
                </div>

                {(card.arcana_type || card.element || card.zodiac_sign || card.suit || card.number != null) && (
                  <dl className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-sm">
                    {[
                      ['类型', card.arcana_type === 'major' ? '大阿尔卡纳' : card.arcana_type === 'minor' ? '小阿尔卡纳' : card.arcana_type],
                      ['花色', card.suit],
                      ['数字', card.number != null ? String(card.number) : ''],
                      ['元素', card.element],
                      ['星座', card.zodiac_sign],
                    ]
                      .filter(([, v]) => v)
                      .map(([label, value]) => (
                        <div key={label} className="rounded-lg bg-mystic-50 px-3 py-2">
                          <dt className="text-xs text-mystic-500">{label}</dt>
                          <dd className="text-mystic-800">{value}</dd>
                        </div>
                      ))}
                  </dl>
                )}

                {card.description && (
                  <div>
                    <h4 className="font-medium text-mystic-900 mb-2">牌面描述</h4>
                    <p className="text-mystic-700 leading-relaxed">{card.description}</p>
                  </div>
                )}
              </>
            )}
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
                disabled={!readingId || isLoading}
                className="text-sm text-primary-600 hover:text-primary-700 flex items-center space-x-1 disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {isLoading ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <RotateCw className="w-4 h-4" />
                )}
                <span>刷新解读</span>
              </button>
            </div>

            {refreshError && (
              <p className="text-sm text-red-500">{refreshError}</p>
            )}

            {interpretation ? (
              <div className="text-mystic-700 leading-relaxed">
                <Markdown content={interpretation} />
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
                        这张牌的解读正在生成中…
                      </p>
                      <p className="text-sm text-yellow-700 mt-1">
                        解读完成后会自动显示在这里，无需刷新。
                      </p>
                    </div>
                  </div>
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
