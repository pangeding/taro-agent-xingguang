'use client'

import { useState, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Sparkles, Loader2, LogIn } from 'lucide-react'
import { login } from '../../lib/auth'
import { NEXT_PARAM } from '../../lib/api'

function LoginForm() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const nextPath = searchParams.get(NEXT_PARAM) || '/'

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (busy) return
    if (!username.trim() || !password) {
      setError('请输入用户名和密码')
      return
    }

    setError('')
    setBusy(true)
    try {
      const result = await login(username.trim(), password)
      // 后端在 prod 下可能要求先改密，此时所有业务接口都会被 403 拦下，
      // 直接送到改密页比让用户撞 403 更清楚。
      if (result?.user?.must_change_password) {
        router.replace('/account/password')
        return
      }
      router.replace(nextPath)
    } catch (err) {
      // login() 传了 skipAuthRedirect，所以这里是后端原文（如「用户名或密码错误」），
      // 不会被 apiFetch 替换成「登录状态已失效」。
      setError(err.message || '登录失败')
      setBusy(false)
    }
  }

  return (
    <div className="max-w-md mx-auto mt-12">
      <div className="bg-white/90 backdrop-blur-sm rounded-2xl shadow-xl border border-mystic-200 p-8">
        <div className="text-center mb-8">
          <div className="w-14 h-14 bg-gradient-mystic rounded-full mx-auto mb-4 flex items-center justify-center">
            <Sparkles className="w-7 h-7 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-mystic-900">登录星光塔罗</h1>
          <p className="text-sm text-mystic-600 mt-2">请使用管理员分配的账号登录</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label htmlFor="username" className="block text-sm font-medium text-mystic-800 mb-1.5">
              用户名
            </label>
            <input
              id="username"
              type="text"
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              disabled={busy}
              className="w-full px-4 py-2.5 rounded-lg border border-mystic-300 focus:border-primary-500 focus:ring-2 focus:ring-primary-200 outline-none transition disabled:opacity-60"
              placeholder="请输入用户名"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-mystic-800 mb-1.5">
              密码
            </label>
            <input
              id="password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={busy}
              className="w-full px-4 py-2.5 rounded-lg border border-mystic-300 focus:border-primary-500 focus:ring-2 focus:ring-primary-200 outline-none transition disabled:opacity-60"
              placeholder="请输入密码"
            />
          </div>

          {error && (
            <div className="px-4 py-2.5 rounded-lg bg-red-50 border border-red-200 text-sm text-red-700">
              {error}
            </div>
          )}

          <button
            type="submit"
            disabled={busy}
            className="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-primary-600 hover:bg-primary-700 disabled:bg-primary-300 text-white font-medium transition"
          >
            {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <LogIn className="w-4 h-4" />}
            {busy ? '登录中…' : '登录'}
          </button>
        </form>
      </div>

      <p className="text-center text-xs text-mystic-500 mt-6">
        没有账号？本系统不开放自助注册，请联系管理员创建。
      </p>
    </div>
  )
}

export default function LoginPage() {
  // useSearchParams 需要 Suspense 边界，否则 next build 预渲染阶段报错。
  return (
    <Suspense
      fallback={
        <div className="flex justify-center mt-20">
          <Loader2 className="w-6 h-6 animate-spin text-mystic-400" />
        </div>
      }
    >
      <LoginForm />
    </Suspense>
  )
}
