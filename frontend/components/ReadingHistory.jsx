'use client'

import { useState, useEffect } from 'react'
import { History, Calendar, Eye, Trash2 } from 'lucide-react'
import axios from 'axios'

const ReadingHistory = ({ sessionId }) => {
  const [readings, setReadings] = useState([])
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    if (sessionId) {
      fetchReadings()
    }
  }, [sessionId])

  const fetchReadings = async () => {
    setIsLoading(true)
    try {
      // 这里应该调用API获取该session的占卜历史
      // 暂时使用模拟数据
      await new Promise(resolve => setTimeout(resolve, 500))
      setReadings([
        {
          id: 1,
          question: '工作发展前景如何？',
          spread_type: 'single',
          created_at: '2024-01-15T10:30:00',
          cards_count: 1,
        },
        {
          id: 2,
          question: '感情关系的未来走向',
          spread_type: 'three',
          created_at: '2024-01-14T15:45:00',
          cards_count: 3,
        },
      ])
    } catch (error) {
      console.error('获取占卜历史失败:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const formatDate = (dateString) => {
    const date = new Date(dateString)
    return date.toLocaleDateString('zh-CN', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const handleViewReading = (readingId) => {
    // 查看占卜详情
    console.log('查看占卜:', readingId)
  }

  const handleDeleteReading = (readingId) => {
    // 删除占卜记录
    setReadings(readings.filter(r => r.id !== readingId))
  }

  if (!sessionId) {
    return (
      <div className="bg-white rounded-2xl shadow-xl p-6">
        <div className="flex items-center space-x-3 mb-4">
          <History className="w-6 h-6 text-mystic-600" />
          <h3 className="text-lg font-semibold text-mystic-900">占卜历史</h3>
        </div>
        <div className="text-center py-8">
          <Calendar className="w-12 h-12 text-mystic-300 mx-auto mb-3" />
          <p className="text-mystic-500">开始你的第一次占卜，历史记录将在这里显示</p>
        </div>
      </div>
    )
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
          className="text-sm text-primary-600 hover:text-primary-700"
          disabled={isLoading}
        >
          {isLoading ? '刷新中...' : '刷新'}
        </button>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {[1, 2].map(i => (
            <div key={i} className="animate-pulse">
              <div className="h-16 bg-mystic-200 rounded-xl"></div>
            </div>
          ))}
        </div>
      ) : readings.length === 0 ? (
        <div className="text-center py-8">
          <Calendar className="w-12 h-12 text-mystic-300 mx-auto mb-3" />
          <p className="text-mystic-500">暂无占卜记录</p>
        </div>
      ) : (
        <div className="space-y-3">
          {readings.map(reading => (
            <div
              key={reading.id}
              className="group hover:bg-mystic-50 border border-mystic-200 rounded-xl p-4 transition-colors"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-mystic-900 truncate">
                    {reading.question}
                  </p>
                  <div className="flex items-center space-x-3 mt-2">
                    <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-mystic-100 text-mystic-700">
                      {reading.spread_type === 'single' ? '单张牌阵' : '三张牌阵'}
                    </span>
                    <span className="text-xs text-mystic-500">
                      {formatDate(reading.created_at)}
                    </span>
                    <span className="text-xs text-mystic-500">
                      {reading.cards_count}张牌
                    </span>
                  </div>
                </div>
                <div className="flex items-center space-x-1 ml-2 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    onClick={() => handleViewReading(reading.id)}
                    className="p-1.5 text-mystic-500 hover:text-primary-600 hover:bg-mystic-100 rounded-lg"
                    title="查看详情"
                  >
                    <Eye className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => handleDeleteReading(reading.id)}
                    className="p-1.5 text-mystic-500 hover:text-red-600 hover:bg-red-50 rounded-lg"
                    title="删除记录"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {readings.length > 0 && (
        <div className="mt-4 pt-4 border-t border-mystic-100">
          <p className="text-sm text-mystic-500 text-center">
            共 {readings.length} 条占卜记录
          </p>
        </div>
      )}
    </div>
  )
}

export default ReadingHistory