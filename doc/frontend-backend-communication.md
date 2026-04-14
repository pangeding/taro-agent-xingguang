# Frontend-Backend Communication Flow

## Overview

This document explains how the Next.js/React frontend communicates with the FastAPI backend in our application. The communication follows a standard client-server REST API pattern with proper CORS configuration.

## Request Flow

1. **Frontend Initiation**
   - React components in Next.js initiate API requests using `fetch` or Axios
   - Requests are sent to configured backend endpoints

2. **Environment Configuration**
   ```env
   # .env.local
   NEXT_PUBLIC_API_BASE_URL=http://localhost:8000
   ```

3. **Frontend Request Example**
   ```javascript
   // Using fetch
   const response = await fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL}/api/users`, {
     method: 'GET',
     headers: {
       'Content-Type': 'application/json',
       'Authorization': `Bearer ${token}`
     }
   });

   // Using Axios (recommended)
   const response = await axios.get('/api/users', {
     baseURL: process.env.NEXT_PUBLIC_API_BASE_URL,
     withCredentials: true
   });
   ```

4. **Backend Configuration (FastAPI)**
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

## CORS Implementation

Critical configuration to enable cross-origin requests:

- Frontend origin must be whitelisted in FastAPI's CORS middleware
- Credentials handling requires `allow_credentials=True` and proper cookie configuration
- Development vs production origins should be managed through environment variables

## Common Issues

| Issue | Solution |
|-------|----------|
| CORS errors during development | Verify FastAPI's allowed origins match Next.js dev server URL |
| 401 Unauthorized | Ensure authentication tokens are properly included in headers |
| Requests timing out | Check backend server status and network connectivity |

## Production Configuration

In production, requests are typically routed through a reverse proxy:

```
# Nginx configuration example
location /api/ {
    proxy_pass http://fastapi-backend:8000;
}
```

This allows the frontend and backend to share the same domain, avoiding CORS concerns.