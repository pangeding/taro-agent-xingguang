# 星光塔罗AI助手 - Makefile
# 基于 README 中的快速开始指南构建

.PHONY: help install-backend install-frontend install init-db import-cards backend frontend dev clean test

# 默认目标
help:
	@echo "星光塔罗AI助手 - 可用命令:"
	@echo ""
	@echo "安装和初始化:"
	@echo "  make install          - 安装后端和前端所有依赖"
	@echo "  make install-backend  - 仅安装后端依赖"
	@echo "  make install-frontend - 仅安装前端依赖"
	@echo "  make init-db          - 初始化数据库"
	@echo "  make import-cards     - 导入塔罗牌数据"
	@echo ""
	@echo "运行服务:"
	@echo "  make backend          - 启动后端服务器 (端口 8000)"
	@echo "  make frontend         - 启动前端开发服务器 (端口 3000)"
	@echo "  make dev              - 同时启动后端和前端"
	@echo ""
	@echo "其他:"
	@echo "  make clean            - 清理临时文件和依赖"
	@echo "  make test             - 运行测试"

# 安装所有依赖
install: install-backend install-frontend
	@echo "✅ 所有依赖安装完成"

# 安装后端依赖
install-backend:
	@echo "📦 安装后端依赖..."
	cd backend && uv sync
	@echo "✅ 后端依赖安装完成"

# 安装前端依赖
install-frontend:
	@echo "📦 安装前端依赖..."
	cd frontend && pnpm install
	@echo "✅ 前端依赖安装完成"

# 初始化数据库
init-db:
	@echo "🗄️  初始化数据库..."
	cd backend && python scripts/init_db.py
	@echo "✅ 数据库初始化完成"

# 导入塔罗牌数据
import-cards:
	@echo "🃏 导入塔罗牌数据..."
	cd backend && python scripts/import_cards.py
	@echo "✅ 塔罗牌数据导入完成"

# 启动后端服务器
backend:
	@echo "🚀 启动后端服务器..."
	@echo "📍 API 文档: http://localhost:8000/docs"
	cd backend && python run.py

# 启动前端开发服务器
frontend:
	@echo "🚀 启动前端开发服务器..."
	@echo "📍 访问地址: http://localhost:3000"
	cd frontend && pnpm dev

# 同时启动后端和前端（后台运行）
dev: 
	@echo "🚀 同时启动后端和前端服务..."
	@echo "📍 后端: http://localhost:8000/docs"
	@echo "📍 前端: http://localhost:3000"
	@echo ""
	@echo "提示: 使用 Ctrl+C 停止所有服务"
	@trap 'kill %1 %2 2>/dev/null' EXIT; \
	cd backend && python run.py & \
	cd frontend && pnpm dev & \
	wait

# 清理临时文件和依赖
clean:
	@echo "🧹 清理临时文件和依赖..."
	# 清理 Python 缓存
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete 2>/dev/null || true
	find . -type f -name "*.pyo" -delete 2>/dev/null || true
	# 清理数据库文件
	rm -f backend/taro.db 2>/dev/null || true
	# 清理前端构建文件
	rm -rf frontend/.next 2>/dev/null || true
	rm -rf frontend/node_modules 2>/dev/null || true
	# 清理后端虚拟环境
	rm -rf backend/.venv 2>/dev/null || true
	@echo "✅ 清理完成"

# 运行测试（预留）
test:
	@echo "🧪 运行测试..."
	@echo "测试功能待实现"
