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
	"testing"
)

func TestTurnCanvasLedger_RecordAndRender(t *testing.T) {
	l := newTurnCanvasLedger()
	l.record(turnCanvasLedgerOp{Op: "add_node", NodeTag: "probe", Type: "3", Title: "探针节点"})
	l.record(turnCanvasLedgerOp{Op: "configure_node", NodeTag: "probe", Detail: `{"prompt":"复述"}`})
	l.record(turnCanvasLedgerOp{Op: "connect", From: "start", To: "probe"})

	out := l.renderForAgent()
	for _, want := range []string{"add_node", "probe", "type=3", "探针节点", "configure_node", "connect", "start->probe"} {
		if !strings.Contains(out, want) {
			t.Fatalf("render 缺少 %q,实际: %s", want, out)
		}
	}
}

func TestTurnCanvasLedger_ClearResets(t *testing.T) {
	l := newTurnCanvasLedger()
	l.record(turnCanvasLedgerOp{Op: "add_node", NodeTag: "a", Type: "3"})
	l.record(turnCanvasLedgerOp{Op: "clear_canvas"})
	l.record(turnCanvasLedgerOp{Op: "add_node", NodeTag: "b", Type: "5"})

	out := l.renderForAgent()
	if strings.Contains(out, "node_tag=a") {
		t.Fatalf("clear 后不应再含清空前的节点 a,实际: %s", out)
	}
	if !strings.Contains(out, "node_tag=b") {
		t.Fatalf("clear 后新增的 b 应保留,实际: %s", out)
	}
	if !strings.Contains(out, "clear_canvas") {
		t.Fatalf("应提示本轮清空过画布,实际: %s", out)
	}
}

func TestTurnCanvasLedger_NilAndEmptySafe(t *testing.T) {
	var l *turnCanvasLedger
	l.record(turnCanvasLedgerOp{Op: "add_node", NodeTag: "x"}) // 不应 panic
	if l.renderForAgent() != "" {
		t.Fatal("nil ledger 应返回空串")
	}
	if newTurnCanvasLedger().renderForAgent() != "" {
		t.Fatal("空 ledger 应返回空串")
	}
}
