# -*- coding: utf-8 -*-
"""生成 猎鹰 Studio 对外开放 API 集成手册：HTML(源+base64单文件) + docx。数据驱动，避免三引号嵌套。"""
import os, re, base64, html as _html

HERE = os.path.dirname(os.path.abspath(__file__))
SD = os.path.join(HERE, 'screenshots')
CSS = open('/tmp/word-guide-style.css', encoding='utf-8').read()
CSS = CSS.replace('--brand: #1F3A66;', '--brand: #2247A8;').replace('--accent: #C4521E;', '--accent: #1E80EB;')

def esc(s): return _html.escape(s, quote=False)

# ---------- 内容数据 ----------
CONCEPTS = [
    ('对外开放 API', '猎鹰 Studio 是 AI 智能体开发平台。你在平台里搭好「智能体(Bot)」和「工作流(Workflow)」并发布后，业务系统可用 API 调用它们——对话、跑工作流、传文件。'),
    ('PAT 访问令牌', '业务系统用个人访问令牌（PAT，形如一串密钥）以 Authorization: Bearer 调用，无需登录、无需 cookie。在 Studio Web 个人设置里创建。'),
    ('两个核心能力', '① 与智能体对话 /v3/chat（OpenAI/Coze v3 兼容，SSE 流式）；② 运行工作流 /v1/workflow/run（同步）与 stream_run（流式）。'),
    ('业务参数', 'bot_id / workflow_id 从 Studio 编辑页 URL 里取；user_id 由业务方自定义（终端用户唯一标识，用于数据隔离）。'),
]

# 步骤：(tag, title, lead, [items]) ; item=(kind, *args)
STEPS = [
 ('STEP 1','创建并发布智能体','先在 Studio 里搭一个智能体并发布——只有已发布的智能体才能被对外 API 调用。', [
   ('h','① 在「项目开发」创建/打开智能体'),
   ('p','左侧「项目开发」进入智能体列表，新建或打开一个智能体。'),
   ('s','03-project-list.png','项目开发页：智能体列表'),
   ('h','② 编排、调试、发布'),
   ('p','在编排页配置人设/技能/知识，右侧「预览与调试」可即时对话。确认无误后点右上角【发布】。注意浏览器地址栏 URL：/space/.../bot/<bot_id>/arrange，其中的 bot_id 就是调用 /v3/chat 要用的智能体 ID。'),
   ('s','05-bot-edit.png','智能体编排页：右上角【发布】、右侧预览调试、URL 含 bot_id'),
   ('w','⚠️ 必须发布：','/v3/chat 只能调到已发布的智能体（走线上版），草稿调不通。改动后需重新发布生效。'),
 ]),
 ('STEP 2','创建 PAT 访问令牌','业务系统用 PAT 作为 API Key 调用。在个人设置里创建，密钥要妥善保存。', [
   ('h','① 进入「API 授权」'),
   ('p','点左下角用户头像 → 菜单选「API 授权」，进入个人访问令牌管理页。'),
   ('s','01-pat-list.png','API 授权页：个人访问令牌列表'),
   ('h','② 添加新令牌'),
   ('p','点【添加新令牌】，填名称、选过期时间，确定后生成令牌。'),
   ('s','02-pat-create.png','添加新的个人访问令牌：填名称 + 过期时间'),
   ('w','⚠️ 妥善保存：','令牌只在创建时完整显示一次，不要泄露、不要写进前端代码。调用时放 Authorization: Bearer <令牌>。'),
 ]),
 ('STEP 3','调用 /v3/chat 对话','拿到 bot_id + PAT，业务系统即可与智能体对话。默认 SSE 流式返回。', [
   ('h','① 发起对话'),
   ('c','curl -N -X POST "http://<BASE_URL>/v3/chat?conversation_id=" \\\n  -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" \\\n  -d \'{"bot_id":"7620395637741191168","user_id":"biz_user_001","stream":true,"additional_messages":[{"role":"user","content":"你好","content_type":"text"}]}\''),
   ('h','② 处理 SSE 事件流'),
   ('c','event: conversation.message.delta\ndata: {"role":"assistant","content":"你","content_type":"text",...}\n\nevent: conversation.message.completed\ndata: {...完整回复...}\n\nevent: conversation.chat.completed\ndata: {...usage:{token_count,input_count,output_count}}\n\nevent: conversation.stream.done'),
   ('t','💡 增量拼接：','delta 事件的 content 是增量文本，按顺序拼接即得完整回复；chat.completed 带 token 用量。conversation_id 不传会自动新建，在事件里返回，下轮带上即可多轮对话。'),
 ]),
 ('STEP 4','运行工作流','工作流(Workflow)适合固定流程/编排任务。发布后用 workflow_id 调用。', [
   ('h','① 拿 workflow_id'),
   ('p','「资源库 → 工作流」里打开一个工作流，URL 里即其 ID。'),
   ('s','04-workflow-list.png','资源库工作流列表'),
   ('h','② 同步运行'),
   ('c','curl -X POST "http://<BASE_URL>/v1/workflow/run" \\\n  -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" \\\n  -d \'{"workflow_id":"7400000000000000000","parameters":"{\\"input\\":\\"hello\\"}"}\''),
   ('w','⚠️ parameters 是字符串：','工作流入参 parameters 要传「JSON 序列化后的字符串」，不是 JSON 对象。流式运行用 /v1/workflow/stream_run（SSE），中断用 stream_resume 恢复。'),
 ]),
 ('STEP 5','文件上传与会话管理','对话里要带图片/文件，先上传拿到 URI；多轮会话用会话接口管理。', [
   ('h','① 上传文件'),
   ('p','两种方式：/api/bot/upload_file（base64）或 /v1/files/upload（原始二进制）。返回文件 URL/URI，可在对话 content 里引用。'),
   ('c','curl -X POST "http://<BASE_URL>/v1/files/upload" \\\n  -H "Authorization: Bearer <PAT>" -H "Content-Type: image/png" \\\n  --data-binary "@logo.png"'),
   ('h','② 会话管理'),
   ('p','/v1/conversation/create 建会话、/v1/conversations 列会话、/v1/conversation/message/list 列消息、/v1/conversations/:id/clear 清空上下文。详见「API 接口」。'),
   ('t','✅ 到这里链路打通：','发布智能体 → 建 PAT → 调 /v3/chat / workflow → 传文件 / 管会话。完整接口契约见「API 接口」标签，可配合同目录 Postman 集合直接测试。'),
 ]),
]

# API：(method, path, name, group, desc, [req rows], resp_example, curl)
APIS = [
 ('POST','/v3/chat','与智能体对话','对话','与已发布智能体对话（OpenAI/Coze v3 兼容，默认 SSE）。bot_id 在 body，conversation_id 在 query。',
   [['bot_id','i64→string','是(body)','智能体ID，须已发布'],['user_id','string','是(body)','业务方自定义终端用户标识'],['conversation_id','i64→string','否(query)','不传自动新建会话'],['stream','bool','否','默认 true→SSE'],['additional_messages','list','否','本轮消息，role=user，content_type=text/card/object_string'],['custom_variables','map','否','自定义变量，持久化到会话']],
   'event: conversation.message.delta\ndata: {"role":"assistant","content":"增量文本","content_type":"text"}\nevent: conversation.chat.completed\ndata: {"status":"completed","usage":{"token_count":120}}\nevent: conversation.stream.done',
   'curl -N -X POST "http://<BASE_URL>/v3/chat?conversation_id=" -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" -d \'{"bot_id":"73000","user_id":"u1","stream":true,"additional_messages":[{"role":"user","content":"你好","content_type":"text"}]}\''),
 ('POST','/v1/conversation/create','创建会话','对话','创建一个会话。',
   [['bot_id','i64→string','否','关联智能体'],['connector_id','i64→string','否','渠道'],['meta_data','map','否','自定义字段']],
   '{"code":0,"msg":"","data":{"id":"7460...","created_at":1781,"last_section_id":"..."}}',
   'curl -X POST "http://<BASE_URL>/v1/conversation/create" -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" -d \'{"bot_id":"73000"}\''),
 ('GET','/v1/conversations','列出会话','对话','列出某 bot 下的会话。',
   [['bot_id','string','是','智能体ID'],['page_num','int','是','页码'],['page_size','int','是','每页数'],['sort_order','string','否','ASC/DESC'],['sort_field','string','否','如 created_at']],
   '{"code":0,"data":{"conversations":[{"id":"...","created_at":1781}],"has_more":false}}',
   'curl "http://<BASE_URL>/v1/conversations?bot_id=73000&page_num=1&page_size=20" -H "Authorization: Bearer <PAT>"'),
 ('POST','/v1/conversation/message/list','列出会话消息','对话','列出会话内消息。',
   [['conversation_id','string','是','会话ID'],['limit','int','否','条数'],['order','string','否','desc/asc'],['before_id/after_id','string','否','分页游标']],
   '{"code":0,"data":[{"id":"...","role":"user","content":"...","content_type":"text"}],"has_more":false}',
   'curl -X POST "http://<BASE_URL>/v1/conversation/message/list" -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" -d \'{"conversation_id":"7460","limit":20,"order":"desc"}\''),
 ('POST','/v1/conversations/:id/clear','清空会话上下文','对话','清空会话上下文，开启新 section。路径参数 conversation_id。',
   [['conversation_id','string','是(path)','会话ID']],
   '{"code":0,"data":{"id":"section_id","conversation_id":"..."}}',
   'curl -X POST "http://<BASE_URL>/v1/conversations/7460/clear" -H "Authorization: Bearer <PAT>"'),
 ('GET','/v1/bot/get_online_info','查智能体线上信息','对话','查询已发布 bot 的线上配置。',
   [['bot_id','string','是','智能体ID'],['version','string','否','默认最新']],
   '{"code":0,"data":{"name":"...","description":"...","model_info":{...},"plugin_info_list":[...],"workflow_info_list":[...]}}',
   'curl "http://<BASE_URL>/v1/bot/get_online_info?bot_id=73000" -H "Authorization: Bearer <PAT>"'),
 ('POST','/v1/workflow/run','运行工作流','工作流','同步运行工作流。parameters 为 JSON 序列化字符串。',
   [['workflow_id','string','是','工作流ID'],['parameters','string','否','入参JSON字符串(非对象)'],['bot_id','string','否','关联bot'],['is_async','bool','否','异步执行'],['execute_mode','string','否','DEBUG=试运行'],['app_id','string','否','应用ID']],
   '{"code":0,"msg":"","data":"{\\"result\\":\\"...\\"}","token":120,"cost":"0.01","execute_id":"..."}',
   'curl -X POST "http://<BASE_URL>/v1/workflow/run" -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" -d \'{"workflow_id":"7400","parameters":"{\\"input\\":\\"hi\\"}"}\''),
 ('POST','/v1/workflow/stream_run','流式运行工作流','工作流','流式运行工作流（SSE）。请求体同 /v1/workflow/run。',
   [['workflow_id','string','是','工作流ID'],['parameters','string','否','入参JSON字符串']],
   'id: 1\nevent: message\ndata: {"content":"...","node_title":"LLM","node_is_finish":false}\nevent: done',
   'curl -N -X POST "http://<BASE_URL>/v1/workflow/stream_run" -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" -d \'{"workflow_id":"7400","parameters":"{}"}\''),
 ('POST','/v1/workflow/stream_resume','工作流中断恢复','工作流','工作流中断后流式恢复（SSE）。',
   [['event_id','string','是','来自 stream_run 的 interrupt_data.event_id'],['interrupt_type','int','是','1本地插件/2提问/3要素/5输入/7OAuth'],['resume_data','string','是','补充输入'],['workflow_id','string','是','工作流ID']],
   'event: message\ndata: {"content":"...","node_is_finish":true}\nevent: done',
   'curl -N -X POST "http://<BASE_URL>/v1/workflow/stream_resume" -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" -d \'{"workflow_id":"7400","event_id":"e1","interrupt_type":2,"resume_data":"补充"}\''),
 ('GET','/v1/workflow/get_run_history','工作流执行历史','工作流','查工作流执行历史。',
   [['workflow_id','string','是','工作流ID'],['execute_id','string','否','执行ID']],
   '{"code":0,"data":[{"execute_id":"...","execute_status":"Success","output":"...","token":120,"cost":"0.01"}]}',
   'curl "http://<BASE_URL>/v1/workflow/get_run_history?workflow_id=7400" -H "Authorization: Bearer <PAT>"'),
 ('POST','/v1/workflows/chat','与对话流对话','工作流','与 chatflow（对话流）对话（SSE）。',
   [['workflow_id','string','是','对话流ID'],['additional_messages','list','否','本轮消息'],['parameters','string','否','入参'],['conversation_id','string','否','会话ID']],
   'event: message\ndata: {...}\nevent: done',
   'curl -N -X POST "http://<BASE_URL>/v1/workflows/chat" -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" -d \'{"workflow_id":"7400","additional_messages":[{"role":"user","content":"你好","content_type":"text"}]}\''),
 ('POST','/api/bot/upload_file','上传文件(base64)','文件','上传文件（base64 方式）。',
   [['file_head.file_type','string','是','后缀如 png/pdf'],['file_head.biz_type','int','是','1=BOT_ICON,2=BOT_DATASET...'],['data','string','是','文件 base64 字符串']],
   '{"code":0,"data":{"upload_url":"https://...","upload_uri":"..."}}',
   'curl -X POST "http://<BASE_URL>/api/bot/upload_file" -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" -d "{\\"file_head\\":{\\"file_type\\":\\"png\\",\\"biz_type\\":1},\\"data\\":\\"$(base64 -i logo.png)\\"}"'),
 ('POST','/v1/files/upload','上传文件(原始二进制)','文件','上传文件（OpenAPI 原始二进制）。Content-Type 用文件 MIME。',
   [['(body)','binary','是','原始文件二进制（非base64）']],
   '{"code":0,"data":{"id":"...","uri":"...","url":"https://...","bytes":12345,"file_name":"logo.png"}}',
   'curl -X POST "http://<BASE_URL>/v1/files/upload" -H "Authorization: Bearer <PAT>" -H "Content-Type: image/png" --data-binary "@logo.png"'),
 ('POST','/api/playground/upload/auth_token','获取OSS直传凭证','文件','获取 OSS 直传临时凭证（大文件直传）。',
   [['scene','string','是','场景，如 bot'],['data_type','string','否','数据类型']],
   '{"code":0,"data":{"upload_host":"...","auth":{"access_key_id":"...","session_token":"...","expired_time":1781}}}',
   'curl -X POST "http://<BASE_URL>/api/playground/upload/auth_token" -H "Authorization: Bearer <PAT>" -H "Content-Type: application/json" -d \'{"scene":"bot"}\''),
]

OPS = [('03-project-list.png','项目开发：智能体列表'),('04-workflow-list.png','资源库：工作流'),('01-pat-list.png','API 授权：个人访问令牌')]

print('内容数据就绪：concepts=%d steps=%d apis=%d' % (len(CONCEPTS), len(STEPS), len(APIS)))
open('/tmp/studio_css.txt','w',encoding='utf-8').write(CSS)
import json
json.dump({'CONCEPTS':CONCEPTS,'STEPS':STEPS,'APIS':APIS,'OPS':OPS}, open('/tmp/studio_data.json','w',encoding='utf-8'), ensure_ascii=False)
print('数据已存 /tmp/studio_data.json')
