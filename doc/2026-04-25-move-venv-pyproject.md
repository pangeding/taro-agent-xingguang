# 2026-04-25 技术文档：移动 .venv 和 pyproject.toml 到 backend 目录

## 执行指令清单
1. 创建 backend 目录（如不存在）
   `mkdir -p backend`

2. 移动 pyproject.toml
   `mv pyproject.toml backend/`

3. 移动 .venv 目录
   `mv .venv backend/`

## 强制校验规则
- 必须先执行 `ls` 验证 backend 目录是否存在
- 移动前必须确认根目录存在 pyproject.toml 和 .venv
- 禁止修改 Makefile 位置

## 执行序列要求
1. 创建目录 → 2. 移动配置文件 → 3. 移动虚拟环境

## 验证命令
```
ls backend/pyproject.toml
ls backend/.venv
```