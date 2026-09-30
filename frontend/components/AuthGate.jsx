'use client'

import { useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { getMe } from '../lib/auth'

/**
 * 客户端认证守卫。
 *
 * 为什么不用 Next middleware 做服务端守卫：
 *   1. 现有页面全部是 'use client'（app/page.jsx:1、app/chat/page.jsx:1）；
 *   2. 会话有效性只能由后端判定（/auth/me）。middleware 里最多只能看
 *      Cookie 是否存在，伪造一个同名 Cookie 就能绕过，等于给了虚假的安全感。
 *
 * 代价是未登录时会有一瞬间已登录界面的闪现。对本地生产可接受；
 * 若要消除，可加一个服务端 layout 做 Cookie 存在性预检再 redirect。
 */
export default function AuthGate({ children }) {
  const [user, setUser] = useState(null)
  const [checking, setChecking] = useState(true)

  useEffect(() => {
    let cancelled = false

    ;(async () => {
      try {
        const data = await getMe()
        if (!cancelled) setUser(data?.user ?? null)
      } catch (err) {
        // 401 时 apiFetch 已经跳转登录页并标记 handled，这里什么都不用做；
        // 其他错误（比如后端没起来）也不该无限转圈，标记为无用户即可。
        if (!cancelled && !err.handled) setUser(null)
      } finally {
        if (!cancelled) setChecking(false)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [])

  if (checking) {
    return (
      <div className="flex flex-col items-center justify-center py-24 text-mystic-500">
        <Loader2 className="w-6 h-6 animate-spin mb-3" />
        <p className="text-sm">正在校验登录状态…</p>
      </div>
    )
  }

  // getMe 失败且未被 apiFetch 处理（例如后端不可达）时，给出明确提示而不是空白页。
  if (!user) {
    return (
      <div className="max-w-md mx-auto mt-16 text-center">
        <p className="text-mystic-700">无法获取登录状态，请确认后端服务已启动。</p>
        <button
          type="button"
          onClick={() => window.location.reload()}
          className="mt-4 px-4 py-2 rounded-lg bg-primary-600 hover:bg-primary-700 text-white text-sm"
        >
          重新加载
        </button>
      </div>
    )
  }

  return children
}
