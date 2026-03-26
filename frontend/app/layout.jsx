import { Inter } from 'next/font/google'
import './globals.css'

const inter = Inter({ subsets: ['latin'] })

export const metadata = {
  title: '星光塔罗AI助手',
  description: '结合传统塔罗牌智慧与现代人工智能技术的个性化占卜体验',
}

export default function RootLayout({ children }) {
  return (
    <html lang="zh-CN">
      <body className={`${inter.className} bg-gradient-to-br from-mystic-50 to-mystic-100`}>
        <div className="min-h-screen">
          {/* 导航栏 */}
          <nav className="bg-white/80 backdrop-blur-sm border-b border-mystic-200">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
              <div className="flex justify-between items-center h-16">
                <div className="flex items-center">
                  <div className="flex-shrink-0">
                    <div className="flex items-center space-x-2">
                      <div className="w-8 h-8 bg-gradient-mystic rounded-full"></div>
                      <span className="text-xl font-bold text-mystic-900">星光塔罗</span>
                    </div>
                  </div>
                </div>
                <div className="hidden md:block">
                  <div className="ml-10 flex items-baseline space-x-4">
                    <a href="/" className="text-mystic-700 hover:text-primary-600 px-3 py-2 rounded-md text-sm font-medium">
                      首页
                    </a>
                    <a href="#" className="text-mystic-700 hover:text-primary-600 px-3 py-2 rounded-md text-sm font-medium">
                      我的占卜
                    </a>
                    <a href="#" className="text-mystic-700 hover:text-primary-600 px-3 py-2 rounded-md text-sm font-medium">
                      塔罗知识
                    </a>
                    <a href="#" className="text-mystic-700 hover:text-primary-600 px-3 py-2 rounded-md text-sm font-medium">
                      关于我们
                    </a>
                  </div>
                </div>
              </div>
            </div>
          </nav>

          {/* 主要内容 */}
          <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            {children}
          </main>

          {/* 页脚 */}
          <footer className="bg-white/80 backdrop-blur-sm border-t border-mystic-200 mt-12">
            <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
              <div className="text-center text-mystic-600 text-sm">
                <p>© 2024 星光塔罗AI助手. 本产品仅供娱乐参考，请理性看待占卜结果。</p>
                <p className="mt-2">结合传统智慧与现代AI技术，为您提供个性化塔罗牌解读体验。</p>
              </div>
            </div>
          </footer>
        </div>
      </body>
    </html>
  )
}