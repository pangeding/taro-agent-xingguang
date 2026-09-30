'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { KeyRound, Loader2 } from 'lucide-react'
import { changePassword } from '../../../lib/auth'

/**
 * 强制改密页。
 *
 * 触发场景：APP_ENV=prod 且账号 must_change_password=true
 * （管理员通过 admin_user reset-password 重置口令后，或新建账号时指定 --must-change）。
 * 此时后端除 /auth/me、/auth/logout、/auth/password 外的所有接口都返回
 * 403 password_change_required，所以必须先完成这一步才能进入业务页面。
 */
export default function ChangePasswordPage() {
  const router = useRouter()

  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (busy) return

    if (!oldPassword || !newPassword) {
      setError('请填写原密码和新密码')
      return
    }
    if (newPassword !== confirmPassword) {
      setError('两次输入的新密码不一致')
      return
    }
    if (newPassword === oldPassword) {
      setError('新密码不能与原密码相同')
      return
    }

    setError('')
    setBusy(true)
    try {
      await changePassword(oldPassword, newPassword)
      router.replace('/')
    } catch (err) {
      if (err.handled) return
      setError(err.message || '修改失败')
      setBusy(false)
    }
  }

  return (
    <div className="max-w-md mx-auto mt-12">
      <div className="bg-white/90 backdrop-blur-sm rounded-2xl shadow-xl border border-mystic-200 p-8">
        <div className="text-center mb-8">
          <div className="w-14 h-14 bg-gradient-mystic rounded-full mx-auto mb-4 flex items-center justify-center">
            <KeyRound className="w-7 h-7 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-mystic-900">修改密码</h1>
          <p className="text-sm text-mystic-600 mt-2">
            当前账号需要先修改密码才能继续使用
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label htmlFor="old" className="block text-sm font-medium text-mystic-800 mb-1.5">
              原密码
            </label>
            <input
              id="old"
              type="password"
              autoComplete="current-password"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              disabled={busy}
              className="w-full px-4 py-2.5 rounded-lg border border-mystic-300 focus:border-primary-500 focus:ring-2 focus:ring-primary-200 outline-none transition disabled:opacity-60"
            />
          </div>

          <div>
            <label htmlFor="new" className="block text-sm font-medium text-mystic-800 mb-1.5">
              新密码
            </label>
            <input
              id="new"
              type="password"
              autoComplete="new-password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              disabled={busy}
              className="w-full px-4 py-2.5 rounded-lg border border-mystic-300 focus:border-primary-500 focus:ring-2 focus:ring-primary-200 outline-none transition disabled:opacity-60"
            />
            <p className="text-xs text-mystic-500 mt-1.5">
              不能是常见弱口令，也不能与原密码相同。
            </p>
          </div>

          <div>
            <label htmlFor="confirm" className="block text-sm font-medium text-mystic-800 mb-1.5">
              确认新密码
            </label>
            <input
              id="confirm"
              type="password"
              autoComplete="new-password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              disabled={busy}
              className="w-full px-4 py-2.5 rounded-lg border border-mystic-300 focus:border-primary-500 focus:ring-2 focus:ring-primary-200 outline-none transition disabled:opacity-60"
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
            {busy ? <Loader2 className="w-4 h-4 animate-spin" /> : <KeyRound className="w-4 h-4" />}
            {busy ? '提交中…' : '确认修改'}
          </button>
        </form>

        <p className="text-xs text-mystic-500 mt-6 text-center">
          修改成功后，其他设备上的登录状态会全部失效。
        </p>
      </div>
    </div>
  )
}
