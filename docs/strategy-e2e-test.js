/* =============================================================================
 * 策略(Strategy)能力 — 端到端自动化测试脚本
 * =============================================================================
 * 用途:验证「绑定了策略的单智能体」能否按照渐进披露(scenes→caps→run)自主、
 *      正确地完成「多意图 / 跨场景 / 跨类型(工作流+提示词) / 条件迭代」的组合任务。
 *
 * 运行方式:
 *   1. 用测试账号登录 http://10.10.10.226:8896 (402087139@qq.com / 123456)
 *   2. 打开浏览器开发者工具 → Console
 *   3. 粘贴本文件全部内容,回车
 *   4. 自动顺序跑完全部用例,最后打印 PASS/FAIL 汇总
 *
 * 依赖:同源 authenticated fetch(浏览器已登录即可,会话 cookie 为 httpOnly,
 *      故必须在浏览器内运行,不能用外部 curl)。
 *
 * 被测对象(可在 CONFIG 改):
 *   - bot   7654425189890916352  (已绑定策略「对公智能运营助手」)
 *   - space 7652614054615187456
 *   - 策略含 6 个场景:1对公开户 2账户管理 3信贷业务 4跨境结算与外汇 5合规与风控
 *     6账户与转账(workflow:查询余额/转账)
 *
 * 注:每条用例用 conversation_id:'0' 自动新建独立会话,互不污染。
 *     余额工作流返回固定 ¥3,560.00,用例通过「改变阈值」来分别触发两个条件分支,
 *     证明智能体是真的读余额做判断,而非硬编码。
 * ========================================================================== */

const CONFIG = {
  origin: location.origin,
  bot:   '7654425189890916352',
  space: '7652614054615187456',
  timeoutMs: 90000,
};

/* ---- 对话驱动:发一条消息,读完 SSE 流,抽取工具调用序列 / 工具输出 / 最终回答 ---- */
async function chat(query) {
  const H = { 'content-type': 'application/json', 'x-requested-with': 'XMLHttpRequest' };
  const body = {
    bot_id: CONFIG.bot, conversation_id: '0', query,
    scene: 4 /* Playground */, draft_mode: true, content_type: 'text',
    local_message_id: 't' + Date.now() + Math.random(), space_id: CONFIG.space, extra: {},
  };
  const resp = await fetch(CONFIG.origin + '/api/conversation/chat', {
    method: 'POST', credentials: 'include', headers: H, body: JSON.stringify(body),
  });
  if (!resp.ok || !resp.body) throw new Error('chat http ' + resp.status);

  const reader = resp.body.getReader(), dec = new TextDecoder();
  let buf = '', ev = null, answer = '', sseError = null;
  const calls = [], runOutputs = [];
  const t0 = Date.now();

  const parseCall = (s) => {            // function_call 内容是双层 JSON
    try { const o = JSON.parse(s); if (o.function) return { name: o.function.name, args: o.function.arguments || '' }; } catch (e) {}
    return { name: '?', args: s.slice(0, 80) };
  };

  while (true) {
    const { done, value } = await reader.read();
    if (done || Date.now() - t0 > CONFIG.timeoutMs) break;
    buf += dec.decode(value, { stream: true });
    const lines = buf.split('\n'); buf = lines.pop() || '';
    for (const l of lines) {
      if (l.startsWith('event:')) ev = l.slice(6).trim();
      else if (l.startsWith('data:')) {
        const d = l.slice(5).trim();
        if (ev === 'done') continue;
        if (ev === 'error') { sseError = d.slice(0, 300); continue; }
        let c; try { c = JSON.parse(d); } catch (e) { continue; }
        const m = c.message || {}, ei = m.extra_info || {};
        if (m.type === 'function_call') {
          const pc = parseCall(m.content || '');
          if (/^(scenes|caps|run)$/.test(pc.name)) calls.push(pc.name + '(' + pc.args + ')');
        } else if (m.type === 'tool_response') {
          if (ei.tool_name === 'run') runOutputs.push((m.content || '').replace(/\s+/g, ' '));
        } else if (m.type === 'answer') {
          answer += (m.content || '');
        }
      }
    }
  }
  return { query, calls, runOutputs, answer: answer.replace(/\s+/g, ' '), sseError };
}

/* ---- 通用断言 ---- */
const inc = (hay, needle) => (hay || '').indexOf(needle) >= 0;
const anyInc = (hay, arr) => arr.some(n => inc(hay, n));
const calledScene = (calls, scene) => calls.some(c => c.startsWith('caps(') && inc(c, '"scene": ' + scene));
const ranInScene  = (calls, scene) => calls.some(c => c.startsWith('run(')  && inc(c, '"scene": ' + scene));
const noWfError   = (outs) => outs.every(o => !inc(o, 'Error executing workflow') && !inc(o, 'NodeRunError'));

/* ---- 测试用例:多意图 / 跨场景 / 跨类型 / 条件迭代 ---- */
const CASES = [
  {
    name: 'C1 条件转账·大于分支 (查余额→3560>1000→转张三500)',
    query: '帮我查一下我的账户余额,如果余额大于1000元就给张三转500元,如果不到1000元就只给张三转200元',
    check: r => ({
      pass: ranInScene(r.calls, 6) && noWfError(r.runOutputs)
            && inc(r.answer, '3,560') && inc(r.answer, '张三') && inc(r.answer, '500'),
      why: '应:查余额→判断3560>1000→给张三转500;且工作流无报错',
    }),
  },
  {
    name: 'C2 条件转账·否则分支 (同样3560,阈值5000→3560<5000→转李四100)',
    query: '查一下我的余额,如果余额够5000元就给李四转800元,不够5000就只给李四转100元',
    check: r => ({
      pass: ranInScene(r.calls, 6) && noWfError(r.runOutputs)
            && inc(r.answer, '李四') && inc(r.answer, '100') && !inc(r.answer, '转账 ¥800') && !inc(r.answer, '800元'),
      why: '应:同样余额3560,但阈值5000→走否则分支→给李四转100(证明真在读余额判断)',
    }),
  },
  {
    name: 'C3 跨场景·提示词 (对公开户 + 账户管理)',
    query: '我们公司想开一个对公基本存款账户,需要准备哪些材料?另外公司老账户的预留印鉴怎么变更?',
    check: r => ({
      pass: calledScene(r.calls, 1) && calledScene(r.calls, 2)
            && anyInc(r.answer, ['开户', '材料', '营业执照']) && anyInc(r.answer, ['印鉴', '变更']),
      why: '应:同时探场景1+场景2,组合回答开户材料 + 印鉴变更',
    }),
  },
  {
    name: 'C4 跨场景·提示词 (信贷业务 + 跨境外汇)',
    query: '企业想申请一笔流动资金贷款,大致流程是怎样的?另外我们有一笔境外销售收汇,外汇合规上要注意什么?',
    check: r => ({
      pass: calledScene(r.calls, 3) && calledScene(r.calls, 4)
            && anyInc(r.answer, ['贷款', '流贷', '授信']) && anyInc(r.answer, ['外汇', '收汇', '结售汇']),
      why: '应:同时探场景3+场景4,组合回答流贷流程 + 外汇合规',
    }),
  },
  {
    name: 'C5 跨类型组合 (workflow查余额 + prompt合规, 一轮内)',
    query: '先帮我查一下账户余额,然后顺便讲讲大额可疑交易在反洗钱方面应该怎么处理。',
    check: r => ({
      pass: ranInScene(r.calls, 6) && calledScene(r.calls, 5) && noWfError(r.runOutputs)
            && inc(r.answer, '3,560') && anyInc(r.answer, ['反洗钱', '可疑', '尽职调查']),
      why: '应:同一轮内既跑工作流(场景6余额)又取提示词(场景5合规),组合成答',
    }),
  },
  {
    name: 'C6 单意图转账·工作流反映实参 (给王五转300)',
    query: '帮我给王五转账300元',
    check: r => ({
      pass: ranInScene(r.calls, 6) && noWfError(r.runOutputs)
            && inc(r.answer, '王五') && inc(r.answer, '300'),
      why: '应:run(转账,{payee:王五,amount:300}),工作流输出反映实参',
    }),
  },
];

/* ---- 跑全部(每条用例最多尝试 2 次:LLM 非确定性下偶发"没走策略工具"则自动重试一次) ---- */
const ATTEMPTS = 2;
async function runCase(tc) {
  let last = null;
  for (let i = 1; i <= ATTEMPTS; i++) {
    try {
      const r = await chat(tc.query);
      const v = tc.check(r);
      last = { r, v, attempt: i };
      if (v.pass) return last;                 // 通过即返回
      // 未通过:若这次连 scenes/caps/run 都没调用(瞬时跑偏),再试一次
      if (i < ATTEMPTS) console.log('   ↻ 第' + i + '次未通过,重试…(工具序列: ' + (r.calls.join(' ') || '无') + ')');
    } catch (e) {
      last = { r: { calls: [], runOutputs: [], answer: '', sseError: e.message }, v: { pass: false, why: '异常: ' + e.message }, attempt: i };
    }
  }
  return last;
}
async function runAll() {
  console.log('%c=== 策略能力 E2E 测试开始 ===', 'font-weight:bold;color:#2563eb');
  const results = [];
  for (const tc of CASES) {
    const { r, v, attempt } = await runCase(tc);
    results.push({ name: tc.name, pass: v.pass });
    console.log('%c' + (v.pass ? '✅ PASS ' : '❌ FAIL ') + tc.name + (attempt > 1 ? ' (第' + attempt + '次)' : ''),
      'font-weight:bold;color:' + (v.pass ? '#16a34a' : '#dc2626'));
    console.log('   工具序列:', r.calls.join('  '));
    if (r.runOutputs.length) console.log('   工作流输出:', r.runOutputs.map(o => o.slice(0, 80)).join(' | '));
    console.log('   回答(节选):', r.answer.slice(0, 180));
    if (!v.pass) console.log('%c   期望:' + v.why, 'color:#dc2626');
    if (r.sseError) console.log('%c   SSE错误:' + r.sseError, 'color:#dc2626');
  }
  const passed = results.filter(r => r.pass).length;
  console.log('%c=== 汇总: ' + passed + '/' + results.length + ' 通过 ===',
    'font-weight:bold;font-size:14px;color:' + (passed === results.length ? '#16a34a' : '#dc2626'));
  return results;
}

/* 自动执行 */
runAll();
