'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { LogOut, KeyRound, User, Loader2 } from 'lucide-react'
import { getMe, logout, isAdmin } from '../lib/auth'

/**
 * 导航栏。从 app/layout.jsx 抽出并改为客户端组件，
 * 因为它需要读取当前登录用户并处理登出。
 *
 * 刻意不渲染任何内容（返回占位）当用户未登录：
 * 登录页/改密页也会走 RootLayout，此时不该显示「退出登录」。
 */
export default function NavBar() {
  const router = useRouter()
  const [user, setUser] = useState(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const data = await getMe()
        if (!cancelled) setUser(data?.user ?? null)
      } catch {
        // 未登录：apiFetch 已跳转登录页；这里保持 nav 不显示用户区即可。
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const handleLogout = async () => {
    setBusy(true)
    await logout()
    router.replace('/login')
  }

  return (
    <nav className="bg-white/80 backdrop-blur-sm border-b border-mystic-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-16">
          <a href="/" className="flex items-center space-x-2">
            <div className="w-8 h-8 bg-gradient-mystic rounded-full"></div>
            <span className="text-xl font-bold text-mystic-900">星光塔罗</span>
          </a>

          <div className="flex items-center gap-1 sm:gap-2">
            <a
              href="/"
              className="text-mystic-700 hover:text-primary-600 px-3 py-2 rounded-md text-sm font-medium"
            >
              首页占卜
            </a>
            <a
              href="/chat"
              className="text-mystic-700 hover:text-primary-600 px-3 py-2 rounded-md text-sm font-medium"
            >
              塔罗对话
            </a>

            {user && (
              <>
                <span className="hidden sm:flex items-center gap-1.5 text-mystic-700 text-sm px-2 border-l border-mystic-200 ml-1 pl-3">
                  <User className="w-4 h-4" />
                  <span className="font-medium">{user.display_name || user.username}</span>
                  {isAdmin(user) && (
                    <span className="text-xs px-1.5 py-0.5 rounded bg-primary-100 text-primary-700">
                      管理员
                    </span>
                  )}
                </span>
                <a
                  href="/account/password"
                  title="修改密码"
                  className="text-mystic-600 hover:text-primary-600 p-2 rounded-md"
                >
                  <KeyRound className="w-4 h-4" />
                </a>
                <button
                  type="button"
                  onClick={handleLogout}
                  disabled={busy}
                  title="退出登录"
                  className="text-mystic-600 hover:text-red-600 p-2 rounded-md disabled:opacity-50"
                >
                  {busy ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <LogOut className="w-4 h-4" />
                  )}
                </button>
              </>
            )}
          </div>
        </div>
      </div>
    </nav>
  )
}
