# -*- coding: utf-8 -*-
"""读数据生成 studio 手册 HTML(源+base64) + docx。"""
import os, re, base64, json, html as _html
HERE = os.path.dirname(os.path.abspath(__file__))
SD = os.path.join(HERE, 'screenshots')
CSS = open('/tmp/studio_css.txt', encoding='utf-8').read()
D = json.load(open('/tmp/studio_data.json', encoding='utf-8'))
def esc(s): return _html.escape(str(s), quote=False)

def shot(img, cap):
    return f'<figure class="shot"><img src="screenshots/{img}" alt="{esc(cap)}"><figcaption>📷 {esc(cap)}</figcaption></figure>'
def code(text):
    return f'<pre class="code"><button class="copy-btn">复制</button>{esc(text)}</pre>'
def ep(method, path):
    return f'<div class="endpoint-card"><div class="method-row"><span class="method">{method}</span><span class="path">{esc(path)}</span></div></div>'

# ---------- ASIDE ----------
api_groups = {}
for a in D['APIS']:
    api_groups.setdefault(a[3], []).append(a)
api_nav = ''
for g, arr in api_groups.items():
    api_nav += f'<h4>{g}</h4>'
    for a in arr:
        aid = 'api-' + re.sub(r'[^a-z0-9]+','-', a[1].lower()).strip('-')
        api_nav += f'<a href="#{aid}">{esc(a[2])}<span class="endpoint-tag">{esc(a[1])}</span></a>'

step_nav = ''
for i,(tag,title,lead,items) in enumerate(D['STEPS'],1):
    step_nav += f'<a href="#step{i}"><span class="step-num">{i}</span>{esc(title)}</a>'

ASIDE = f'''<aside class="nav">
  <div class="brand"><div class="logo">猎鹰<em>Studio</em></div><div class="sub">对外开放 API · 集成手册</div></div>
  <div class="tab-bar">
    <button class="tab-btn active" data-tab="guide">集成教程</button>
    <button class="tab-btn" data-tab="api">API 接口</button>
    <button class="tab-btn" data-tab="ops">功能速览</button>
  </div>
  <nav class="toc">
    <div class="group active" data-group="guide">
      <h4>开始</h4>
      <a href="#concepts" class="current"><span class="step-num">0</span>能做什么</a>
      <h4>跟着做（5 步）</h4>{step_nav}
    </div>
    <div class="group" data-group="api">
      <h4>通用</h4><a href="#api-base">BASE_URL & 认证<span class="endpoint-tag">Bearer PAT</span></a>
      {api_nav}
    </div>
    <div class="group" data-group="ops"><h4>管理后台</h4><a href="#ops"><span class="step-num">·</span>功能速览</a></div>
  </nav>
</aside>'''

# ---------- GUIDE ----------
concept_rows = ''
pills = ['agent','scene','intent','skill']
for i,(name,desc) in enumerate(D['CONCEPTS']):
    arrow = '↓' if i < len(D['CONCEPTS'])-1 else '·'
    concept_rows += f'<div class="concept-row"><div><span class="pill {pills[i%4]}">{esc(name)}</span></div><div class="desc">{esc(desc)}</div><div class="arrow">{arrow}</div></div>'

GUIDE = f'''<section class="page active" id="concepts">
  <div class="breadcrumb">集成教程 <span class="sep">›</span> 开始</div>
  <h1 class="page-title">猎鹰 Studio<br>对外开放 API 能做什么</h1>
  <p class="lead">在 Studio 里搭好智能体/工作流并发布后，业务系统用一把 PAT 令牌即可通过 API 调用它们——对话、跑工作流、传文件。本手册讲怎么从零接入。</p>
  <div class="concept-diagram">{concept_rows}</div>
  <div class="tip"><strong>💡 集成顺序：</strong>① 发布智能体拿 bot_id → ② 建 PAT 拿令牌 → ③ 调 /v3/chat 对话 / workflow 跑流程。下面 5 步照着做；「API 接口」标签是完整契约，配合同目录 Postman 集合可直接测。</div>
  <div class="pager"><div class="spacer"></div><a href="#step1" class="next"><span class="label">下一步</span><span class="title">第 1 步 · 创建并发布智能体 →</span></a></div>
</section>'''

kinds = {'h':lambda *a: f'<h2>{esc(a[0])}</h2>', 'p':lambda *a: f'<p>{esc(a[0])}</p>',
         's':lambda *a: shot(a[0],a[1]), 'c':lambda *a: code(a[0]),
         't':lambda *a: f'<div class="tip"><strong>{esc(a[0])}</strong>{esc(a[1])}</div>',
         'w':lambda *a: f'<div class="warn"><strong>{esc(a[0])}</strong>{esc(a[1])}</div>'}
for i,(tag,title,lead,items) in enumerate(D['STEPS'],1):
    body = ''.join(kinds[it[0]](*it[1:]) for it in items)
    prev = '#concepts' if i==1 else f'#step{i-1}'
    nxt = '#api-base' if i==len(D['STEPS']) else f'#step{i+1}'
    nxt_label = 'API 接口文档' if i==len(D['STEPS']) else f'第 {i+1} 步'
    GUIDE += f'''<section class="page" id="step{i}">
      <div class="breadcrumb">集成教程 <span class="sep">›</span> 跟着做</div>
      <div class="step-header"><span class="step-tag">{tag}</span></div>
      <h1 class="page-title">{esc(title)}</h1><p class="lead">{esc(lead)}</p>{body}
      <div class="pager"><a href="{prev}" class="prev"><span class="label">上一步</span><span class="title">← 上一步</span></a><a href="{nxt}" class="next"><span class="label">下一步</span><span class="title">{nxt_label} →</span></a></div>
    </section>'''

# ---------- API ----------
API = '''<section class="page" id="api-base"><div class="breadcrumb">API 接口 <span class="sep">›</span> 通用</div>
  <h1 class="page-title">BASE_URL & 认证</h1>
  <p class="lead">所有对外接口用 PAT 以 Bearer 调用。BASE_URL 由部署方提供（如 http://10.10.10.220:9888）。</p>
  <table><thead><tr><th>项</th><th>说明</th></tr></thead><tbody>
  <tr><td>认证</td><td><code>Authorization: Bearer &lt;PAT&gt;</code>（个人访问令牌，在 Web「个人设置→API 授权」创建）</td></tr>
  <tr><td>ID 字段</td><td>bot_id/conversation_id 等 i64 在 JSON 里都是<strong>字符串</strong></td></tr>
  <tr><td>SSE 接口</td><td>/v3/chat（默认）、workflow/stream_run、stream_resume、workflows/chat 返回 SSE，用 curl -N</td></tr>
  </tbody></table></section>'''
for a in D['APIS']:
    method,path,name,group,desc,req,resp,curl = a
    aid = 'api-' + re.sub(r'[^a-z0-9]+','-', path.lower()).strip('-')
    rows = ''.join(f'<tr><td>{esc(r[0])}</td><td>{esc(r[1])}</td><td>{esc(r[2])}</td><td>{esc(r[3])}</td></tr>' for r in req)
    API += f'''<section class="page" id="{aid}"><div class="breadcrumb">API 接口 <span class="sep">›</span> {group}</div>
      <h1 class="page-title">{esc(name)}</h1>{ep(method,path)}<p>{esc(desc)}</p>
      <h2>请求参数</h2><table><thead><tr><th>字段</th><th>类型</th><th>必填</th><th>说明</th></tr></thead><tbody>{rows}</tbody></table>
      <h2>响应示例</h2>{code(resp)}<h2>curl</h2>{code(curl)}</section>'''

# ---------- OPS ----------
OPS = '<section class="page" id="ops"><div class="breadcrumb">功能速览</div><h1 class="page-title">管理后台功能速览</h1>'
OPS += ''.join(shot(img,cap) for img,cap in D['OPS']) + '</section>'

JS = '''<script>
document.querySelectorAll('.tab-btn').forEach(btn=>{btn.addEventListener('click',()=>{const tab=btn.dataset.tab;document.querySelectorAll('.tab-btn').forEach(b=>b.classList.toggle('active',b===btn));document.querySelectorAll('nav.toc .group').forEach(g=>g.classList.toggle('active',g.dataset.group===tab));const f=document.querySelector('nav.toc .group.active a');if(f)f.click();});});
function showSection(id){document.querySelectorAll('section.page').forEach(s=>s.classList.toggle('active',s.id===id));document.querySelectorAll('nav.toc a').forEach(a=>a.classList.toggle('current',a.getAttribute('href')==='#'+id));const al=document.querySelector('nav.toc a[href="#'+id+'"]');if(al){const g=al.closest('.group');if(g){const t=g.dataset.group;document.querySelectorAll('.tab-btn').forEach(b=>b.classList.toggle('active',b.dataset.tab===t));document.querySelectorAll('nav.toc .group').forEach(x=>x.classList.toggle('active',x===g));}}const m=document.querySelector('main.content');if(m)m.scrollTop=0;window.scrollTo(0,0);history.replaceState(null,'','#'+id);}
document.addEventListener('click',e=>{const a=e.target.closest('a[href^="#"]');if(!a)return;const h=a.getAttribute('href');if(h.length<2)return;e.preventDefault();showSection(h.slice(1));});
const ini=location.hash?location.hash.slice(1):'concepts';if(document.getElementById(ini))showSection(ini);
window.addEventListener('hashchange',()=>{const id=location.hash.slice(1);if(id&&document.getElementById(id))showSection(id);});
document.querySelectorAll('pre.code').forEach(pre=>{const b=pre.querySelector('.copy-btn');if(!b)return;b.addEventListener('click',()=>{const t=pre.innerText.replace(/^复制\\n?/,'');navigator.clipboard.writeText(t).then(()=>{const o=b.textContent;b.textContent='已复制';setTimeout(()=>b.textContent=o,1200);});});});
</script>'''

htmldoc = ('<!doctype html>\n<html lang="zh-CN">\n<head>\n<meta charset="utf-8" />\n<title>猎鹰 Studio · 对外开放 API 集成手册</title>\n<meta name="viewport" content="width=1280" />\n'
           + CSS + '\n</head>\n<body>\n' + ASIDE + '\n<main class="content">\n' + GUIDE + API + OPS + '\n</main>\n' + JS + '\n</body>\n</html>')
out = os.path.join(HERE, 'Studio-对外API集成手册.html')
open(out,'w',encoding='utf-8').write(htmldoc)
print('HTML:', out, '| %.0fKB' % (len(htmldoc.encode())/1024), '| sections', htmldoc.count('<section'), '| imgs', htmldoc.count('<img'))

# base64 单文件
def embed(m):
    p = os.path.join(HERE, m.group(1))
    return 'src="data:image/png;base64,' + base64.b64encode(open(p,'rb').read()).decode() + '"' if os.path.exists(p) else m.group(0)
single = re.sub(r'src="(screenshots/[^"]+)"', embed, htmldoc)
out2 = os.path.join(HERE, '猎鹰Studio-对外API集成手册-单文件.html')
open(out2,'w',encoding='utf-8').write(single)
print('单文件:', out2, '| %.1fMB' % (len(single.encode())/1024/1024))
