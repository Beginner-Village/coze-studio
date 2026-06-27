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
# load-prod-resources.sh — 把生产 224 的【全部插件 + 抽样工作流】定向灌进 226 某账号的空间
# 目的：丰富 402087139 的测试空间，方便测"工作流自动编排"(orchestration 主要用插件做积木)。
# 设计：docs/superpowers/specs/2026-06-26-prod-data-seed-and-migration-design.md
#
# 做法（staging + 改归属，不 DROP、不动 402087139 本身）：
#   - 插件 6 表(plugin/plugin_draft/plugin_version/tool*)：经 staging，把 space_id/developer_id 改成
#     402087139 的空间/uid，再 INSERT IGNORE 进 226(plugin id 雪花、226 插件表为空 → 零冲突)。
#   - 工作流：抽样 WF_SAMPLE 个(默认 300，最近的)，workflow_meta 改 space_id/creator_id 后灌入，
#     workflow_draft 按同一批 id 灌入(雪花 id，与 226 已验证不撞)。
# 铁律：生产侧【只读】；226 改动前【全量备份(--force 跳坏视图)】可回滚；DRY_RUN=1 预览。
#   用法： DRY_RUN=1 SRC_SSH_PASS=.. DST_SSH_PASS=.. ./load-prod-resources.sh
# =============================================================================
set -euo pipefail

SRC_SSH="${SRC_SSH:-dev@10.10.10.224}"; SRC_SSH_PASS="${SRC_SSH_PASS:-}"
SRC_MYSQL_CTR="${SRC_MYSQL_CTR:-coze-mysql}"; SRC_DB="${SRC_DB:-opencoze}"; SRC_MYSQL_USER="${SRC_MYSQL_USER:-root}"; SRC_MYSQL_PASS="${SRC_MYSQL_PASS:-}"
DST_SSH="${DST_SSH:-dev@10.10.10.226}"; DST_SSH_PASS="${DST_SSH_PASS:-}"
DST_MYSQL_CTR="${DST_MYSQL_CTR:-coze-mysql}"; DST_DB="${DST_DB:-openynet}"; DST_MYSQL_USER="${DST_MYSQL_USER:-root}"; DST_MYSQL_PASS="${DST_MYSQL_PASS:-}"
TARGET_EMAIL="${TARGET_EMAIL:-402087139@qq.com}"   # 灌到这个账号的空间
WF_SAMPLE="${WF_SAMPLE:-300}"                       # 抽样工作流个数
STAGING_DB="${STAGING_DB:-openynet_loadstage}"
WORKDIR="${WORKDIR:-/tmp/load-prod}"; DRY_RUN="${DRY_RUN:-0}"; TS="$(date +%Y%m%d-%H%M%S)"

log(){ printf '\033[1;36m[load %s]\033[0m %s\n' "$(date +%H:%M:%S)" "$*"; }
die(){ printf '\033[1;31m[load FATAL]\033[0m %s\n' "$*" >&2; exit 1; }
ssh_src(){ sshpass -p "$SRC_SSH_PASS" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=15 "$SRC_SSH" "$@"; }
ssh_dst(){ sshpass -p "$DST_SSH_PASS" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=15 "$DST_SSH" "$@"; }
src_q(){ printf '%s\n' "$1" | ssh_src "docker exec -i -e MYSQL_PWD='$SRC_MYSQL_PASS' $SRC_MYSQL_CTR mysql -u$SRC_MYSQL_USER -N --default-character-set=utf8mb4 ${2:-$SRC_DB}"; }
dst_q(){ printf '%s\n' "$1" | ssh_dst "docker exec -i -e MYSQL_PWD='$DST_MYSQL_PASS' $DST_MYSQL_CTR mysql -u$DST_MYSQL_USER -N --default-character-set=utf8mb4 ${2:-$DST_DB}"; }
dst_run(){ ssh_dst "docker exec -i -e MYSQL_PWD='$DST_MYSQL_PASS' $DST_MYSQL_CTR mysql -u$DST_MYSQL_USER --default-character-set=utf8mb4 ${1:-$DST_DB}"; }

SP=""; UID2=""

phase0_preflight(){
  log "预检（目标账号 ${TARGET_EMAIL}）"
  command -v sshpass >/dev/null || die "缺 sshpass"
  [ -n "$SRC_SSH_PASS" ] && [ -n "$DST_SSH_PASS" ] || die "需 export SRC_SSH_PASS / DST_SSH_PASS"
  ssh_src 'echo ok' >/dev/null 2>&1 || die "连不上 ${SRC_SSH}"
  ssh_dst 'echo ok' >/dev/null 2>&1 || die "连不上 ${DST_SSH}"
  [ -z "$SRC_MYSQL_PASS" ] && SRC_MYSQL_PASS="$(ssh_src "docker inspect $SRC_MYSQL_CTR --format '{{range .Config.Env}}{{println .}}{{end}}'|sed -n 's/^MYSQL_ROOT_PASSWORD=//p'")"
  [ -z "$DST_MYSQL_PASS" ] && DST_MYSQL_PASS="$(ssh_dst "docker inspect $DST_MYSQL_CTR --format '{{range .Config.Env}}{{println .}}{{end}}'|sed -n 's/^MYSQL_ROOT_PASSWORD=//p'")"
  [ -n "$SRC_MYSQL_PASS" ] && [ -n "$DST_MYSQL_PASS" ] || die "需 SRC_MYSQL_PASS / DST_MYSQL_PASS"
  UID2="$(dst_q "SELECT id FROM user WHERE email='$TARGET_EMAIL' LIMIT 1;"|tr -d '[:space:]')"
  [ -n "$UID2" ] || die "226 上找不到账号 $TARGET_EMAIL"
  SP="$(dst_q "SELECT id FROM space WHERE owner_id=$UID2 AND (deleted_at IS NULL OR deleted_at=0) ORDER BY id LIMIT 1;"|tr -d '[:space:]')"
  [ -n "$SP" ] || die "找不到 $TARGET_EMAIL 名下的空间"
  log "目标 uid=$UID2  空间 sid=$SP"
  mkdir -p "$WORKDIR"
}

phase1_backup(){
  log "备份 226 ${DST_DB}（红线，可回滚）"
  local f="$WORKDIR/backup-$DST_DB-$TS.sql.gz"
  if [ "$DRY_RUN" = 1 ]; then log "DRY-RUN: 备份 → $f"; return; fi
  # 226 有坏视图(ragflow_*)，mysqldump --force 会跳过它们继续 dump 但仍非零退出 → 容忍退出码，靠表数校验完整性
  ssh_dst "docker exec -e MYSQL_PWD='$DST_MYSQL_PASS' $DST_MYSQL_CTR mysqldump -u$DST_MYSQL_USER --force --single-transaction --quick --set-gtid-purged=OFF --default-character-set=utf8mb4 $DST_DB" 2>/dev/null | gzip > "$f" || true
  local nt; nt=$(gunzip -c "$f"|grep -c 'CREATE TABLE' || true)
  [ "${nt:-0}" -gt 50 ] || die "备份只含 $nt 张表（疑似不完整或坏视图过多），中止"
  log "已备份 → $f （$(du -h "$f"|cut -f1)，$nt 张表）"
}

stage_init(){ dst_run information_schema <<SQL
DROP DATABASE IF EXISTS \`$STAGING_DB\`; CREATE DATABASE \`$STAGING_DB\` DEFAULT CHARACTER SET utf8mb4;
SQL
}
stage_drop(){ dst_run information_schema <<SQL
DROP DATABASE IF EXISTS \`$STAGING_DB\`;
SQL
}

# load_table <table> <where(无引号,可空)> <rewrite-sql(在staging执行,可空)>
load_table(){
  local t="$1" where="${2:-}" rw="${3:-}" wopt="" cnt
  dst_run "$STAGING_DB" <<SQL
CREATE TABLE \`$t\` LIKE \`$DST_DB\`.\`$t\`;
SQL
  [ -n "$where" ] && wopt="--where='$where'"
  ssh_src "docker exec -e MYSQL_PWD='$SRC_MYSQL_PASS' $SRC_MYSQL_CTR mysqldump -u$SRC_MYSQL_USER --no-create-info --complete-insert --single-transaction --quick --skip-triggers --set-gtid-purged=OFF --default-character-set=utf8mb4 --hex-blob $wopt $SRC_DB $t" | dst_run "$STAGING_DB"
  [ -n "$rw" ] && printf '%s\n' "$rw" | dst_run "$STAGING_DB"
  cnt=$(dst_q "SELECT COUNT(*) FROM \`$t\`;" "$STAGING_DB"|tr -d '[:space:]')
  dst_run "$DST_DB" <<SQL
SET FOREIGN_KEY_CHECKS=0;
INSERT IGNORE INTO \`$DST_DB\`.\`$t\` SELECT * FROM \`$STAGING_DB\`.\`$t\`;
SQL
  log "  $t: staged ${cnt:-0} → 灌入 226"
}

phase2_plugins(){
  [ "${SKIP_PLUGINS:-0}" = 1 ] && { log "跳过插件导入(SKIP_PLUGINS=1)"; return; }
  log "导入全部插件 → 空间 ${SP}（改 space_id/developer_id）"
  if [ "$DRY_RUN" = 1 ]; then log "DRY-RUN: plugin/plugin_draft/plugin_version 改归属 + tool/tool_draft/tool_version 随行 → 灌入"; return; fi
  stage_init
  for t in plugin plugin_draft plugin_version; do
    load_table "$t" "" "UPDATE \`$t\` SET space_id=$SP, developer_id=$UID2;"
  done
  for t in tool tool_draft tool_version; do
    load_table "$t" "" ""
  done
  stage_drop
}

phase3_workflows(){
  log "抽样 $WF_SAMPLE 个工作流 → 空间 ${SP}（改 space_id/creator_id）"
  local ids
  ids="$(src_q "SET SESSION group_concat_max_len=100000000; SELECT GROUP_CONCAT(id) FROM (SELECT id FROM workflow_meta WHERE (deleted_at IS NULL OR deleted_at=0) ORDER BY id DESC LIMIT $WF_SAMPLE) x;")"
  [ -n "$ids" ] || die "取不到工作流样本"
  if [ "$DRY_RUN" = 1 ]; then log "DRY-RUN: workflow_meta/draft WHERE id IN(${WF_SAMPLE}个) 改归属 → 灌入"; return; fi
  stage_init
  load_table "workflow_meta" "id IN ($ids)" "UPDATE workflow_meta SET space_id=$SP, creator_id=$UID2;"
  load_table "workflow_draft" "id IN ($ids)" ""
  stage_drop
}

phase4_verify(){
  log "校验：402087139 空间 $SP 的资源量"
  printf '  plugin(space=%s):   %s\n' "$SP" "$(dst_q "SELECT COUNT(*) FROM plugin WHERE space_id=$SP;"|tr -d '[:space:]')"
  printf '  workflow(space=%s): %s\n' "$SP" "$(dst_q "SELECT COUNT(*) FROM workflow_meta WHERE space_id=$SP AND (deleted_at IS NULL OR deleted_at=0);"|tr -d '[:space:]')"
  log "MinIO 插件/工作流图标镜像为执行期 TODO（不影响功能，仅图标）"
  log "请登录 402087139 → Personal Space 抽查：插件库丰富、工作流可打开可编排"
}

main(){
  log "=== load-prod-resources  src=$SRC_SSH/$SRC_DB → 226 $TARGET_EMAIL 空间  DRY_RUN=$DRY_RUN ==="
  phase0_preflight; phase1_backup; phase2_plugins; phase3_workflows; phase4_verify
  log "=== 完成 ==="
}
main "$@"
