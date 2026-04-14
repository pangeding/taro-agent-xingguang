# 前后端通信流程

## 概述

本文档解释了 Next.js/React 前端如何与 FastAPI 后端在我们的应用程序中进行通信。该通信遵循标准的客户端-服务器 REST API 模式，并配置了适当的 CORS（跨域资源共享）。

## 请求流程

1. **前端发起请求**
   - Next.js 中的 React 组件使用 `fetch` 或 Axios 发起 API 请求
   - 请求被发送到配置好的后端端点

2. **环境配置**
   ```env
   # .env.local
   NEXT_PUBLIC_API_BASE_URL=http://localhost:8000
   ```

3. **前端请求示例**
   ```javascript
   // 使用 fetch
   const response = await fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL}/api/users`, {
     method: 'GET',
     headers: {
       'Content-Type': 'application/json',
       'Authorization': `Bearer ${token}`
     }
   });

   // 使用 Axios（推荐）
   const response = await axios.get('/api/users', {
     baseURL: process.env.NEXT_PUBLIC_API_BASE_URL,
     withCredentials: true
   });
   ```

4. **后端配置 (FastAPI)**
   ```python
   from fastapi import FastAPI
   from fastapi.middleware.cors import CORSMiddleware

   app = FastAPI()

   app.add_middleware(
       CORSMiddleware,
       allow_origins=["http://localhost:3000"],
       allow_credentials=True,
       allow_methods=["*"],
       allow_headers=["*"],
   )

   @app.get("/api/users")
   async def get_users():
       return {"data": ["user1", "user2"]}
   ```

## CORS 实现

启用跨域请求的关键配置：

- FastAPI 的 CORS 中间件中必须白名单前端源地址
- 处理凭证需要设置 `allow_credentials=True` 和适当的 cookie 配置
- 开发环境和生产环境的源地址应通过环境变量进行管理

## 常见问题

| 问题 | 解决方案 |
|-------|----------|
| 开发期间的 CORS 错误 | 验证 FastAPI 允许的源地址是否与 Next.js 开发服务器 URL 匹配 |
| 401 未授权 | 确保身份验证令牌已正确包含在请求头中 |
| 请求超时 | 检查后端服务器状态和网络连接 |

## 生产环境配置

在生产环境中，请求通常通过反向代理进行路由：

```
# Nginx 配置示例
location /api/ {
    proxy_pass http://fastapi-backend:8000;
}
```

这允许前端和后端共享同一个域名，从而避免 CORS 问题。
