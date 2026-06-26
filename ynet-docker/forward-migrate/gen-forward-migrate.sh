#!/usr/bin/env bash
#
# Copyright 2025 ynet-dev Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# =============================================================================
# gen-forward-migrate.sh — B：从当前代码权威 schema 生成"自包含、幂等、只增不删"的迁移 SQL
# 设计：docs/superpowers/specs/2026-06-26-prod-data-seed-and-migration-design.md
#
# 思路（声明式状态收敛）：
#   1) 起一个临时参考 MySQL，灌入权威 schema + 增量 = "期望态"
#   2) 缺表  → mysqldump --no-data 取干净 DDL，整体改写为 CREATE TABLE IF NOT EXISTS
#   3) 缺列  → 从 information_schema 重建列定义，逐列生成"守卫式 ADD COLUMN"
#   4) 追加  known-fixups.sql（站点特有改名/改类型，纯增量表达不了的）
# 产物 forward-migrate-<ver>.sql：现场只需 mysql 客户端执行，无需 Atlas，对任意起点都收敛。
#
# 每次发版跑一次（我方有 docker 的机器）：  ./gen-forward-migrate.sh
# 备注：列定义重建覆盖 type/null/default/extra 常见情形；关键列另在 forward-migrate.sample.sql
#       里以精确定义二次保障。生产级可改用 Atlas 声明式 `schema apply --dry-run` 交叉校验。
# =============================================================================
set -euo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
PROJ="$(cd "$HERE/../.." && pwd)"                 # coze-studio 根
SCHEMA_SQL="${SCHEMA_SQL:-$PROJ/docker/volumes/mysql/schema.sql}"
# schema.sql 之后的增量（按序），把"期望态"补到最新
INCREMENTS=(
  "$PROJ/docs/ynet-database-sql/04-ynet-sync.sql"
  "$PROJ/docs/ynet-database-sql/05-ynet-release.sql"
  "$PROJ/docs/ynet-database-sql/99-skill-version.sql"
  "$PROJ/docs/ynet-database-sql/100-skill-publish-marketplace.sql"
  "$PROJ/docs/ynet-database-sql/101-super-agent-user-memory.sql"
  "$PROJ/docs/ynet-database-sql/102-super-agent-tool-config.sql"
  "$PROJ/docs/ynet-database-sql/103-skill-review-status.sql"
  "$PROJ/docs/ynet-database-sql/104-ai-product-foundation.sql"
  "$PROJ/docs/ynet-database-sql/105-agent-app-shadow-columns.sql"
  "$PROJ/docker/migrations/20260617_add_agent_type.sql"
)
REF_IMG="${REF_IMG:-mysql:8.4.5}"
REF_DB="refschema"
VER="${VER:-$(git -C "$PROJ" rev-parse --short HEAD 2>/dev/null || date +%Y%m%d)}"
OUT="${OUT:-$HERE/forward-migrate-$VER.sql}"
# 动态/运行表不参与收敛（绝不在目标侧 DROP，它们只是不被本文件管理）
EXCLUDE_RE='^(table_[0-9]|ragflow_|atlas_schema_revisions$)'

command -v docker >/dev/null || { echo "需要 docker 起临时参考库" >&2; exit 1; }
[ -f "$SCHEMA_SQL" ] || { echo "找不到 $SCHEMA_SQL" >&2; exit 1; }

echo "[gen] 起临时参考库 $REF_IMG …"
CID=$(docker run -d --rm -e MYSQL_ALLOW_EMPTY_PASSWORD=1 -e MYSQL_DATABASE="$REF_DB" "$REF_IMG")
trap 'docker stop "$CID" >/dev/null 2>&1 || true' EXIT
for _ in $(seq 1 60); do docker exec "$CID" mysqladmin ping -uroot --silent >/dev/null 2>&1 && break; sleep 2; done
ref(){ docker exec -i "$CID" mysql -uroot --default-character-set=utf8mb4 "$@"; }

echo "[gen] 应用 schema.sql"; ref "$REF_DB" < "$SCHEMA_SQL"
for f in "${INCREMENTS[@]}"; do [ -f "$f" ] && { echo "[gen] +$(basename "$f")"; ref "$REF_DB" < "$f" 2>/dev/null || true; }; done

# ---- 头 ----
{
  echo "-- forward-migrate-$VER.sql  （自动生成，勿手改）"
  echo "-- 声明式收敛·只增不删：缺库→CREATE DATABASE、缺表→CREATE TABLE IF NOT EXISTS、缺列→守卫 ADD COLUMN"
  echo "-- 用法：mysql -h<host> -P<port> -u<user> -p <目标库> < $(basename "$OUT")"
  echo "SET NAMES utf8mb4;"
  echo "SET @OLD_FK := @@FOREIGN_KEY_CHECKS; SET FOREIGN_KEY_CHECKS=0;"
  echo
  echo "-- ===== 缺表 ====="
} > "$OUT"

# ---- 缺表：mysqldump --no-data（干净、每表一条），整体 CREATE TABLE -> CREATE TABLE IF NOT EXISTS ----
mapfile -t TABLES < <(ref -N -e "SELECT table_name FROM information_schema.tables WHERE table_schema='$REF_DB';" | grep -Ev "$EXCLUDE_RE")
docker exec "$CID" mysqldump -uroot --no-data --skip-comments --skip-add-drop-table \
    --default-character-set=utf8mb4 "$REF_DB" "${TABLES[@]}" \
  | sed -E 's/^CREATE TABLE /CREATE TABLE IF NOT EXISTS /' \
  | grep -vE '^/\*|^--|^SET ' >> "$OUT"

# ---- 缺列：守卫式 ADD COLUMN ----
{
  echo
  echo "-- ===== 缺列（守卫式，MySQL8/OceanBase 通用，幂等）====="
  echo "DROP PROCEDURE IF EXISTS __add_col;"
  echo "DELIMITER //"
  echo "CREATE PROCEDURE __add_col(IN t VARCHAR(128), IN c VARCHAR(128), IN def TEXT)"
  echo "BEGIN"
  echo "  IF NOT EXISTS (SELECT 1 FROM information_schema.columns"
  echo "                 WHERE table_schema=DATABASE() AND table_name=t AND column_name=c) THEN"
  echo "    SET @ddl=CONCAT('ALTER TABLE \`',t,'\` ADD COLUMN \`',c,'\` ',def);"
  echo "    PREPARE s FROM @ddl; EXECUTE s; DEALLOCATE PREPARE s;"
  echo "  END IF;"
  echo "END //"
  echo "DELIMITER ;"
} >> "$OUT"

for t in "${TABLES[@]}"; do
  ref -N "$REF_DB" -e "
    SELECT CONCAT(
      'CALL __add_col(''', '$t', ''',''', column_name, ''',''',
      REPLACE(CONCAT(
        column_type,
        IF(is_nullable='NO',' NOT NULL',''),
        IF(column_default IS NOT NULL,
           CONCAT(' DEFAULT ', IF(column_default='CURRENT_TIMESTAMP' OR column_default REGEXP '^-?[0-9]', column_default, CONCAT('\\'', column_default, '\\''))),
           ''),
        IF(extra<>'' AND extra NOT LIKE 'DEFAULT_GENERATED%', CONCAT(' ', extra), '')
      ), '''', ''''''),
      ''');')
    FROM information_schema.columns
    WHERE table_schema='$REF_DB' AND table_name='$t'
    ORDER BY ordinal_position;" >> "$OUT"
done
echo "DROP PROCEDURE __add_col;" >> "$OUT"

# ---- known-fixups（站点特有改名/改类型）----
{ echo; echo "-- ===== known-fixups（站点特有，手维护，全部守卫幂等）====="; } >> "$OUT"
cat "$HERE/known-fixups.sql" >> "$OUT" 2>/dev/null || true
echo "SET FOREIGN_KEY_CHECKS=@OLD_FK;" >> "$OUT"

echo "[gen] 完成 → $OUT"
echo "[gen] 期望表数=${#TABLES[@]}"

# ---- 可选只读自检：与目标库比对，报告"会新增什么"（不改目标）----
# 用法：CHECK_SSH='dev@10.10.10.224' CHECK_SSH_PASS=*** CHECK_CTR=coze-mysql CHECK_DB=opencoze CHECK_PW=*** ./gen-forward-migrate.sh
if [ -n "${CHECK_SSH:-}" ]; then
  echo "[gen] 只读自检：对 $CHECK_SSH/$CHECK_DB 比对差集（不修改目标）"
  : "${CHECK_SSH_PASS:?CHECK 模式需设置 CHECK_SSH_PASS}" "${CHECK_PW:?CHECK 模式需设置 CHECK_PW}"
  tq(){ sshpass -p "$CHECK_SSH_PASS" ssh -o StrictHostKeyChecking=no "$CHECK_SSH" \
        "docker exec -i -e MYSQL_PWD='$CHECK_PW' ${CHECK_CTR:-coze-mysql} mysql -uroot -N -e \"$1\" ${CHECK_DB:-opencoze}"; }
  miss_t=0
  for t in "${TABLES[@]}"; do
    [ "$(tq "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='${CHECK_DB:-opencoze}' AND table_name='$t';" | tr -d '[:space:]')" = 0 ] \
      && { echo "  缺表: $t"; miss_t=$((miss_t+1)); }
  done
  echo "[gen] 自检：目标缺 $miss_t 张表（其余存在；列差集由守卫 ADD COLUMN 现场兜底）"
fi
