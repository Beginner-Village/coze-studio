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
# seed-from-prod.sh — A：把生产 224 的"创作层"真实数据【合并】进 226(coze-super 活库)
# 设计：docs/superpowers/specs/2026-06-26-prod-data-seed-and-migration-design.md
#
# 策略（merge-via-staging，不 DROP → 226 现有 dev-only 数据零丢失）：
#   - 雪花 id 表(workflow/user/space/knowledge/plugin…)：直接 INSERT IGNORE 灌入(carry id，已验证与226不撞)
#   - 顺序自增 id 表(single_agent_draft/version/folder/space_user…)：经 staging，去掉 id 重分配
#     (业务键如 agent_id 是雪花、随行保留；跨表引用走业务键，不引用本地自增 id)
#   - 只删 402087139(226 与 prod 各一份)；排除日志/重表/动态表
#   - 顺带用 forward-migrate 给 226 补齐缺的 4 张表
#
# 铁律：生产侧【只读】；226 改动前【全量备份】可回滚；DRY_RUN=1 只预览。
#   依赖：sshpass。 用法： DRY_RUN=1 SRC_SSH_PASS=.. DST_SSH_PASS=.. ./seed-from-prod.sh
# =============================================================================
set -euo pipefail

# ----------------------------- 配置（env 可覆盖）-----------------------------
SRC_SSH="${SRC_SSH:-dev@10.10.10.224}"; SRC_SSH_PASS="${SRC_SSH_PASS:-}"   # 生产源(只读)
SRC_MYSQL_CTR="${SRC_MYSQL_CTR:-coze-mysql}"; SRC_DB="${SRC_DB:-opencoze}"
SRC_MYSQL_USER="${SRC_MYSQL_USER:-root}"; SRC_MYSQL_PASS="${SRC_MYSQL_PASS:-}"  # 留空=探测容器env
DST_SSH="${DST_SSH:-dev@10.10.10.226}"; DST_SSH_PASS="${DST_SSH_PASS:-}"   # 226 coze-super 活库
DST_MYSQL_CTR="${DST_MYSQL_CTR:-coze-mysql}"; DST_DB="${DST_DB:-openynet}"
DST_MYSQL_USER="${DST_MYSQL_USER:-root}"; DST_MYSQL_PASS="${DST_MYSQL_PASS:-}"  # 留空=探测
STAGING_DB="${STAGING_DB:-openynet_prodstage}"
DELETE_EMAIL="${DELETE_EMAIL:-402087139@qq.com}"
SNOWFLAKE_MIN="${SNOWFLAKE_MIN:-1000000000000000}"   # MAX(id)>=此=雪花(carry)，否则=顺序自增(重分配)
PROJ_ROOT="${PROJ_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
WORKDIR="${WORKDIR:-/tmp/seed-from-prod}"; DRY_RUN="${DRY_RUN:-0}"; TS="$(date +%Y%m%d-%H%M%S)"

# 排除：运行/日志/重表（创作层=其余）。动态 table_<id>/ragflow_* 另在 dump 时排除。
EXCLUDE_TABLES="node_execution workflow_execution run_record workflow_snapshot conversation message \
knowledge_document_slice operation_log audit_logs chatflow_conversation_history \
agent_conversation_mapping statistics_export_file data_copy_task \
model_template workflow_version single_agent_version"
# 注：workflow_version(672MB)/single_agent_version(80MB) 是历史发布版本快照，用户确认不需要 → 不导；工作流实体在 workflow_draft。
# 注：model_template 无雪花业务键(仅 id+provider)且是环境相关的全局模型配置 → 不导(保留226自己的)。
# 已验证其余 reassign 表均有业务键(agent_id/skill_id 或按 space_id/user_id 配置)，重分配 id 安全。
# 已知跨环境局限：导入的 prod 智能体/工作流的"模型绑定"指向 prod 模型，226 需按需重选模型。

# 给 226 补缺表用：纯 CREATE TABLE IF NOT EXISTS 的 docs（幂等；不含 ALTER 文件，226 已有那些列）
SCHEMA_TABLE_FILES="04-ynet-sync 05-ynet-release 99-skill-version 100-skill-publish-marketplace \
101-super-agent-user-memory 103-skill-review-status 104-ai-product-foundation"

# ------------------------------- 工具 ----------------------------------------
log(){ printf '\033[1;36m[seed %s]\033[0m %s\n' "$(date +%H:%M:%S)" "$*"; }
die(){ printf '\033[1;31m[seed FATAL]\033[0m %s\n' "$*" >&2; exit 1; }
confirm(){ if [ "$DRY_RUN" = 1 ]; then log "DRY-RUN 跳过确认：$*"; return 0; fi
  read -r -p $'\033[1;33m⚠️  '"$*"$'  输入 yes 继续：\033[0m ' a; [ "$a" = yes ] || die "用户取消"; }
ssh_src(){ sshpass -p "$SRC_SSH_PASS" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=15 "$SRC_SSH" "$@"; }
ssh_dst(){ sshpass -p "$DST_SSH_PASS" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=15 "$DST_SSH" "$@"; }
# SQL 经 stdin 传入(不用 -e)，避免反引号被本地 shell 命令替换
src_q(){ printf '%s\n' "$1" | ssh_src "docker exec -i -e MYSQL_PWD='$SRC_MYSQL_PASS' $SRC_MYSQL_CTR mysql -u$SRC_MYSQL_USER -N --default-character-set=utf8mb4 ${2:-$SRC_DB}"; }
dst_run(){ ssh_dst "docker exec -i -e MYSQL_PWD='$DST_MYSQL_PASS' $DST_MYSQL_CTR mysql -u$DST_MYSQL_USER --default-character-set=utf8mb4 ${1:-$DST_DB}"; }
dst_q(){ printf '%s\n' "$1" | ssh_dst "docker exec -i -e MYSQL_PWD='$DST_MYSQL_PASS' $DST_MYSQL_CTR mysql -u$DST_MYSQL_USER -N --default-character-set=utf8mb4 ${2:-$DST_DB}"; }
in_excl(){ case " $EXCLUDE_TABLES " in *" $1 "*) return 0;; *) return 1;; esac; }

REASSIGN=""   # 顺序自增表(运行时探测)
CARRY=""      # 雪花表

# ------------------------------- 阶段 ----------------------------------------
phase0_preflight(){
  log "阶段0｜预检（merge 模式，目标=$DST_SSH/${DST_DB}）"
  command -v sshpass >/dev/null || die "缺 sshpass"
  [ -n "$SRC_SSH_PASS" ] || die "需 export SRC_SSH_PASS"; [ -n "$DST_SSH_PASS" ] || die "需 export DST_SSH_PASS"
  ssh_src 'echo ok' >/dev/null 2>&1 || die "连不上生产 ${SRC_SSH}"
  ssh_dst 'echo ok' >/dev/null 2>&1 || die "连不上 226 ${DST_SSH}"
  if [ -z "$SRC_MYSQL_PASS" ]; then SRC_MYSQL_PASS="$(ssh_src "docker inspect $SRC_MYSQL_CTR --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^MYSQL_ROOT_PASSWORD=//p'")"; fi
  [ -n "$SRC_MYSQL_PASS" ] || die "需 SRC_MYSQL_PASS"
  if [ -z "$DST_MYSQL_PASS" ]; then DST_MYSQL_PASS="$(ssh_dst "docker inspect $DST_MYSQL_CTR --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -n 's/^MYSQL_ROOT_PASSWORD=//p'")"; fi
  [ -n "$DST_MYSQL_PASS" ] || die "需 DST_MYSQL_PASS"
  log "源用户=$(src_q 'SELECT COUNT(*) FROM user;'|tr -d '[:space:]')  226用户=$(dst_q 'SELECT COUNT(*) FROM user;'|tr -d '[:space:]')"
  mkdir -p "$WORKDIR"
}

phase1_backup(){
  log "阶段1｜226 全量备份（红线，可回滚）"
  local f="$WORKDIR/backup-$DST_DB-$TS.sql.gz"
  if [ "$DRY_RUN" = 1 ]; then log "DRY-RUN: 备份 $DST_DB → $f"; return; fi
  ssh_dst "docker exec -e MYSQL_PWD='$DST_MYSQL_PASS' $DST_MYSQL_CTR mysqldump -u$DST_MYSQL_USER --single-transaction --quick --set-gtid-purged=OFF --default-character-set=utf8mb4 $DST_DB" | gzip > "$f"
  [ -s "$f" ] || die "备份为空，中止"
  log "已备份 → $f （$(du -h "$f"|cut -f1)）"
}

phase2_schema_topup(){
  log "阶段2｜forward-migrate 给 226 补缺表（CREATE IF NOT EXISTS，幂等）"
  for n in $SCHEMA_TABLE_FILES; do
    local f="$PROJ_ROOT/docs/ynet-database-sql/$n.sql"
    [ -f "$f" ] || { log "  ⚠️ 缺 $f"; continue; }
    log "  +$n"; [ "$DRY_RUN" = 1 ] || dst_run < "$f"
  done
}

phase3_del_dst_402(){
  log "阶段3｜删 226 的 ${DELETE_EMAIL}（解 email 唯一冲突）"
  cat <<SQL | { [ "$DRY_RUN" = 1 ] && cat || dst_run; }
SET @u := (SELECT id FROM user WHERE email='$DELETE_EMAIL' LIMIT 1);
DELETE FROM space_user WHERE user_id=@u;
UPDATE space SET deleted_at=UNIX_TIMESTAMP()*1000 WHERE owner_id=@u AND deleted_at=0;
DELETE FROM user WHERE id=@u;
SQL
}

classify_tables(){
  # 创作层表 = 226现有表 ∩ prod现有表，去 EXCLUDE 与 动态/ragflow；3 次批量查询，不做 per-table SSH
  log "分类创作层表（雪花 carry / 顺序自增 reassign）…"
  local dsttabs prodtabs reassign_raw t
  dsttabs="$(dst_q "SELECT table_name FROM information_schema.tables WHERE table_schema='$DST_DB';")"
  prodtabs="$(src_q "SELECT table_name FROM information_schema.tables WHERE table_schema='$SRC_DB' AND table_name NOT LIKE 'table\\_%' AND table_name NOT LIKE 'ragflow\\_%';")"
  # 顺序自增表：id 列 auto_increment 且 information_schema 的 AUTO_INCREMENT(下个值) < 雪花阈值
  reassign_raw="$(src_q "SELECT c.table_name FROM information_schema.columns c JOIN information_schema.tables t ON t.table_schema=c.table_schema AND t.table_name=c.table_name WHERE c.table_schema='$SRC_DB' AND c.column_name='id' AND c.extra LIKE '%auto_increment%' AND t.AUTO_INCREMENT IS NOT NULL AND t.AUTO_INCREMENT < $SNOWFLAKE_MIN;")"
  local cand="" unionq="" nonempty=""
  for t in $prodtabs; do
    in_excl "$t" && continue
    echo "$dsttabs" | grep -qxF "$t" || continue            # 必须 226 也有此表
    if echo "$reassign_raw" | grep -qxF "$t"; then cand="$cand $t"; else CARRY="$CARRY $t"; fi
  done
  # 候选自增表里，只有"226 实际有行"的才真 reassign；226 空表 → carry(prod 直灌 exact，零冲突零丢失)
  for t in $cand; do
    [ -z "$unionq" ] && unionq="SELECT '$t' n,COUNT(*) c FROM \`$t\`" || unionq="$unionq UNION ALL SELECT '$t',COUNT(*) FROM \`$t\`"
  done
  [ -n "$unionq" ] && nonempty="$(dst_q "$unionq" | awk '$2>0{print $1}')"
  for t in $cand; do
    if echo "$nonempty" | grep -qxF "$t"; then REASSIGN="$REASSIGN $t"; else CARRY="$CARRY $t"; fi
  done
  log "分类：carry(直灌exact)=$(echo $CARRY|wc -w)张  reassign(226有行的自增表)=$(echo $REASSIGN|wc -w)张"
  log "reassign 表：${REASSIGN:-(无)}"
}

phase4_import_carry(){
  log "阶段4｜雪花表直接灌入(INSERT IGNORE, carry id)"
  [ -n "$CARRY" ] || { log "  无 carry 表"; return; }
  # 只导 CARRY 表：用显式表清单
  if [ "$DRY_RUN" = 1 ]; then log "DRY-RUN: mysqldump $(echo $CARRY|wc -w) 张雪花表 --insert-ignore → 直灌 $DST_DB"; return; fi
  ssh_src "docker exec -e MYSQL_PWD='$SRC_MYSQL_PASS' $SRC_MYSQL_CTR mysqldump -u$SRC_MYSQL_USER \
      --no-create-info --complete-insert --insert-ignore --single-transaction --quick --skip-triggers \
      --set-gtid-purged=OFF --default-character-set=utf8mb4 --hex-blob $SRC_DB $CARRY" \
    | { echo 'SET FOREIGN_KEY_CHECKS=0; SET UNIQUE_CHECKS=0;'; cat; } | dst_run
  log "  雪花表灌入完成"
}

phase5_import_reassign(){
  log "阶段5｜自增表经 staging 去 id 重分配"
  [ -n "$REASSIGN" ] || { log "  无 reassign 表"; return; }
  if [ "$DRY_RUN" = 1 ]; then log "DRY-RUN: staging=${STAGING_DB}；对每张 reassign 表 INSERT(去id) SELECT"; return; fi
  dst_run "information_schema" <<SQL
DROP DATABASE IF EXISTS \`$STAGING_DB\`; CREATE DATABASE \`$STAGING_DB\` DEFAULT CHARACTER SET utf8mb4;
SQL
  local t cols
  for t in $REASSIGN; do
    log "  reassign: $t"
    dst_run "$STAGING_DB" <<SQL
CREATE TABLE \`$t\` LIKE \`$DST_DB\`.\`$t\`;
SQL
    ssh_src "docker exec -e MYSQL_PWD='$SRC_MYSQL_PASS' $SRC_MYSQL_CTR mysqldump -u$SRC_MYSQL_USER \
        --no-create-info --complete-insert --single-transaction --quick --skip-triggers \
        --set-gtid-purged=OFF --default-character-set=utf8mb4 --hex-blob $SRC_DB $t" | dst_run "$STAGING_DB"
    cols="$(dst_q "SET SESSION group_concat_max_len=1000000; SELECT GROUP_CONCAT(CONCAT('\`',column_name,'\`') ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema='$DST_DB' AND table_name='$t' AND column_name<>'id';" "$DST_DB")"
    dst_run "$DST_DB" <<SQL
SET FOREIGN_KEY_CHECKS=0;
INSERT IGNORE INTO \`$DST_DB\`.\`$t\` ($cols) SELECT $cols FROM \`$STAGING_DB\`.\`$t\`;
SQL
  done
  dst_run "information_schema" <<SQL
DROP DATABASE IF EXISTS \`$STAGING_DB\`;
SQL
  log "  reassign 完成，staging 已清理"
}

phase6_del_prod_402(){
  log "阶段6｜删合并后 prod 侧 $DELETE_EMAIL 痕迹（其 space 软删，账号删）"
  cat <<SQL | { [ "$DRY_RUN" = 1 ] && cat || dst_run; }
SET @u := (SELECT id FROM user WHERE email='$DELETE_EMAIL' LIMIT 1);
UPDATE space SET deleted_at=UNIX_TIMESTAMP()*1000 WHERE owner_id=@u AND deleted_at=0;
DELETE FROM space_user WHERE user_id=@u;
DELETE FROM user WHERE id=@u;
SQL
}

phase7_minio(){ log "阶段7｜MinIO 镜像 TODO[执行期]：mc mirror 224桶→226桶（头像/知识库文档/上传件）"; }
phase8_es(){ log "阶段8｜ES 重建 TODO[执行期]：调后端重建索引 对创作层资源重新 index"; }

phase9_verify(){
  log "阶段9｜校验（合并后 226 应 ≈ 226原有 + prod）"
  for t in user space single_agent_draft workflow_meta plugin knowledge; do
    printf '  %-20s src=%s dst=%s\n' "$t" "$(src_q "SELECT COUNT(*) FROM $t;"|tr -d '[:space:]')" "$(dst_q "SELECT COUNT(*) FROM $t;"|tr -d '[:space:]')"
  done
  log "dev-only 6 账号应仍在："
  dst_q "SELECT email FROM user WHERE email LIKE 'codex-%' OR email='sa-e2e-test@ynet.local';"
  log "请人工抽查：登录 226，dev-only 账号空间 + 新导入的 prod 工作流均可见可打开"
}

main(){
  log "=== seed-from-prod (MERGE)  src=$SRC_SSH/$SRC_DB → dst=$DST_SSH/$DST_DB  DRY_RUN=$DRY_RUN ==="
  phase0_preflight; phase1_backup; phase2_schema_topup; phase3_del_dst_402
  classify_tables; phase4_import_carry; phase5_import_reassign; phase6_del_prod_402
  phase7_minio; phase8_es; phase9_verify
  log "=== 完成（MinIO/ES 为执行期 TODO）==="
}
main "$@"
