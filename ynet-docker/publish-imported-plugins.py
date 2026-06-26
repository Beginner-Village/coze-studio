#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# publish-imported-plugins.py
# 把 226 某空间里"有 openapi_doc 但无 tool"的插件批量"发布":
#   解析每个 plugin.openapi_doc 的 paths → 生成 tool_draft / tool / tool_version / plugin_version 行,
#   版本统一 v1.0.0,与后端 publish 的不变式一致(tool.id==tool_draft.id, tool_version.tool_id==tool.id,
#   四处 version 一致)。bytes 走 base64 往返,绝不丢中文。id 从 7.7e18 起,确保高于现有 MAX 不撞。
# 设计依据见 docs/superpowers/specs/2026-06-26-...md(及插件发布链路调研)。
#
# 用法:
#   python3 publish-imported-plugins.py            # dry-run:只生成 SQL + 统计,不写库
#   python3 publish-imported-plugins.py --limit 3  # 只处理前3个插件(测试)
#   python3 publish-imported-plugins.py --apply     # 实际写入 226(会先要求你确认)
#   python3 publish-imported-plugins.py --limit 3 --apply
# 生产224只读无关(本脚本只动226);写前请确保已备份 openynet。
import base64, json, subprocess, sys, time

SP        = "7652614054615187456"          # 目标空间(402087139 的 Personal Space)
VERSION   = "v1.0.0"
ID_BASE   = 7_700_000_000_000_000_000       # 新 id 起点(高于各表现有 MAX≈7.6556e18)
DST_SSH   = "dev@10.10.10.226"
SSH_PASS  = "root1234"
CTR       = "coze-mysql"
DB        = "openynet"
HTTP_METHODS = {"get","post","put","delete","patch","head","options","trace"}

def mysql(sql_text, raw=False):
    """把 SQL 经 stdin 灌给 226 的 mysql(utf8mb4),避免引号地狱。raw=True 不加 -N。"""
    flag = "" if raw else "-N"
    remote = (f"docker exec -i -e MYSQL_PWD={'root'} {CTR} "
              f"mysql -uroot --default-character-set=utf8mb4 {flag} {DB}")
    cmd = ["sshpass","-p",SSH_PASS,"ssh","-o","StrictHostKeyChecking=no","-o","ConnectTimeout=15",DST_SSH,remote]
    r = subprocess.run(cmd, input=sql_text.encode("utf-8"), capture_output=True)
    if r.returncode != 0:
        sys.stderr.write(r.stderr.decode("utf-8","replace"))
    return r.stdout.decode("utf-8","replace"), r.stderr.decode("utf-8","replace")

def b64(s: str) -> str:
    return base64.b64encode(s.encode("utf-8")).decode("ascii")

def fetch_plugins(limit):
    lim = f"LIMIT {int(limit)}" if limit else ""
    q = (f"SELECT id, space_id, developer_id, app_id, plugin_type, "
         f"REPLACE(TO_BASE64(COALESCE(icon_uri,'')),'\\n',''), "
         f"REPLACE(TO_BASE64(COALESCE(server_url,'')),'\\n',''), "
         f"REPLACE(TO_BASE64(COALESCE(version_desc,'')),'\\n',''), "
         f"REPLACE(TO_BASE64(COALESCE(manifest,'')),'\\n',''), "
         f"REPLACE(TO_BASE64(openapi_doc),'\\n','') "
         f"FROM plugin p WHERE p.space_id={SP} "
         f"AND JSON_LENGTH(JSON_EXTRACT(openapi_doc,'$.paths'))>0 "
         f"AND NOT EXISTS(SELECT 1 FROM tool t WHERE t.plugin_id=p.id) "
         f"ORDER BY p.id {lim};")
    out, _ = mysql(q)
    rows = []
    for line in out.splitlines():
        if not line.strip():
            continue
        parts = line.split("\t")
        if len(parts) != 10:
            continue
        rows.append(parts)
    return rows

def gen_sql(rows):
    # 动态算 id 起点:高于各表现有 MAX(含上次运行已写入的),避免跨次运行撞主键
    out, _ = mysql("SELECT GREATEST(IFNULL((SELECT MAX(id) FROM tool),0),"
                   "IFNULL((SELECT MAX(id) FROM tool_draft),0),"
                   "IFNULL((SELECT MAX(id) FROM tool_version),0),"
                   "IFNULL((SELECT MAX(id) FROM plugin_version),0),"
                   f"{ID_BASE})+1000;")
    try: base = int(out.strip())
    except Exception: base = ID_BASE
    print(f"[gen] id 起点 base={base}")
    sql, counter = [], 0
    now = int(time.time()*1000)
    n_plugins = n_tools = 0
    def nid():
        nonlocal counter; counter += 1; return base + counter
    sql.append("SET FOREIGN_KEY_CHECKS=0; SET UNIQUE_CHECKS=0;")
    for (pid, space, dev, app, ptype, icon_b, surl_b, vdesc_b, manifest_b, openapi_b) in rows:
        try:
            doc = json.loads(base64.b64decode(openapi_b))
        except Exception:
            continue
        paths = doc.get("paths") or {}
        tools = []
        for path, methods in paths.items():
            if not isinstance(methods, dict):
                continue
            for method, op in methods.items():
                if method.lower() not in HTTP_METHODS or not isinstance(op, dict):
                    continue
                op_b = b64(json.dumps(op, ensure_ascii=False, separators=(",",":")))
                tools.append((nid(), b64(path), method.upper(), op_b))
        if not tools:
            continue
        n_plugins += 1; n_tools += len(tools)
        pvid = nid()
        sql.append(
            f"INSERT IGNORE INTO plugin_version (id,space_id,developer_id,plugin_id,app_id,icon_uri,server_url,"
            f"plugin_type,version,version_desc,manifest,openapi_doc,created_at) VALUES "
            f"({pvid},{space},{dev},{pid},{app},CONVERT(FROM_BASE64('{icon_b}') USING utf8mb4),"
            f"CONVERT(FROM_BASE64('{surl_b}') USING utf8mb4),{ptype},'{VERSION}',"
            f"CONVERT(FROM_BASE64('{vdesc_b}') USING utf8mb4),CONVERT(FROM_BASE64('{manifest_b}') USING utf8mb4),"
            f"CONVERT(FROM_BASE64('{openapi_b}') USING utf8mb4),{now});")
        for (T, suburl_b, method, op_b) in tools:
            su = f"CONVERT(FROM_BASE64('{suburl_b}') USING utf8mb4)"
            opv = f"CONVERT(FROM_BASE64('{op_b}') USING utf8mb4)"
            sql.append(f"INSERT IGNORE INTO tool_draft (id,plugin_id,created_at,updated_at,sub_url,method,operation,"
                       f"debug_status,activated_status) VALUES ({T},{pid},{now},{now},{su},'{method}',{opv},1,0);")
            sql.append(f"INSERT IGNORE INTO tool (id,plugin_id,created_at,updated_at,version,sub_url,method,operation,"
                       f"activated_status) VALUES ({T},{pid},{now},{now},'{VERSION}',{su},'{method}',{opv},0);")
            tvid = nid()
            sql.append(f"INSERT IGNORE INTO tool_version (id,tool_id,plugin_id,version,sub_url,method,operation,created_at) "
                       f"VALUES ({tvid},{T},{pid},'{VERSION}',{su},'{method}',{opv},{now});")
        sql.append(f"UPDATE plugin SET version='{VERSION}', version_desc='bulk-published' WHERE id={pid};")
    return "\n".join(sql)+"\n", n_plugins, n_tools

def main():
    limit = None; apply = False
    for i,a in enumerate(sys.argv[1:]):
        if a == "--apply": apply = True
        elif a == "--limit": limit = int(sys.argv[sys.argv.index("--limit")+1])
    rows = fetch_plugins(limit)
    print(f"[gen] 取到待发布插件 {len(rows)} 个 (space={SP}{', limit '+str(limit) if limit else ''})")
    sql, np, nt = gen_sql(rows)
    print(f"[gen] 将发布 {np} 个插件,生成 {nt} 个工具(tool/tool_version/tool_draft 各 {nt} 行 + {np} 个 plugin_version)")
    open("/tmp/publish-plugins.sql","w").write(sql)
    print(f"[gen] SQL 已写 /tmp/publish-plugins.sql ({len(sql)} 字节)")
    if not apply:
        print("[gen] dry-run(未写库)。加 --apply 实际写入。"); return
    print("[gen] 正在写入 226 ...");
    out, err = mysql(sql, raw=True)
    if err.strip():
        print("[gen] ⚠️ stderr:\n"+err[:2000])
    else:
        print("[gen] ✓ 写入完成")

if __name__ == "__main__":
    main()
