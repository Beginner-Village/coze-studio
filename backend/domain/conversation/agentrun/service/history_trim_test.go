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

package agentrun

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestTrimHistoryByTokenBudgetDisabled(t *testing.T) {
	h := []*schema.Message{{Role: schema.User, Content: "a"}, {Role: schema.Assistant, Content: "b"}}
	got := trimHistoryByTokenBudget(h, 0)
	if len(got) != 2 {
		t.Fatalf("budget=0 should not trim, got %d", len(got))
	}
}

func TestTrimHistoryByTokenBudgetKeepsNewest(t *testing.T) {
	big := strings.Repeat("x", 3000) // ~1000 tokens each
	h := []*schema.Message{
		{Role: schema.User, Content: big},      // oldest
		{Role: schema.Assistant, Content: big}, //
		{Role: schema.User, Content: big},      // newest
	}
	got := trimHistoryByTokenBudget(h, 1500) // fits ~1 message
	if len(got) == 0 || len(got) >= 3 {
		t.Fatalf("expected trimming to keep newest subset, got %d", len(got))
	}
	// the kept message must be the newest one
	if got[len(got)-1] != h[2] {
		t.Fatalf("newest message must be kept")
	}
}

func TestTrimHistoryDropsLeadingOrphanToolMsg(t *testing.T) {
	big := strings.Repeat("x", 3000)
	h := []*schema.Message{
		{Role: schema.Assistant, Content: big},                       // oldest (will be trimmed)
		{Role: schema.Tool, Content: "tool result", ToolCallID: "c1"}, // becomes leading orphan after trim
		{Role: schema.User, Content: big},                             // newest
	}
	got := trimHistoryByTokenBudget(h, 1500)
	for _, m := range got {
		if m == got[0] && m.Role == schema.Tool {
			t.Fatalf("result must not start with an orphan tool message")
		}
	}
	if len(got) > 0 && got[0].Role == schema.Tool {
		t.Fatalf("leading tool message should have been dropped")
	}
}
