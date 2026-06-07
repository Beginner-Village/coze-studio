#!/usr/bin/env python3
"""
schema-sync — 增量对齐 MySQL schema（只加不删，幂等）。

用途：现场部署新版本镜像后，自动把"目标库(线上)"对齐到"参考schema(新版本)"：
  - 参考库有、目标库没有的【表】     → CREATE TABLE（照搬参考 DDL）
  - 参考库有、目标库没有的【列】     → ALTER TABLE ADD COLUMN
绝不 DROP / 不改已存在的表或列（避免动现场数据）。默认 dry-run，--apply 才执行。

参考来源二选一：
  A) 另一套已是新版本 schema 的 MySQL（--ref-host ...）
  B) 一个结构 SQL 文件（mysqldump --no-data 产物，--ref-sql schema.sql）
     —— 会临时 load 进 target 的一个 scratch 库做参考，结束后删除。

依赖：只用 `mysql` CLI（部署容器都有），无需 pip 包。

示例：
  # 以 224 为参考，对齐 220（先看要做什么）
  python3 schema-sync.py \
    --ref-host 10.10.10.224 --ref-port 3306 --ref-user root --ref-pass *** --ref-db opencoze \
    --tgt-host 10.10.10.220 --tgt-port 3308 --tgt-user root --tgt-pass *** --tgt-db openynet
  # 确认后真正执行
  ... --apply
  # 用新镜像自带的 schema.sql 作参考（现场推荐）
  python3 schema-sync.py --ref-sql /coze/schema.sql \
    --tgt-host 127.0.0.1 --tgt-port 3306 --tgt-user root --tgt-pass *** --tgt-db opencoze --apply
"""
import argparse, subprocess, sys, os, time


def mysql(host, port, user, pw, db, sql, want_rows=True):
    cmd = ["mysql", f"-h{host}", f"-P{port}", f"-u{user}", f"-p{pw}",
           "--default-character-set=utf8mb4", "-N", "-B"]
    if db:
        cmd.append(db)
    r = subprocess.run(cmd, input=sql, capture_output=True, text=True)
    if r.returncode != 0:
        err = r.stderr.replace(pw, "***")
        raise RuntimeError(f"mysql error: {err.strip()}")
    if not want_rows:
        return None
    rows = []
    for line in r.stdout.splitlines():
        if line.strip():
            rows.append(line.split("\t"))
    return rows


def get_tables(conn, db):
    rows = mysql(*conn, None,
        f"SELECT table_name FROM information_schema.tables "
        f"WHERE table_schema='{db}' AND table_type='BASE TABLE'")
    return [r[0] for r in rows]


def get_columns(conn, db, table):
    # 返回 {col: (column_type, is_nullable, default, extra, ordinal)}
    rows = mysql(*conn, None,
        f"SELECT column_name,column_type,is_nullable,column_default,extra,ordinal_position "
        f"FROM information_schema.columns "
        f"WHERE table_schema='{db}' AND table_name='{table}' ORDER BY ordinal_position")
    out = {}
    for r in rows:
        r = (r + [""] * 6)[:6]
        out[r[0]] = (r[1], r[2], r[3], r[4], r[5])
    return out


def get_create(conn, db, table):
    rows = mysql(*conn, db, f"SHOW CREATE TABLE `{table}`")
    # 行格式: table_name \t CREATE TABLE ...
    return rows[0][1].replace("CREATE TABLE `", "CREATE TABLE IF NOT EXISTS `", 1)


def col_ddl(col, meta):
    ctype, nullable, default, extra, _ = meta
    parts = [f"`{col}` {ctype}"]
    parts.append("NULL" if nullable == "YES" else "NOT NULL")
    if default is not None and default != "":
        if default.upper() in ("CURRENT_TIMESTAMP",) or extra:
            parts.append(f"DEFAULT {default}")
        else:
            parts.append(f"DEFAULT '{default}'")
    if extra and "auto_increment" not in extra.lower():
        parts.append(extra)
    return " ".join(parts)


def main():
    ap = argparse.ArgumentParser()
    for side in ("ref", "tgt"):
        ap.add_argument(f"--{side}-host"); ap.add_argument(f"--{side}-port", default="3306")
        ap.add_argument(f"--{side}-user", default="root"); ap.add_argument(f"--{side}-pass", default="")
        ap.add_argument(f"--{side}-db")
    ap.add_argument("--ref-sql", help="结构SQL文件(mysqldump --no-data)作参考，临时load到target")
    ap.add_argument("--apply", action="store_true", help="真正执行(默认dry-run)")
    a = ap.parse_args()

    tgt = (a.tgt_host, a.tgt_port, a.tgt_user, a.tgt_pass)
    scratch = None
    if a.ref_sql:
        scratch = f"_schemaref_{int(time.time())}"
        print(f"[ref] 从 {a.ref_sql} 临时建参考库 {scratch} ...")
        mysql(*tgt, None, f"CREATE DATABASE `{scratch}`", want_rows=False)
        with open(a.ref_sql) as f:
            mysql(*tgt, scratch, f.read(), want_rows=False)
        ref, ref_db = tgt, scratch
    else:
        ref, ref_db = (a.ref_host, a.ref_port, a.ref_user, a.ref_pass), a.ref_db

    try:
        ref_tables = set(get_tables(ref, ref_db))
        tgt_tables = set(get_tables(tgt, a.tgt_db))
        def is_schema_table(t):
            # 跳过: 用户自建动态表(table_数字) + 手工备份表(*_backup_* / *_bak)
            if t.startswith("table_"):
                return False
            low = t.lower()
            if "_backup_" in low or low.endswith("_bak") or low.endswith("_backup"):
                return False
            return True
        missing_tables = [t for t in sorted(ref_tables - tgt_tables) if is_schema_table(t)]

        actions = []
        for t in missing_tables:
            ddl = get_create(ref, ref_db, t)
            actions.append(("CREATE TABLE", t, ddl))
        for t in sorted(ref_tables & tgt_tables):
            rc = get_columns(ref, ref_db, t)
            tc = get_columns(tgt, a.tgt_db, t)
            for col in rc:
                if col not in tc:
                    actions.append(("ADD COLUMN", t,
                                    f"ALTER TABLE `{t}` ADD COLUMN {col_ddl(col, rc[col])}"))

        print(f"\n=== 差异: 缺表 {len([x for x in actions if x[0]=='CREATE TABLE'])} 张, "
              f"缺列 {len([x for x in actions if x[0]=='ADD COLUMN'])} 个 ===")
        for kind, t, sql in actions:
            print(f"[{kind}] {t}: {sql[:120]}")

        if not actions:
            print("✅ schema 已对齐，无需变更")
        elif a.apply:
            print("\n=== 执行中 ===")
            for kind, t, sql in actions:
                mysql(*tgt, a.tgt_db, sql, want_rows=False)
                print(f"  ✓ {kind} {t}")
            print(f"✅ 已应用 {len(actions)} 项变更")
        else:
            print("\n(dry-run，加 --apply 才执行)")
    finally:
        if scratch:
            mysql(*tgt, None, f"DROP DATABASE `{scratch}`", want_rows=False)
            print(f"[ref] 已清理临时参考库 {scratch}")


if __name__ == "__main__":
    main()
