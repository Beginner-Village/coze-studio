# -*- coding: utf-8 -*-
"""生成 studio 全量手册 HTML(源+base64) + docx：集成教程 + 全量API(21模块179接口) + 功能速览。"""
import os, re, base64, json, html as _html
HERE = os.path.dirname(os.path.abspath(__file__))
SD = os.path.join(HERE, 'screenshots')
CSS = open('/tmp/studio_css.txt', encoding='utf-8').read()
D = json.load(open('/tmp/studio_data.json', encoding='utf-8'))      # CONCEPTS, STEPS
MODULES = json.load(open('/tmp/studio_modules.json', encoding='utf-8'))  # 21 模块
def esc(s): return _html.escape(str(s), quote=False)
def slug(s): return 'm-' + re.sub(r'[^a-z0-9]+','-', s.lower().replace('▸','').replace(' ','')).strip('-') or 'm'
def shot(img, cap): return f'<figure class="shot"><img src="screenshots/{img}" alt="{esc(cap)}"><figcaption>📷 {esc(cap)}</figcaption></figure>'
def code(t): return f'<pre class="code"><button class="copy-btn">复制</button>{esc(t)}</pre>'

# ---- ASIDE ----
ext_mods = [m for m in MODULES if m[1]=='bearer']
mgr_mods = [m for m in MODULES if m[1] in ('cookie','public')]
def navlinks(mods):
    s=''
    for fname,auth,items in mods:
        s += f'<a href="#{slug(fname)}">{esc(fname)}<span class="endpoint-tag">{len(items)}</span></a>'
    return s
step_nav = ''.join(f'<a href="#step{i}"><span class="step-num">{i}</span>{esc(t)}</a>' for i,(tag,t,l,it) in enumerate(D['STEPS'],1))

ASIDE = f'''<aside class="nav">
  <div class="brand"><div class="logo">猎鹰<em>Studio</em></div><div class="sub">全量 API · 集成手册</div></div>
  <div class="tab-bar">
    <button class="tab-btn active" data-tab="guide">集成教程</button>
    <button class="tab-btn" data-tab="api">API 接口</button>
    <button class="tab-btn" data-tab="ops">功能速览</button>
  </div>
  <nav class="toc">
    <div class="group active" data-group="guide">
      <h4>开始</h4><a href="#concepts" class="current"><span class="step-num">0</span>能做什么</a>
      <h4>对外集成（5 步）</h4>{step_nav}
    </div>
    <div class="group" data-group="api">
      <h4>通用</h4><a href="#api-base">认证与约定<span class="endpoint-tag">2套</span></a>
      <h4>对外 API（Bearer PAT）</h4>{navlinks(ext_mods)}
      <h4>平台管理 API（Cookie）</h4>{navlinks(mgr_mods)}
    </div>
    <div class="group" data-group="ops"><h4>管理后台</h4><a href="#ops"><span class="step-num">·</span>功能速览</a></div>
  </nav>
</aside>'''

# ---- GUIDE（概念 + 5步，同前）----
pills=['agent','scene','intent','skill']
crows=''.join(f'<div class="concept-row"><div><span class="pill {pills[i%4]}">{esc(n)}</span></div><div class="desc">{esc(d)}</div><div class="arrow">{"↓" if i<len(D["CONCEPTS"])-1 else "·"}</div></div>' for i,(n,d) in enumerate(D['CONCEPTS']))
GUIDE=f'''<section class="page active" id="concepts"><div class="breadcrumb">集成教程 <span class="sep">›</span> 开始</div>
  <h1 class="page-title">猎鹰 Studio<br>对外开放 API 能做什么</h1>
  <p class="lead">在 Studio 里搭好智能体/工作流并发布后，业务系统用 PAT 令牌通过 API 调用——对话、跑工作流、传文件。本手册「集成教程」讲对外集成 5 步；「API 接口」覆盖全平台 21 个模块 179 个接口（对外 + 平台管理）。</p>
  <div class="concept-diagram">{crows}</div>
  <div class="tip"><strong>💡 两套认证：</strong>对外 API 用 <code>Authorization: Bearer &lt;PAT&gt;</code>；平台管理 API（知识库/插件/数据库等）走前端登录态 <code>session_key</code> cookie。详见「API 接口 › 认证与约定」。</div>
  <div class="pager"><div class="spacer"></div><a href="#step1" class="next"><span class="label">下一步</span><span class="title">第 1 步 →</span></a></div></section>'''
kinds={'h':lambda *a:f'<h2>{esc(a[0])}</h2>','p':lambda *a:f'<p>{esc(a[0])}</p>','s':lambda *a:shot(a[0],a[1]),'c':lambda *a:code(a[0]),'t':lambda *a:f'<div class="tip"><strong>{esc(a[0])}</strong>{esc(a[1])}</div>','w':lambda *a:f'<div class="warn"><strong>{esc(a[0])}</strong>{esc(a[1])}</div>'}
for i,(tag,title,lead,items) in enumerate(D['STEPS'],1):
    body=''.join(kinds[it[0]](*it[1:]) for it in items)
    prev='#concepts' if i==1 else f'#step{i-1}'; nxt='#api-base' if i==len(D['STEPS']) else f'#step{i+1}'
    GUIDE+=f'''<section class="page" id="step{i}"><div class="breadcrumb">集成教程 <span class="sep">›</span> 跟着做</div>
      <div class="step-header"><span class="step-tag">{tag}</span></div><h1 class="page-title">{esc(title)}</h1><p class="lead">{esc(lead)}</p>{body}
      <div class="pager"><a href="{prev}" class="prev"><span class="label">上一步</span><span class="title">← 上一步</span></a><a href="{nxt}" class="next"><span class="label">下一步</span><span class="title">下一步 →</span></a></div></section>'''

# ---- API（通用 + 21 模块）----
API=f'''<section class="page" id="api-base"><div class="breadcrumb">API 接口 <span class="sep">›</span> 通用</div>
  <h1 class="page-title">认证与约定</h1>
  <p class="lead">猎鹰 Studio 共 {sum(len(it) for _,_,it in MODULES)} 个接口（本手册收录），分两套认证。</p>
  <table><thead><tr><th>类别</th><th>认证</th><th>说明</th></tr></thead><tbody>
  <tr><td>对外 API（/v1,/v3）</td><td><code>Authorization: Bearer &lt;PAT&gt;</code></td><td>业务系统集成，PAT 在「个人设置→API授权」创建</td></tr>
  <tr><td>平台管理 API（/api/*）</td><td>Cookie <code>session_key</code></td><td>前端管理后台（知识库/插件/数据库等），登录后携带</td></tr>
  </tbody></table>
  <div class="warn"><strong>⚠️ ID 字段：</strong>所有 i64 ID（bot_id/dataset_id 等）在 JSON 里都是<strong>字符串</strong>。SSE 接口（/v3/chat、stream_run、/api/conversation/chat 等）返回 text/event-stream。</div>
  <div class="tip"><strong>💡 配合 Postman：</strong>同目录 <code>Studio-全量API.postman_collection.json</code> 含全部 {sum(len(it) for _,_,it in MODULES)} 个接口（两套 auth 分组），导入即可逐个测试。</div></section>'''
for fname,auth,items in MODULES:
    sid=slug(fname); badge='Bearer PAT' if auth=='bearer' else ('公开' if auth=='public' else 'Cookie session')
    rows=''.join(f'<tr><td>{m}</td><td>{esc(p)}</td><td>{esc(n)}</td></tr>' for m,p,n,bd,d in items)
    examples=''
    shown=0
    for m,p,n,bd,d in items:
        if bd and shown<3:
            examples+=f'<h3>{m} {esc(n)}</h3>'+(f'<p>{esc(d)}</p>' if d else '')+code(json.dumps(bd,ensure_ascii=False,indent=2))
            shown+=1
    API+=f'''<section class="page" id="{sid}"><div class="breadcrumb">API 接口 <span class="sep">›</span> {esc(fname.split("▸")[0].strip())}</div>
      <h1 class="page-title">{esc(fname)}</h1>
      <div class="endpoint-card"><div class="method-row"><span class="method">{badge}</span><span class="path">{len(items)} 个接口</span></div></div>
      <h2>接口清单</h2><table><thead><tr><th>方法</th><th>路径</th><th>用途</th></tr></thead><tbody>{rows}</tbody></table>
      {('<h2>请求体示例</h2>'+examples) if examples else ''}</section>'''

# ---- OPS 截图 ----
OPS='<section class="page" id="ops"><div class="breadcrumb">功能速览</div><h1 class="page-title">管理后台功能速览</h1>'
for img,cap in [('03-project-list.png','项目开发：智能体列表'),('04-workflow-list.png','资源库：工作流'),('05-bot-edit.png','智能体编排：发布+预览调试(URL含bot_id)'),('11-card-list.png','资源库：卡片'),('10-prompt-list.png','资源库：提示词'),('09-database-list.png','资源库：数据库'),('01-pat-list.png','API授权：个人访问令牌')]:
    OPS+=shot(img,cap)
OPS+='</section>'

JS='''<script>
document.querySelectorAll('.tab-btn').forEach(btn=>{btn.addEventListener('click',()=>{const tab=btn.dataset.tab;document.querySelectorAll('.tab-btn').forEach(b=>b.classList.toggle('active',b===btn));document.querySelectorAll('nav.toc .group').forEach(g=>g.classList.toggle('active',g.dataset.group===tab));const f=document.querySelector('nav.toc .group.active a');if(f)f.click();});});
function showSection(id){document.querySelectorAll('section.page').forEach(s=>s.classList.toggle('active',s.id===id));document.querySelectorAll('nav.toc a').forEach(a=>a.classList.toggle('current',a.getAttribute('href')==='#'+id));const al=document.querySelector('nav.toc a[href="#'+id+'"]');if(al){const g=al.closest('.group');if(g){const t=g.dataset.group;document.querySelectorAll('.tab-btn').forEach(b=>b.classList.toggle('active',b.dataset.tab===t));document.querySelectorAll('nav.toc .group').forEach(x=>x.classList.toggle('active',x===g));}}const m=document.querySelector('main.content');if(m)m.scrollTop=0;window.scrollTo(0,0);history.replaceState(null,'','#'+id);}
document.addEventListener('click',e=>{const a=e.target.closest('a[href^="#"]');if(!a)return;const h=a.getAttribute('href');if(h.length<2)return;e.preventDefault();showSection(h.slice(1));});
const ini=location.hash?location.hash.slice(1):'concepts';if(document.getElementById(ini))showSection(ini);
window.addEventListener('hashchange',()=>{const id=location.hash.slice(1);if(id&&document.getElementById(id))showSection(id);});
document.querySelectorAll('pre.code').forEach(pre=>{const b=pre.querySelector('.copy-btn');if(!b)return;b.addEventListener('click',()=>{const t=pre.innerText.replace(/^复制\\n?/,'');navigator.clipboard.writeText(t).then(()=>{const o=b.textContent;b.textContent='已复制';setTimeout(()=>b.textContent=o,1200);});});});
</script>'''

htmldoc=('<!doctype html>\n<html lang="zh-CN">\n<head>\n<meta charset="utf-8" />\n<title>猎鹰 Studio · 全量 API 集成手册</title>\n<meta name="viewport" content="width=1280" />\n'+CSS+'\n</head>\n<body>\n'+ASIDE+'\n<main class="content">\n'+GUIDE+API+OPS+'\n</main>\n'+JS+'\n</body>\n</html>')
out=os.path.join(HERE,'Studio-全量集成手册.html'); open(out,'w',encoding='utf-8').write(htmldoc)
print('HTML:',out,'| %.0fKB'%(len(htmldoc.encode())/1024),'| sections',htmldoc.count('<section'),'| 接口',sum(len(it) for _,_,it in MODULES))
def embed(m):
    p=os.path.join(HERE,m.group(1))
    return 'src="data:image/png;base64,'+base64.b64encode(open(p,'rb').read()).decode()+'"' if os.path.exists(p) else m.group(0)
single=re.sub(r'src="(screenshots/[^"]+)"',embed,htmldoc)
out2=os.path.join(HERE,'猎鹰Studio-全量集成手册-单文件.html'); open(out2,'w',encoding='utf-8').write(single)
print('单文件:',out2,'| %.1fMB'%(len(single.encode())/1024/1024))
