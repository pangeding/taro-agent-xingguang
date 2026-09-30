#!/usr/bin/env bash
# 生产保护验收脚本（对应技术文档 §11.4）。
#
# 每个用例前重建空的 tarot_prod_test 库，确保确实走到 L1（引导账号校验）——
# 否则 users 表非空会跳过 L1 直接进 L2，测出来的结论是假的。
#
# 用法：先临时把 .env 的 MYSQL_DB 改成 tarot_prod_test，再执行本脚本。
set -u
cd "$(dirname "$0")/.."
export GOCACHE=/home/pangding/ai/taro_agent/.gocache
export BIND_ADDR=127.0.0.1:8099
set -a; . ./.env; set +a

reset_db() {
  mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" \
    -e "DROP DATABASE IF EXISTS tarot_prod_test; CREATE DATABASE tarot_prod_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null
}

case_run() {
  local desc="$1"; shift
  reset_db
  local out
  out=$(env "$@" timeout 25 go run cmd/server/main.go 2>&1)
  local verdict reason
  if echo "$out" | grep -q "启动失败"; then
    verdict="拒绝启动"
    reason=$(echo "$out" | grep -m1 "启动失败" | sed 's/^[0-9\/: ]*//' | cut -c1-88)
  elif echo "$out" | grep -q "运行模式: prod"; then
    verdict="正常启动"
    reason=$(echo "$out" | grep -m1 "bootstrap" | sed 's/^[0-9\/: ]*//' | cut -c1-88)
  else
    verdict="未判定"
    reason=$(echo "$out" | tail -2 | tr '\n' ' ')
  fi
  printf '%-26s -> %-9s | %s\n' "$desc" "$verdict" "$reason"
}

echo "=== L1 引导账号校验（users 表为空，APP_ENV=prod）==="
case_run "空 BOOTSTRAP_*"       APP_ENV=prod
case_run "保留用户名 admin"     APP_ENV=prod BOOTSTRAP_ADMIN_USERNAME=admin BOOTSTRAP_ADMIN_PASSWORD='Str0ng-Passw0rd-2026'
case_run "保留用户名 root"      APP_ENV=prod BOOTSTRAP_ADMIN_USERNAME=root BOOTSTRAP_ADMIN_PASSWORD='Str0ng-Passw0rd-2026'
case_run "弱口令 123456"        APP_ENV=prod BOOTSTRAP_ADMIN_USERNAME=myuser BOOTSTRAP_ADMIN_PASSWORD='123456'
case_run "长度不足 7 位"        APP_ENV=prod BOOTSTRAP_ADMIN_USERNAME=myuser BOOTSTRAP_ADMIN_PASSWORD='Sh0rt-Pw'
case_run "口令==用户名"         APP_ENV=prod BOOTSTRAP_ADMIN_USERNAME=longusername12 BOOTSTRAP_ADMIN_PASSWORD='longusername12'
case_run "口令全同字符"         APP_ENV=prod BOOTSTRAP_ADMIN_USERNAME=myuser BOOTSTRAP_ADMIN_PASSWORD='aaaaaaaaaaaaaaa'
case_run "合规配置"             APP_ENV=prod BOOTSTRAP_ADMIN_USERNAME=myuser BOOTSTRAP_ADMIN_PASSWORD='Str0ng-Passw0rd-2026'

echo
echo "=== L2 存量弱口令扫描（users 表非空）==="
reset_db
# 先用 dev 模式建一个合规账号，再用 CLI 塞一个弱口令账号
env APP_ENV=dev timeout 25 go run cmd/server/main.go >/dev/null 2>&1 &
sleep 6
kill %1 2>/dev/null
wait 2>/dev/null
go run scripts/admin_user/main.go create --username weakuser --password 'password' 2>&1 | tail -1
out=$(env APP_ENV=prod BOOTSTRAP_ADMIN_USERNAME=otheruser BOOTSTRAP_ADMIN_PASSWORD='Str0ng-Passw0rd-2026' timeout 25 go run cmd/server/main.go 2>&1)
if echo "$out" | grep -q "启动失败"; then
  printf '%-26s -> %-9s | %s\n' "库中存在弱口令账号" "拒绝启动" "$(echo "$out" | grep -m1 '启动失败' | sed 's/^[0-9\/: ]*//' | cut -c1-88)"
else
  printf '%-26s -> %-9s | %s\n' "库中存在弱口令账号" "未拦截(FAIL)" "$(echo "$out" | grep -m1 '运行模式')"
fi

reset_db
