/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agentflow

import (
	"strings"
	"sync"
)

// turnCanvasLedger 记录【本轮】(单条用户消息内)编排 agent 已经成功下发的每一个画布
// 编辑操作(add_node/connect/configure_node/delete/...)。
//
// 为什么需要它:画布编辑走"聊天 FuncCall 流 → 前端画布桥"(channel A)异步应用,落到
// 浏览器 document 上有延迟;而 get_canvas_context 读画布走 relay(channel B)或读旧快照。
// 两条通道解耦,导致 agent 在【同一轮】里刚 add_node 后立刻读画布【一定看不到自己刚加的
// 节点】——无论读的是实时画布还是旧快照。agent 因此误判"操作没生效"、重做或 clear_canvas。
//
// ledger 是【后端自己的权威记录】:agent 每调一次画布写工具,后端就在这里记一笔。
// get_canvas_context 把本轮 ledger 一并返回,agent 就能看到"我本轮已经下发过哪些操作",
// 不再依赖滞后的画布读取来确认自己的编辑。这与网络是否抖动、前端应用是否滞后都无关。
//
// 生命周期:BuildAgent 每条用户消息都全新构造 superAgentToolDeps,故 ledger 天然 per-turn,
// 无需手动重置;同一轮内所有画布工具实例共享同一个 *turnCanvasLedger 指针。
type turnCanvasLedger struct {
	mu      sync.Mutex
	ops     []turnCanvasLedgerOp
	cleared bool // 本轮是否调用过 clear_canvas(调用后之前的节点与旧快照都已作废)
}

type turnCanvasLedgerOp struct {
	Op      string // add_node / connect / configure_node / set_node_params / delete_node / delete_line / clear_canvas
	NodeTag string
	Type    string // add_node 的节点类型
	Title   string
	From    string // connect/delete_line
	To      string
	Detail  string // configure_node 的配置摘要等
}

func newTurnCanvasLedger() *turnCanvasLedger {
	return &turnCanvasLedger{}
}

// record 记录一次成功下发的画布操作;nil receiver 安全(未接线时静默)。
// clear_canvas 会清掉之前累计的操作并置 cleared,因为清空后那些节点都不存在了。
func (l *turnCanvasLedger) record(op turnCanvasLedgerOp) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if op.Op == "clear_canvas" {
		l.cleared = true
		l.ops = l.ops[:0]
		return
	}
	l.ops = append(l.ops, op)
}

// renderForAgent 把本轮 ledger 渲染成给 agent 看的一段权威文本;无操作且未清空时返回空串。
func (l *turnCanvasLedger) renderForAgent() string {
	if l == nil {
		return ""
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.ops) == 0 && !l.cleared {
		return ""
	}
	var b strings.Builder
	if l.cleared {
		b.WriteString("[本轮你已 clear_canvas 清空过画布:之前的节点和旧快照都已作废,画布现在只剩 Start/End 加上下面这些你清空后新建的]\n")
	}
	for i, op := range l.ops {
		b.WriteString(itoa(i + 1))
		b.WriteString(") ")
		b.WriteString(op.Op)
		if op.NodeTag != "" {
			b.WriteString(" node_tag=")
			b.WriteString(op.NodeTag)
		}
		if op.Type != "" {
			b.WriteString(" type=")
			b.WriteString(op.Type)
		}
		if op.Title != "" {
			b.WriteString(" title=")
			b.WriteString(op.Title)
		}
		if op.From != "" || op.To != "" {
			b.WriteString(" ")
			b.WriteString(op.From)
			b.WriteString("->")
			b.WriteString(op.To)
		}
		if op.Detail != "" {
			b.WriteString(" ")
			b.WriteString(op.Detail)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// wfTruncate 按 rune 截断字符串(避免截断破坏 UTF-8 中文),超长时补省略号。
func wfTruncate(s string, max int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// itoa 避免引入 strconv 仅为个位数序号。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
