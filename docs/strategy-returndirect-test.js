/* =============================================================================
 * 策略·工作流 returnDirectly(工作流文本直出) 行为测试脚本
 * =============================================================================
 * 目的:检测「绑定策略的智能体执行 workflow 能力」时,工作流返回的文本是
 *      ① 作为 tool_as_answer 直接回给用户(returnDirectly 直出),还是
 *      ② 作为 tool_response 回传给模型、由模型转述(model-relayed)。
 *
 * 判别信号(后端 message type):
 *   - tool_as_answer  → 工作流文本「直出」(EventTypeOfToolsAsChatModelStream)
 *   - answer          → 模型生成/转述的文本
 *   - tool_response   → 工具结果回传给模型
 *
 * 运行方式:
 *   1. 用测试账号登录 http://10.10.10.226:8896 (402087139@qq.com / 123456)
 *   2. F12 → Console,粘贴本文件全部内容,回车
 *   3. 看打印的判别结果
 *
 * ⚠️ 当前实测结论(2026-06-27, 后端含 marker 机制):
 *   即使 force_tool_return=0(开关关闭),tool_as_answer 也【不触发】。
 *   根因:returnDirectly 的「直出」渲染绑定在 eino agent 的「终止态」上;
 *   策略用单一 run 工具承载所有能力,run 未进 eino 的 build-time returnDirectly
 *   集合,所以 agent 不终止、继续调模型 → 工作流文本最终由【模型转述】给用户。
 *   这与「多意图链式(查余额→转账,需要不终止继续)」存在架构层冲突,二者用
 *   同一个 run 工具时无法兼得。本脚本用于持续验证该行为/未来修复后转绿。
 * ========================================================================== */

const CFG = { origin: location.origin, bot: '7654425189890916352', space: '7652614054615187456', timeoutMs: 80000 };

async function chatCapture(query) {
  const H = { 'content-type': 'application/json', 'x-requested-with': 'XMLHttpRequest' };
  const body = { bot_id: CFG.bot, conversation_id: '0', query, scene: 4, draft_mode: true,
    content_type: 'text', local_message_id: 't' + Date.now() + Math.random(), space_id: CFG.space, extra: {} };
  const resp = await fetch(CFG.origin + '/api/conversation/chat',
    { method: 'POST', credentials: 'include', headers: H, body: JSON.stringify(body) });
  if (!resp.ok || !resp.body) throw new Error('chat http ' + resp.status);
  const reader = resp.body.getReader(), dec = new TextDecoder();
  let buf = '', ev = null; const types = {};
  let toolAsAnswer = '', answer = ''; const runOutputs = [];
  const t0 = Date.now();
  while (true) {
    const { done, value } = await reader.read();
    if (done || Date.now() - t0 > CFG.timeoutMs) break;
    buf += dec.decode(value, { stream: true });
    const lines = buf.split('\n'); buf = lines.pop() || '';
    for (const l of lines) {
      if (l.startsWith('event:')) ev = l.slice(6).trim();
      else if (l.startsWith('data:')) {
        const d = l.slice(5).trim(); if (ev === 'done') continue;
        let c; try { c = JSON.parse(d); } catch (e) { continue; }
        const m = c.message || {}; const ty = m.type || '?';
        types[ty] = (types[ty] || 0) + 1;
        if (ty === 'tool_as_answer') toolAsAnswer += (m.content || '');
        else if (ty === 'answer') answer += (m.content || '');
        else if (ty === 'tool_response') runOutputs.push((m.content || '').replace(/\s+/g, ' '));
      }
    }
  }
  return { types, toolAsAnswer: toolAsAnswer.replace(/\s+/g, ' '),
    answer: answer.replace(/\s+/g, ' '), runOutputs };
}

async function run() {
  console.log('%c=== returnDirectly 行为测试 ===', 'font-weight:bold;color:#2563eb');
  const query = '帮我给测试收款人转账300元';
  console.log('查询:', query);
  const r = await chatCapture(query);

  const directFired = (r.types['tool_as_answer'] || 0) > 0;
  const workflowRan = r.runOutputs.some(o => /转账|SN\d/.test(o)) || /SN\d/.test(r.answer);

  console.log('消息类型统计:', r.types);
  console.log('%c工作流是否真实执行: ' + (workflowRan ? '是 ✅' : '否 ❌'),
    'color:' + (workflowRan ? '#16a34a' : '#dc2626'));
  console.log('%creturnDirectly 直出 (tool_as_answer): ' + (directFired ? '触发 ✅' : '未触发 ❌'),
    'font-weight:bold;color:' + (directFired ? '#16a34a' : '#d97706'));

  if (directFired) {
    console.log('  → 直出内容:', r.toolAsAnswer.slice(0, 200));
    console.log('%c结论: 工作流文本以 returnDirectly 方式【直出】给用户。', 'color:#16a34a');
  } else {
    console.log('  → 模型转述(answer):', r.answer.slice(0, 200));
    console.log('%c结论: 工作流文本由【模型转述】(非直出)。', 'color:#d97706');
    console.log('%c  说明: 这是当前架构的已知行为 —— 见文件头注释。', 'color:#6b7280');
    console.log('%c  若要切换为「工具结果回传模型」语义,可在智能体「工具结果回传模型」开关中打开。', 'color:#6b7280');
  }
  return r;
}

run();
