#!/usr/bin/env bash
# 隔离与认证验收脚本（对应技术文档 §11.1 / §11.2）。
#
# 需要后端已在运行（默认 localhost:8000）。
# 脚本会创建两个临时账号 alice/bob，验证完毕后连同其数据一并删除。
#
# 用法：
#   cd backend-go && ./scripts/acceptance/isolation_test.sh
#   BASE=http://localhost:8000/api/v1 ./scripts/acceptance/isolation_test.sh
set -u

BASE="${BASE:-http://localhost:8000/api/v1}"
cd "$(dirname "$0")/../.." || exit 1

# 必须在任何 psql_run 之前加载 .env：
# 否则 MYSQL_* 未定义，读回空串，「空 == 空」会让归属断言蒙对通过（假阳性，
# 比没有这条断言更糟——它会让人以为验过了）。
cd_backend_env() { set -a; . ./.env; set +a; }
cd_backend_env

PASS=0
FAIL=0

# 期望值断言：expect <期望状态码> <描述> <curl 参数...>
expect() {
  local want="$1" desc="$2"; shift 2
  local got
  got=$(curl -s -o /dev/null -w '%{http_code}' "$@")
  if [ "$got" = "$want" ]; then
    printf '  \033[32mPASS\033[0m %-52s %s\n' "$desc" "$got"
    PASS=$((PASS + 1))
  else
    printf '  \033[31mFAIL\033[0m %-52s got=%s want=%s\n' "$desc" "$got" "$want"
    FAIL=$((FAIL + 1))
  fi
}

login() {
  curl -s -i -X POST "$BASE/auth/login" -H 'Content-Type: application/json' \
    -d "{\"username\":\"$1\",\"password\":\"$2\"}" \
    | grep -i '^set-cookie' | sed 's/.*tarot_session=\([^;]*\).*/\1/'
}

# 从 PATH 中取配置，直接操作数据库做清理与归属核对
psql_run() {
  mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DB" -N -e "$1" 2>/dev/null
}

cleanup() {
  cd_backend_env
  psql_run "
    DELETE rc FROM reading_cards rc JOIN readings r ON rc.reading_id=r.id
      WHERE r.owner_id IN (SELECT id FROM users WHERE username IN ('alice','bob'));
    DELETE FROM readings WHERE owner_id IN (SELECT id FROM users WHERE username IN ('alice','bob'));
    DELETE m FROM messages m JOIN conversations c ON m.conversation_id=c.id
      WHERE c.owner_id IN (SELECT id FROM users WHERE username IN ('alice','bob'));
    DELETE FROM conversations WHERE owner_id IN (SELECT id FROM users WHERE username IN ('alice','bob'));
    DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE username IN ('alice','bob'));
    DELETE FROM users WHERE username IN ('alice','bob');
  " >/dev/null
  echo "已清理临时账号 alice / bob 及其数据"
}
trap cleanup EXIT

echo "=== 准备临时账号 ==="
go run scripts/admin_user/main.go create --username alice --password 'Alice-Acceptance-Pw1' >/dev/null 2>&1
go run scripts/admin_user/main.go create --username bob   --password 'Bob-Acceptance-Pw1'   >/dev/null 2>&1
ALICE=$(login alice 'Alice-Acceptance-Pw1')
BOB=$(login bob 'Bob-Acceptance-Pw1')
AC="tarot_session=$ALICE"
BC="tarot_session=$BOB"
echo "alice / bob 已登录"

ACONV=$(curl -s -b "$AC" -X POST -H 'Content-Type: application/json' -d '{"channel":"chat"}' "$BASE/conversations" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
BCONV=$(curl -s -b "$BC" -X POST -H 'Content-Type: application/json' -d '{"channel":"chat"}' "$BASE/conversations" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')
echo "alice conversation=$ACONV  bob conversation=$BCONV"

echo
echo "=== §11.1 认证边界 ==="
expect 401 "无 Cookie 访问 /auth/me"                 "$BASE/auth/me"
expect 401 "无 Cookie 访问 /conversations"          "$BASE/conversations"
expect 401 "伪造 X-User-Id 头（旧机制应已失效）"     -H 'X-User-Id: 00000000-0000-0000-0000-000000000000' "$BASE/conversations"
expect 401 "错误口令登录"                           -X POST -H 'Content-Type: application/json' -d '{"username":"alice","password":"wrong"}' "$BASE/auth/login"
expect 401 "不存在的用户（文案应与口令错误一致）"    -X POST -H 'Content-Type: application/json' -d '{"username":"nosuchuser","password":"wrong"}' "$BASE/auth/login"
expect 200 "合法登录"                               -X POST -H 'Content-Type: application/json' -d '{"username":"alice","password":"Alice-Acceptance-Pw1"}' "$BASE/auth/login"
expect 200 "本人 /auth/me"                          -b "$AC" "$BASE/auth/me"
expect 404 "已删除的 /user/init"                    -X POST -b "$AC" "$BASE/user/init"

echo
echo "=== §11.2 会话隔离 ==="
expect 200 "alice 读自己的会话"                     -b "$AC" "$BASE/conversations/$ACONV"
expect 404 "bob 读 alice 的会话"                    -b "$BC" "$BASE/conversations/$ACONV"
expect 404 "bob 读 alice 的会话消息"                -b "$BC" "$BASE/conversations/$ACONV/messages"
expect 404 "bob 改 alice 的会话标题"                -X PATCH -b "$BC" -H 'Content-Type: application/json' -d '{"title":"hijack"}' "$BASE/conversations/$ACONV/title"
expect 404 "bob 删 alice 的会话"                    -X DELETE -b "$BC" "$BASE/conversations/$ACONV"
expect 404 "bob 往 alice 的会话发消息 (SSE 预检)"    -X POST -b "$BC" -H 'Content-Type: application/json' -d '{"content":"x"}' "$BASE/conversations/$ACONV/messages"
expect 404 "bob 在 alice 的会话里占卜 (SSE 预检)"    -X POST -b "$BC" -H 'Content-Type: application/json' -d '{"question":"x","spread_type":"single"}' "$BASE/conversations/$ACONV/tarot"
expect 404 "不存在的会话 ID"                        -b "$AC" "$BASE/conversations/99999999"

echo
echo "=== §11.2 占卜记录隔离（V1：原 IDOR 点）==="
RID=$(curl -s -b "$AC" -X POST -H 'Content-Type: application/json' \
  -d '{"question":"acceptance check","spread_type":"single"}' "$BASE/readings/" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["reading_id"])' 2>/dev/null)
if [ -n "${RID:-}" ]; then
  echo "alice 创建 reading id=$RID"
  OWNER=$(psql_run "SELECT owner_id FROM readings WHERE id=$RID;")
  ALICE_ID=$(psql_run "SELECT id FROM users WHERE username='alice';")
  # 两个值都必须非空才比较：读不到时「空 == 空」会假通过
  if [ -n "$OWNER" ] && [ -n "$ALICE_ID" ] && [ "$OWNER" = "$ALICE_ID" ]; then
    printf '  \033[32mPASS\033[0m %-52s owner_id=%s\n' "新占卜记录归属正确" "$OWNER"
    PASS=$((PASS + 1))
  else
    printf '  \033[31mFAIL\033[0m %-52s owner_id=%s want=%s\n' "新占卜记录归属正确" "${OWNER:-<读取失败>}" "${ALICE_ID:-<读取失败>}"
    FAIL=$((FAIL + 1))
  fi
  expect 200 "alice 读自己的占卜"           -b "$AC" "$BASE/readings/$RID"
  expect 404 "bob 读 alice 的占卜（IDOR）"  -b "$BC" "$BASE/readings/$RID"
  expect 404 "bob 传 scope=all 应被忽略"    -b "$BC" "$BASE/readings/$RID?scope=all"
  expect 401 "未登录读占卜"                 "$BASE/readings/$RID"
else
  printf '  \033[31mFAIL\033[0m %-52s (创建失败，跳过)\n' "alice 创建占卜记录"
  FAIL=$((FAIL + 1))
fi

echo
echo "=== §11.2 管理员默认不越界 ==="
ADMIN=$(login admin "${ADMIN_PASSWORD:-123456}")
if [ -n "$ADMIN" ]; then
  expect 404 "admin 默认读他人占卜（只应看到自己的）" -b "tarot_session=$ADMIN" "$BASE/readings/$RID"
  expect 200 "admin 加 ?scope=all 可读"               -b "tarot_session=$ADMIN" "$BASE/readings/$RID?scope=all"
else
  echo "  (admin 登录失败，跳过管理员用例；如已改口令请设 ADMIN_PASSWORD)"
fi

echo
echo "=== 结果 ==="
printf '  通过 %d，失败 %d\n' "$PASS" "$FAIL"
echo
echo "注：reading 用例会调用 LLM，耗时约 10-20s。"
[ "$FAIL" -eq 0 ] || exit 1
