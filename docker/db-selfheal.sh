#!/bin/sh
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

# db-selfheal.sh —— 镜像启动时自愈数据库 schema + 模型内部数据，再启动应用。
# 适配内网/离线现场：一切自包含在镜像，零外部脚本上传。
#
# 流程：
#   1. 等待 MySQL 就绪（最多 ~60s）
#   2. schema 自愈：补缺表 + 缺列（schema-sync.py，只加不删，幂等）
#   3. 模型内部数据：INSERT IGNORE 灌入 model-seed.sql（幂等）
#   4. exec /app/openynet（把控制权交还应用，保留原 CMD 语义）
#
# 逃生口：DB_SELFHEAL=false 时跳过自愈直接 exec /app/openynet。
set -e

DB_SELFHEAL="${DB_SELFHEAL:-true}"

DB_DIR="/app/db"
SCHEMA_SQL="${DB_DIR}/schema.sql"
SEED_SQL="${DB_DIR}/model-seed.sql"
SYNC_PY="${DB_DIR}/schema-sync.py"

if [ "$DB_SELFHEAL" != "true" ]; then
  echo "[db-selfheal] DB_SELFHEAL=$DB_SELFHEAL，跳过数据库自愈，直接启动应用"
  exec /app/openynet
fi

echo "[db-selfheal] 自愈开始：host=${MYSQL_HOST} port=${MYSQL_PORT} db=${MYSQL_DATABASE} user=${MYSQL_USER}"

# --- 1. 等待 MySQL 就绪（最多 ~60s）---
echo "[db-selfheal] 等待 MySQL 就绪 ..."
ready=false
i=0
while [ "$i" -lt 30 ]; do
  if mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" \
       --default-character-set=utf8mb4 -e "SELECT 1" "$MYSQL_DATABASE" >/dev/null 2>&1; then
    ready=true
    break
  fi
  i=$((i + 1))
  echo "[db-selfheal]   ... MySQL 尚未就绪（第 ${i} 次重试），2s 后重试"
  sleep 2
done

if [ "$ready" != "true" ]; then
  echo "[db-selfheal] warn: MySQL 在 ~60s 内未就绪，跳过自愈直接启动应用（应用自身会再做连接重试）"
  exec /app/openynet
fi
echo "[db-selfheal] MySQL 已就绪"

# --- 2. schema 自愈（补缺表 + 缺列，单步失败不致命）---
echo "[db-selfheal] schema 自愈 ..."
python3 "$SYNC_PY" \
  --ref-sql "$SCHEMA_SQL" \
  --tgt-host "$MYSQL_HOST" \
  --tgt-port "$MYSQL_PORT" \
  --tgt-user "$MYSQL_USER" \
  --tgt-pass "$MYSQL_PASSWORD" \
  --tgt-db "$MYSQL_DATABASE" \
  --apply || echo "[db-selfheal] warn: schema 自愈失败（继续启动，应用可能因缺表报错，请查上面日志）"

# --- 3. 模型内部数据（幂等 INSERT IGNORE，单步失败不致命）---
echo "[db-selfheal] 灌入模型内部数据（model-seed.sql，幂等）..."
mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" \
  --default-character-set=utf8mb4 "$MYSQL_DATABASE" < "$SEED_SQL" \
  && echo "[db-selfheal] 模型内部数据已就绪" \
  || echo "[db-selfheal] warn: 模型内部数据灌入失败（继续启动，请查上面日志）"

# --- 4. 交还控制权给应用 ---
echo "[db-selfheal] 自愈完成，启动应用 /app/openynet"
exec /app/openynet
