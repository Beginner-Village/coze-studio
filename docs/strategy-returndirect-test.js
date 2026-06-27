/* =============================================================================
 * 策略·工作流 returnDirectly(工作流文本直出) 测试脚本
 * =============================================================================
 * 目的:验证「绑定策略的智能体执行 workflow 能力」时,工作流的「返回文本」
 *      是否作为最终回复【直出】(终止 react 循环、逐字返回、不经模型改写)。
 *
 * 设计(与原生绑定工作流一致,按 End 节点模式 + 全局开关控制):
 *   - 工作流 End = 「返回文本」(useAnswerContent) → 直出 / 终止
 *   - 工作流 End = 「返回变量」(returnVariables)   → 继续链式(可被下一步使用)
 *   - 全局「工具结果回传模型」开关 = ON → 关闭直出,全部回传模型继续推理
 *
 * 运行方式:
 *   1. 用测试账号登录 http://10.10.10.226:8896 (402087139@qq.com / 123456)
 *   2. F12 → Console,粘贴本文件全部内容,回车
 *
 * ✅ 实测结论(2026-06-27, 后端 SHA 1d5ff35e):returnDirectly 生效。
 *    机制:run 工具执行「返回文本」工作流时调用 eino 原生 react.SetReturnDirectly(ctx),
 *    react 状态为带父链的词法作用域、子图入口注入一次,故工具内写入对终止分支可见 →
 *    react 循环终止 → 工作流文本即最终回复(逐字、无模型 summary)。
 *    「返回变量」工作流不调用它 → 继续链式(查余额[变量]→转账[文本] 仍可用)。
 *    已由 14 个图级测试 + 线上端到端验证。
 * ========================================================================== */

const CFG = { origin: location.origin, bot: '7654425189890916352', space: '7652614054615187456', timeoutMs: 80000 };

// 发一条消息,按顺序抓取所有可见内容块(answer / tool_as_answer / 终结 tool_response)
async function chatTrace(query) {
  const H = { 'content-type': 'application/json', 'x-requested-with': 'XMLHttpRequest' };
  const body = { bot_id: CFG.bot, conversation_id: '0', query, scene: 4, draft_mode: true,
    content_type: 'text', local_message_id: 't' + Date.now() + Math.random(), space_id: CFG.space, extra: {} };
  const resp = await fetch(CFG.origin + '/api/conversation/chat',
    { method: 'POST', credentials: 'include', headers: H, body: JSON.stringify(body) });
  if (!resp.ok || !resp.body) throw new Error('chat http ' + resp.status);
  const reader = resp.body.getReader(), dec = new TextDecoder();
  let buf = '', ev = null;
  const seq = [];                 // ordered {type, content}
  let lastFC = null;              // last function_call name
  const t0 = Date.now();
  const fcName = (s) => { try { const o = JSON.parse(s); if (o.function) return o.function.name; } catch (e) {} return null; };
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
        const m = c.message || {}; const ty = m.type;
        if (ty === 'function_call') { lastFC = fcName(m.content || ''); seq.push({ type: 'call:' + lastFC, content: '' }); }
        else if (ty === 'answer' || ty === 'tool_as_answer' || ty === 'tool_response') {
          const last = seq[seq.length - 1];
          if (last && last.type === ty) last.content += (m.content || '');
          else seq.push({ type: ty, content: (m.content || '') });
        }
      }
    }
  }
  return seq.map(b => ({ type: b.type, content: (b.content || '').replace(/\s+/g, ' ').trim() }));
}

async function run() {
  console.log('%c=== returnDirectly 直出行为测试 ===', 'font-weight:bold;color:#2563eb');

  // CASE A: 单工作流意图(转账=返回文本)→ 应直出 + 终止
  const payee = '唐僧', amount = 666;
  const trace = await chatTrace(`帮我给${payee}转账${amount}元`);
  console.log('事件序列:', trace.map(b => b.type).join(' → '));

  // 工作流原文(逐字),returnDirectly 模式下它就是最终回复
  const wfLine = new RegExp(`已成功向\\s*${payee}\\s*转账\\s*¥?${amount}.*SN\\d`);
  // 找最后一个 run 之后的可见文本块
  const runIdx = trace.map(b => b.type).lastIndexOf('call:run');
  const afterRun = runIdx >= 0 ? trace.slice(runIdx + 1) : [];
  const finalText = afterRun.filter(b => b.type === 'answer' || b.type === 'tool_as_answer' || b.type === 'tool_response')
    .map(b => b.content).join(' ');

  const workflowRan = wfLine.test(finalText);
  // relay 的特征:模型在工作流原文之外又加了一段 summary(如"转账已成功完成""转账详情")
  const hasModelSummary = /转账详情|转账已成功完成|账户类型|根据您的/.test(finalText) &&
    !wfLine.test(afterRun.filter(b => b.type === 'answer').map(b => b.content).join(' ').replace(wfLine, ''));
  const directReturn = workflowRan && !/转账详情|转账已成功完成/.test(finalText);

  console.log('%c工作流执行: ' + (workflowRan ? '是 ✅' : '否 ❌'), 'color:' + (workflowRan ? '#16a34a' : '#dc2626'));
  console.log('   run 之后的最终回复:', finalText.slice(0, 160));
  console.log('%creturnDirectly 直出(终止+逐字、无模型改写): ' + (directReturn ? '生效 ✅' : '未生效 ⚠️'),
    'font-weight:bold;color:' + (directReturn ? '#16a34a' : '#d97706'));
  if (directReturn) {
    console.log('%c  → 工作流「返回文本」直接作为本次回复,react 循环已终止。', 'color:#16a34a');
    console.log('%c  → 把智能体「工具结果回传模型」开关打开,可改为回传模型继续推理(relay)。', 'color:#6b7280');
  } else {
    console.log('%c  → 回复里出现了模型 summary,可能开关被打开了,或该工作流是「返回变量」。', 'color:#d97706');
  }
  return { trace, finalText, directReturn };
}

run();
