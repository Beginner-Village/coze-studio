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

package conversation

import (
	"context"
	"strings"
	"sync"
	"time"
)

var activeAgentRuns sync.Map

type ActiveAgentRunState struct {
	RunID     string `json:"run_id"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type activeAgentRunEntry struct {
	state  ActiveAgentRunState
	cancel context.CancelFunc
}

func RegisterActiveAgentRun(runID string, cancel context.CancelFunc) func() {
	runID = strings.TrimSpace(runID)
	if runID == "" || cancel == nil {
		return func() {}
	}
	now := time.Now().UnixMilli()
	activeAgentRuns.Store(runID, &activeAgentRunEntry{
		state: ActiveAgentRunState{
			RunID:     runID,
			Status:    "in_progress",
			CreatedAt: now,
			UpdatedAt: now,
		},
		cancel: cancel,
	})
	return func() {
		activeAgentRuns.Delete(runID)
	}
}

func GetActiveAgentRun(runID string) (ActiveAgentRunState, bool) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return ActiveAgentRunState{}, false
	}
	runAny, ok := activeAgentRuns.Load(runID)
	if !ok {
		return ActiveAgentRunState{}, false
	}
	switch run := runAny.(type) {
	case *activeAgentRunEntry:
		if run == nil {
			return ActiveAgentRunState{}, false
		}
		return run.state, true
	case context.CancelFunc:
		now := time.Now().UnixMilli()
		return ActiveAgentRunState{
			RunID:     runID,
			Status:    "in_progress",
			CreatedAt: now,
			UpdatedAt: now,
		}, true
	default:
		return ActiveAgentRunState{}, false
	}
}

func CancelActiveAgentRun(runID string) bool {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return false
	}
	cancelAny, ok := activeAgentRuns.LoadAndDelete(runID)
	if !ok {
		return false
	}
	switch run := cancelAny.(type) {
	case *activeAgentRunEntry:
		if run == nil || run.cancel == nil {
			return false
		}
		run.cancel()
		return true
	case context.CancelFunc:
		if run == nil {
			return false
		}
		run()
		return true
	default:
		return false
	}
}
