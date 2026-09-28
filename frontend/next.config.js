/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  skipTrailingSlashRedirect: true,

  // 必须关闭：SSE 走 /api 的 rewrite 代理时，Next 的 gzip 压缩会把整个流缓冲到结束才下发，
  // 而浏览器一定会发送 Accept-Encoding: gzip，结果是「逐字流式输出」在浏览器里完全失效
  // （curl 不带该头时反而是正常的，容易误判）。关闭后 SSE 才能逐块到达。
  // 生产环境如需压缩静态资源，应由 nginx/CDN 承担。
  compress: false,

  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://localhost:8000/api/:path*',
      },
    ]
  },
  images: {
    domains: ['localhost', '127.0.0.1'],
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**',
      },
    ],
  },
}

module.exports = nextConfig