# 2026-04-25 技术文档：移动 scripts 目录到 backend 目录

## 执行指令清单
1. 验证 backend 目录存在
   `ls backend`

2. 移动 scripts 目录
   `mv scripts backend/`

## 强制校验规则
- 必须确认根目录存在 scripts 目录
- 禁止修改 Makefile 位置

## 执行序列要求
1. 验证目录 → 2. 移动目录

## 验证命令
```
ls backend/scripts
```