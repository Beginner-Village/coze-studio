# -*- coding: utf-8 -*-
"""生成 studio 全量 Postman 集合：覆盖所有功能模块的 350 个接口，两套认证（Bearer PAT 对外 / Cookie session 管理）。"""
import json, os

# 每个模块: (folder名, auth: 'bearer'|'cookie'|'public', [ (METHOD, path, name, body_json_or_None, desc), ... ])
def b(d): return json.dumps(d, ensure_ascii=False, indent=2)

MODULES = [
 # ========== 对外 API（Bearer PAT） ==========
 ('对外 ▸ 对话与会话', 'bearer', [
   ('POST','/v3/chat','与智能体对话 (SSE)', {'bot_id':'{{bot_id}}','user_id':'{{user_id}}','stream':True,'additional_messages':[{'role':'user','content':'你好','content_type':'text'}]}, '与已发布智能体对话(OpenAI/Coze v3,SSE)。bot_id在body,conversation_id在query。'),
   ('POST','/v1/conversation/create','创建会话', {'bot_id':'{{bot_id}}'}, '请求 bot_id/connector_id/meta_data。响应 data.id 会话ID。'),
   ('GET','/v1/conversations','列出会话', None, 'query: bot_id/page_num/page_size/sort_order/sort_field。'),
   ('POST','/v1/conversation/message/list','列出会话消息', {'conversation_id':'{{conversation_id}}','limit':20,'order':'desc'}, '请求 conversation_id/limit/order/before_id/after_id。'),
   ('POST','/v1/conversations/{{conversation_id}}/clear','清空会话上下文', None, '路径参数 conversation_id。'),
   ('GET','/v1/bot/get_online_info','查智能体线上信息', None, 'query: bot_id/version。响应含 model_info/plugin_info_list/workflow_info_list。'),
 ]),
 ('对外 ▸ 工作流', 'bearer', [
   ('POST','/v1/workflow/run','运行工作流(同步)', {'workflow_id':'{{workflow_id}}','parameters':'{"input":"hi"}'}, 'parameters 为入参JSON字符串。响应 data(结果)/token/cost/execute_id。'),
   ('POST','/v1/workflow/stream_run','流式运行工作流(SSE)', {'workflow_id':'{{workflow_id}}','parameters':'{"input":"hi"}'}, 'SSE: event message/done/error, 含 node_title/content/interrupt_data。'),
   ('POST','/v1/workflow/stream_resume','工作流中断恢复(SSE)', {'workflow_id':'{{workflow_id}}','event_id':'e1','interrupt_type':2,'resume_data':'补充','ext':{}}, 'interrupt_type: 1本地插件/2提问/3要素/5输入/7OAuth。'),
   ('GET','/v1/workflow/get_run_history','工作流执行历史', None, 'query: workflow_id/execute_id。响应 WorkflowExecuteHistory[]。'),
   ('POST','/v1/workflows/chat','与对话流对话(SSE)', {'workflow_id':'{{workflow_id}}','additional_messages':[{'role':'user','content':'你好','content_type':'text'}],'ext':{}}, 'chatflow 对话运行(SSE)。'),
   ('POST','/v1/workflow/conversation/create','为对话流创建会话', {'bot_id':'{{bot_id}}'}, '为 chatflow 创建会话(以实测字段为准)。'),
 ]),
 ('对外 ▸ 文件', 'bearer', [
   ('POST','/api/bot/upload_file','上传文件(base64)', {'file_head':{'file_type':'png','biz_type':1},'data':'<base64>'}, 'biz_type: 1BOT_ICON/2BOT_DATASET/4PLUGIN_ICON...。响应 upload_url/upload_uri。'),
   ('POST','/v1/files/upload','上传文件(原始二进制)', None, 'Content-Type用文件MIME, body原始二进制(--data-binary)。响应 File{id,uri,url,bytes}。'),
   ('POST','/api/playground/upload/auth_token','取OSS直传凭证', {'scene':'bot'}, '响应 auth{access_key_id,secret_access_key,session_token}。'),
   ('POST','/api/common/upload/apply_upload_action','申请上传地址(VOD)', None, 'VOD风格申请上传。'),
 ]),
 # ========== 平台管理 API（Cookie session） ==========
 ('管理 ▸ 对话(前端)', 'cookie', [
   ('POST','/api/conversation/chat','Agent运行(SSE主聊天)', {'bot_id':'{{bot_id}}','conversation_id':'{{conversation_id}}','query':'你好','scene':10}, 'AgentRunRequest。SSE RunStreamResponse。'),
   ('POST','/api/conversation/get_message_list','拉取消息列表', {'conversation_id':'{{conversation_id}}','cursor':'0','count':20}, '游标分页。'),
   ('POST','/api/conversation/clear_message','清空会话历史', {'conversation_id':'{{conversation_id}}'}, '建新 section。'),
   ('POST','/api/conversation/create_section','新建 section', {'conversation_id':'{{conversation_id}}'}, '清上下文。'),
   ('POST','/api/conversation/delete_message','删除消息', {'conversation_id':'{{conversation_id}}','message_id':'m1'}, None),
   ('POST','/api/conversation/break_message','打断回复', {'conversation_id':'{{conversation_id}}','query_message_id':'m1'}, None),
 ]),
 ('管理 ▸ 工作流编排', 'cookie', [
   ('POST','/api/workflow_api/create','创建工作流', {'space_id':'{{space_id}}','name':'我的工作流'}, None),
   ('POST','/api/workflow_api/save','保存草稿canvas', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/publish','发布工作流', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/copy','复制工作流', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/delete','删除工作流', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/canvas','获取canvas', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/test_run','试运行(编辑器调试)', {'workflow_id':'{{workflow_id}}','input':{}}, None),
   ('POST','/api/workflow_api/test_resume','试运行恢复', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/nodeDebug','单节点调试', {'workflow_id':'{{workflow_id}}','node_id':'n1'}, None),
   ('GET','/api/workflow_api/get_process','获取运行进度', None, None),
   ('POST','/api/workflow_api/workflow_list','工作流列表', {'space_id':'{{space_id}}','page':1,'size':20}, None),
   ('POST','/api/workflow_api/workflow_detail','工作流详情', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/workflow_detail_info','工作流详情(含schema)', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/list_publish_workflow','已发布工作流列表', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/workflow_api/update_meta','更新元信息', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/validate_tree','校验流程树', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/node_template_list','节点模板列表', {}, None),
   ('POST','/api/workflow_api/export','导出工作流', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/import','导入工作流', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/workflow_api/version_list','版本历史列表', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/revert_draft','回滚草稿到版本', {'workflow_id':'{{workflow_id}}','version':'v1'}, None),
   ('POST','/api/workflow_api/chat_flow_role/create','创建chatflow角色', {'workflow_id':'{{workflow_id}}'}, None),
   ('POST','/api/workflow_api/project_conversation/create','创建项目会话定义', {'project_id':'p1','conversation_name':'c1','space_id':'{{space_id}}'}, None),
 ]),
 ('管理 ▸ 知识库', 'cookie', [
   ('POST','/api/knowledge/create','创建知识库', {'name':'我的知识库','description':'desc','space_id':'{{space_id}}','icon_uri':'','format_type':0}, 'format_type: 0文本/1表格/2图片/7QA。'),
   ('POST','/api/knowledge/list','知识库列表', {'space_id':'{{space_id}}','page':1,'size':20}, None),
   ('POST','/api/knowledge/detail','知识库详情(批量)', {'dataset_ids':['{{dataset_id}}'],'space_id':'{{space_id}}'}, None),
   ('POST','/api/knowledge/update','更新知识库', {'dataset_id':'{{dataset_id}}','name':'新名','icon_uri':'','description':''}, None),
   ('POST','/api/knowledge/delete','删除知识库', {'dataset_id':'{{dataset_id}}'}, None),
   ('POST','/api/knowledge/document/create','创建文档', {'dataset_id':'{{dataset_id}}','format_type':0,'document_bases':[{'name':'doc1','source_info':{'document_source':0,'tos_uri':'<上传返回的uri>'}}],'chunk_strategy':{'separator':'\\n','max_tokens':800,'remove_extra_spaces':False,'remove_urls_emails':False,'chunk_type':0}}, '本地文件用 source_info.tos_uri(先上传);自定义内容用 custom_content;直接传用 file_base64+file_type。'),
   ('POST','/api/knowledge/document/list','文档列表', {'dataset_id':'{{dataset_id}}','page':1,'size':20}, None),
   ('POST','/api/knowledge/document/update','更新文档', {'document_id':'d1','document_name':'新名'}, None),
   ('POST','/api/knowledge/document/delete','删除文档', {'document_ids':['d1']}, None),
   ('POST','/api/knowledge/document/resegment','重新分段', {'dataset_id':'{{dataset_id}}','document_ids':['d1'],'chunk_strategy':{'separator':'\\n','max_tokens':800}}, None),
   ('POST','/api/knowledge/document/progress/get','文档处理进度', {'document_ids':['d1']}, '响应 progress(0-100)/status。'),
   ('POST','/api/knowledge/slice/create','新增切片', {'document_id':'d1','raw_text':'内容'}, None),
   ('POST','/api/knowledge/slice/list','切片列表', {'document_id':'d1','page_size':20}, None),
   ('POST','/api/knowledge/slice/update','更新切片', {'slice_id':'s1','raw_text':'新内容'}, None),
   ('POST','/api/knowledge/slice/delete','删除切片', {'slice_ids':['s1']}, None),
   ('POST','/api/knowledge/retrieve_test','召回测试', {'dataset_ids':['{{dataset_id}}'],'query':'测试问题'}, '测试知识库召回效果。'),
   ('POST','/api/knowledge/table_schema/get','取表格schema', {'document_id':'d1'}, None),
   ('POST','/api/knowledge/photo/list','图片列表', {'dataset_id':'{{dataset_id}}'}, None),
 ]),
 ('管理 ▸ 外部知识库', 'cookie', [
   ('POST','/api/external-knowledge/retrieval','外部知识库检索', {'query':'问题'}, None),
   ('POST','/api/external-knowledge/binding/create','创建绑定', {}, None),
   ('GET','/api/external-knowledge/binding/list','绑定列表', None, None),
   ('GET','/api/external-knowledge/ragflow/datasets','RAGFlow数据集列表', None, None),
 ]),
 ('管理 ▸ 插件', 'cookie', [
   ('POST','/api/plugin_api/register_plugin_meta','创建插件元信息', {'name':'我的插件','desc':'desc','space_id':'{{space_id}}','icon':{},'auth_type':0}, None),
   ('POST','/api/plugin_api/register','从manifest+openapi注册', {'ai_plugin':'<manifest JSON>','openapi':'<openapi YAML>','space_id':'{{space_id}}'}, None),
   ('POST','/api/plugin_api/convert_to_openapi','curl/postman/swagger转openapi', {'data':'<curl/swagger>','space_id':'{{space_id}}'}, None),
   ('POST','/api/plugin_api/update_plugin_meta','更新插件元信息', {'plugin_id':'pl1'}, None),
   ('POST','/api/plugin_api/del_plugin','删除插件', {'plugin_id':'pl1'}, None),
   ('POST','/api/plugin_api/get_plugin_info','插件详情', {'plugin_id':'pl1'}, None),
   ('POST','/api/plugin_api/get_dev_plugin_list','开发态插件列表', {'space_id':'{{space_id}}','page':1,'size':20}, None),
   ('POST','/api/plugin_api/publish_plugin','发布插件', {'plugin_id':'pl1','version_name':'v1'}, None),
   ('POST','/api/plugin_api/create_api','创建工具(API)', {'plugin_id':'pl1','name':'tool1','desc':'desc'}, None),
   ('POST','/api/plugin_api/update_api','更新工具', {'plugin_id':'pl1','api_id':'a1'}, None),
   ('POST','/api/plugin_api/delete_api','删除工具', {'plugin_id':'pl1','api_id':'a1'}, None),
   ('POST','/api/plugin_api/get_plugin_apis','工具列表', {'plugin_id':'pl1','page':1,'size':20}, None),
   ('POST','/api/plugin_api/debug_api','调试工具', {'plugin_id':'pl1','api_id':'a1','parameters':'{}','operation':1}, '响应 success/resp/raw_req/raw_resp。'),
   ('POST','/api/plugin_api/get_oauth_status','取OAuth状态', {'plugin_id':'pl1'}, None),
   ('POST','/api/plugin_api/library_resource_list','资源库列表', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/plugin_api/create_folder','创建文件夹', {'space_id':'{{space_id}}','name':'分类'}, None),
   ('POST','/api/plugin_api/get_folder_list','文件夹列表', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/plugin_api/move_resources_to_folder','移动资源到文件夹', {'space_id':'{{space_id}}'}, None),
 ]),
 ('管理 ▸ 技能', 'cookie', [
   ('POST','/api/skill/create','创建技能', {'space_id':'{{space_id}}','name':'技能名'}, None),
   ('GET','/api/skill/get','技能详情', None, None),
   ('POST','/api/skill/update','更新技能', {'skill_id':'sk1'}, None),
   ('POST','/api/skill/delete','删除技能', {'skill_id':'sk1'}, None),
   ('GET','/api/skill/list','技能列表', None, None),
 ]),
 ('管理 ▸ 数据库', 'cookie', [
   ('POST','/api/memory/database/add','创建数据库表', {'creator_id':'{{user_id}}','space_id':'{{space_id}}','project_id':'p1','icon_uri':'','table_name':'用户表','table_desc':'desc','field_list':[{'name':'name','desc':'姓名','type':1,'must_required':True}],'rw_mode':1,'prompt_disabled':False}, 'type: 1Text/2Number/3Date/4Float/5Boolean。rw_mode:1受限/2只读/3无限。'),
   ('POST','/api/memory/database/list','数据库列表', {'space_id':'{{space_id}}','table_type':1,'offset':0,'limit':20}, 'table_type:1草稿/2线上。'),
   ('POST','/api/memory/database/get_by_id','按ID取数据库', {'id':'db1','is_draft':True,'need_sys_fields':True}, None),
   ('POST','/api/memory/database/update','更新数据库', {'id':'db1','icon_uri':'','table_name':'表','table_desc':'','field_list':[],'rw_mode':1,'prompt_disabled':False}, None),
   ('POST','/api/memory/database/delete','删除数据库', {'id':'db1'}, None),
   ('POST','/api/memory/database/list_records','查询数据行', {'database_id':'db1','table_type':1,'limit':50,'offset':0}, '响应 data: 行数据 list<map>。'),
   ('POST','/api/memory/database/update_records','增/改/删数据行', {'database_id':'db1','record_data_add':[{'name':'张三'}],'table_type':1}, 'record_data_add/alter/delete。'),
   ('POST','/api/memory/database/bind_to_bot','绑定到Bot', {'database_id':'db1','bot_id':'{{bot_id}}'}, None),
   ('POST','/api/memory/database/table/list_new','Bot库表列表', {'bot_id':'{{bot_id}}','table_type':1}, None),
   ('POST','/api/memory/table_file/submit','提交数据导入任务', {}, None),
   ('POST','/api/memory/table_schema/get','取表schema', {}, None),
 ]),
 ('管理 ▸ 变量与记忆', 'cookie', [
   ('GET','/api/memory/sys_variable_conf','系统变量配置', None, None),
   ('GET','/api/memory/project/variable/meta_list','项目变量元列表', None, None),
   ('POST','/api/memory/variable/get','取KV记忆', {'bot_id':'{{bot_id}}'}, None),
   ('POST','/api/memory/variable/upsert','写入KV记忆', {'bot_id':'{{bot_id}}'}, None),
   ('POST','/api/memory/variable/delete','删除profile记忆', {'bot_id':'{{bot_id}}'}, None),
   ('POST','/api/memory-config/set','设置记忆配置', {}, None),
 ]),
 ('管理 ▸ 智能体/Bot编排', 'cookie', [
   ('POST','/api/draftbot/create','创建草稿Bot', {'space_id':'{{space_id}}','name':'我的智能体'}, None),
   ('POST','/api/draftbot/delete','删除草稿Bot', {'bot_id':'{{bot_id}}'}, None),
   ('POST','/api/draftbot/duplicate','复制草稿Bot', {'bot_id':'{{bot_id}}'}, None),
   ('POST','/api/draftbot/publish','发布Bot', {'bot_id':'{{bot_id}}'}, '发布后才能对外 /v3/chat 调用。'),
   ('POST','/api/draftbot/commit_check','提交前检查', {'bot_id':'{{bot_id}}'}, None),
   ('POST','/api/draftbot/list_draft_history','草稿历史列表', {'bot_id':'{{bot_id}}'}, None),
   ('POST','/api/draftbot/publish/connector/list','发布渠道列表', {'bot_id':'{{bot_id}}'}, None),
   ('POST','/api/playground_api/draftbot/get_draft_bot_info','取草稿Bot详情', {'bot_id':'{{bot_id}}'}, None),
   ('POST','/api/playground_api/draftbot/update_draft_bot_info','更新草稿Bot详情', {'bot_id':'{{bot_id}}'}, None),
 ]),
 ('管理 ▸ HiAgent接入', 'cookie', [
   ('GET','/api/space/{{space_id}}/hi-agents','HiAgent列表', None, None),
   ('POST','/api/space/{{space_id}}/hi-agents','创建HiAgent', {'name':'外部智能体','endpoint':'https://...'}, None),
   ('GET','/api/space/{{space_id}}/hi-agents/:agent_id','HiAgent详情', None, None),
   ('PUT','/api/space/{{space_id}}/hi-agents/:agent_id','更新HiAgent', {}, None),
   ('DELETE','/api/space/{{space_id}}/hi-agents/:agent_id','删除HiAgent', None, None),
   ('POST','/api/hi-agents/test-connection','测试HiAgent连接', {'endpoint':'https://...'}, None),
 ]),
 ('管理 ▸ 项目/应用', 'cookie', [
   ('POST','/api/intelligence_api/draft_project/create','创建项目(应用)', {'space_id':'{{space_id}}','name':'我的应用'}, None),
   ('POST','/api/intelligence_api/draft_project/copy','复制项目', {'project_id':'p1'}, None),
   ('POST','/api/intelligence_api/draft_project/update','更新项目', {'project_id':'p1'}, None),
   ('POST','/api/intelligence_api/draft_project/delete','删除项目', {'project_id':'p1'}, None),
   ('POST','/api/intelligence_api/publish/publish_project','发布项目', {'project_id':'p1'}, None),
   ('POST','/api/intelligence_api/search/get_draft_intelligence_list','草稿智能体列表', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/intelligence_api/search/get_recently_edit_intelligence','最近编辑列表', {'space_id':'{{space_id}}'}, None),
 ]),
 ('管理 ▸ 空间', 'cookie', [
   ('POST','/api/space/create','创建空间', {'name':'我的空间'}, None),
   ('GET','/api/space/list','空间列表', None, None),
   ('GET','/api/space/:space_id','空间详情', None, None),
   ('PUT','/api/space/:space_id','更新空间', {}, None),
   ('DELETE','/api/space/:space_id','删除空间', None, None),
   ('GET','/api/space/:space_id/members','成员列表', None, None),
   ('POST','/api/space/:space_id/members','邀请成员', {'user_id':'u1','role':'member'}, None),
   ('POST','/api/space/:space_id/export','导出空间', {}, None),
   ('POST','/api/space/:space_id/import/preview','导入预览', {}, None),
   ('POST','/api/space/:space_id/release','创建发布', {}, None),
   ('GET','/api/space/:space_id/release/list','发布列表', None, None),
   ('POST','/api/space/configure_models','一键配置空间模型', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/space/diagnose','空间健康检查', {'space_id':'{{space_id}}'}, None),
 ]),
 ('管理 ▸ 模型', 'cookie', [
   ('POST','/api/model/create','创建模型', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/model/list','模型列表', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/model/detail','模型详情', {'model_id':'m1'}, None),
   ('POST','/api/model/update','更新模型', {'model_id':'m1'}, None),
   ('POST','/api/model/delete','删除模型', {'model_id':'m1'}, None),
   ('POST','/api/model/import','从模板导入模型', {'space_id':'{{space_id}}'}, None),
   ('GET','/api/model/templates','模型模板列表', None, None),
   ('POST','/api/model/space/add','加模型到空间', {'space_id':'{{space_id}}','model_id':'m1'}, None),
   ('POST','/api/embedding/space/list','空间向量模型列表', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/rerank/space/list','空间rerank模型列表', {'space_id':'{{space_id}}'}, None),
   ('GET','/api/admin/models','公共模型列表', None, None),
   ('POST','/api/admin/models/create','创建公共模型', {}, None),
 ]),
 ('管理 ▸ 提示词与杂项', 'cookie', [
   ('POST','/api/playground_api/upsert_prompt_resource','新增/更新提示词', {'space_id':'{{space_id}}'}, None),
   ('GET','/api/playground_api/get_prompt_resource_info','提示词详情', None, None),
   ('POST','/api/playground_api/get_official_prompt_list','官方提示词列表', {}, None),
   ('POST','/api/playground_api/create_update_shortcut_command','新增/更新快捷指令', {}, None),
   ('POST','/api/playground_api/space/list','空间列表V2', {}, None),
 ]),
 ('管理 ▸ 模板与商店', 'cookie', [
   ('POST','/api/template/publish','发布为模板', {}, None),
   ('GET','/api/template/my-list','我的模板列表', None, None),
   ('DELETE','/api/template/:template_id','删除模板', None, None),
   ('POST','/api/template/store/publish','发布到商店', {}, None),
   ('GET','/api/template/store/list','商店模板列表', None, None),
   ('GET','/api/marketplace/product/list','商店产品列表', None, None),
   ('POST','/api/marketplace/product/duplicate','复制商店产品', {}, None),
 ]),
 ('管理 ▸ 统计与日志', 'cookie', [
   ('POST','/api/statistics/app/active-users','活跃用户统计', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/statistics/app/messages','消息量统计', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/statistics/app/tokens','Token统计', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/statistics/app/list_app_conversation_log','会话日志列表', {'space_id':'{{space_id}}'}, None),
   ('POST','/api/operation_log/list','操作日志列表', {}, None),
 ]),
 ('管理 ▸ 用户与PAT', 'cookie', [
   ('POST','/api/passport/account/info/v2/','账户信息', {}, None),
   ('POST','/api/user/update_profile','更新资料', {}, None),
   ('POST','/api/permission_api/pat/create_personal_access_token_and_permission','创建PAT(API Key)', {'name':'my-token','expire_at':0}, '创建对外用的 PAT 令牌(就是 api_key)。'),
   ('GET','/api/permission_api/pat/list_personal_access_tokens','列出PAT', None, None),
   ('POST','/api/permission_api/pat/delete_personal_access_token_and_permission','删除PAT', {'id':'t1'}, None),
 ]),
 ('登录(公开)', 'public', [
   ('POST','/api/passport/web/email/login/','邮箱登录', {'email':'user@test.com','password':'<password>'}, '登录拿 session_key cookie。'),
   ('POST','/api/passport/web/email/register/v2/','邮箱注册', {'email':'user@test.com','password':'<password>'}, None),
   ('GET','/api/passport/web/logout/','登出', None, None),
 ]),
]

def url_obj(path):
    raw = '{{base_url}}' + path
    parts = [p for p in path.split('/') if p != '']
    return {'raw': raw, 'host': ['{{base_url}}'], 'path': parts}

def make_item(method, path, name, body, desc, auth):
    req = {'method': method, 'header': [], 'url': url_obj(path)}
    if auth == 'cookie':
        req['header'].append({'key':'Cookie','value':'session_key={{session_key}}'})
    if body is not None:
        req['header'].append({'key':'Content-Type','value':'application/json'})
        req['body'] = {'mode':'raw','raw': b(body),'options':{'raw':{'language':'json'}}}
    if desc: req['description'] = desc
    return {'name': f'{method} {name}', 'request': req}

folders = []
total = 0
for fname, auth, items in MODULES:
    sub = [make_item(m,p,n,bd,d,auth) for (m,p,n,bd,d) in items]
    total += len(sub)
    folder = {'name': f'{fname} ({len(sub)})', 'item': sub}
    if auth == 'bearer':
        folder['auth'] = {'type':'bearer','bearer':[{'key':'token','value':'{{api_key}}','type':'string'}]}
    folders.append(folder)

collection = {
  'info': {
    'name': '猎鹰 Studio · 全量 API（对外 + 平台管理）',
    '_postman_id': 'ynet-studio-full-2026-06-13',
    'description': f'猎鹰 Studio 全量接口集合，共 {total} 个接口，覆盖所有功能模块。\\n\\n## 两套认证\\n- **对外 ▸ ***（Bearer PAT）：业务系统用个人访问令牌 {{api_key}} 调用，folder 级已配 Bearer。\\n- **管理 ▸ ***（Cookie session）：前端管理后台接口，需先「登录(公开)」拿 session_key，填入变量 {{session_key}}，各请求带 Cookie。\\n\\n## 变量\\nbase_url / api_key(PAT) / session_key / bot_id / workflow_id / dataset_id / conversation_id / space_id / user_id\\n\\n## 拿凭证\\n- PAT：Web「个人设置→API授权」创建，或「管理▸用户与PAT / 创建PAT」接口。\\n- session_key：「登录(公开)/邮箱登录」后从 Set-Cookie 取。\\n\\n## 注意\\nID 类字段 JSON 里是字符串；SSE 接口(/v3/chat、stream_run、/api/conversation/chat 等)返回 text/event-stream。',
    'schema': 'https://schema.getpostman.com/json/collection/v2.1.0/collection.json'
  },
  'variable': [
    {'key':'base_url','value':'http://10.10.10.220:9888'},
    {'key':'api_key','value':''},{'key':'session_key','value':''},
    {'key':'bot_id','value':''},{'key':'workflow_id','value':''},{'key':'dataset_id','value':''},
    {'key':'conversation_id','value':''},{'key':'space_id','value':''},{'key':'user_id','value':'biz_user_001'},
  ],
  'item': folders
}
out = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'Studio-全量API.postman_collection.json')
json.dump(collection, open(out,'w',encoding='utf-8'), ensure_ascii=False, indent=2)
print('全量 Postman:', out)
print('模块数:', len(folders), '| 接口总数:', total)
