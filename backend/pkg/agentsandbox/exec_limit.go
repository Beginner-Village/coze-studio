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

package agentsandbox

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ynet-dev/ynet-studio/backend/pkg/observability"
)

// P1 hardening: cap concurrent sandbox Exec per sandbox (per user/agent/connector key)
// so a single user can't exhaust node resources with a burst of parallel commands. The
// limit is a blocking semaphore — callers wait for a slot rather than being dropped, and
// only fail if their context is cancelled/times out while waiting.
const defaultMaxConcurrentExecPerSandbox = 8

var (
	execSlots       sync.Map // key -> chan struct{} (buffered, cap = limit)
	execLimitOnce   sync.Once
	execLimitPerSbx int
)

func execLimit() int {
	execLimitOnce.Do(func() {
		execLimitPerSbx = defaultMaxConcurrentExecPerSandbox
		if v := strings.TrimSpace(os.Getenv("SANDBOX_EXEC_MAX_CONCURRENCY")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				execLimitPerSbx = n
			}
		}
	})
	return execLimitPerSbx
}

// acquireExecSlot blocks until a concurrency slot for this sandbox is free (capping
// per-sandbox parallel Exec), returning a release func. If ctx is cancelled while waiting
// for a slot it returns ctx.Err() and records a rejection metric.
func acquireExecSlot(ctx context.Context, key string) (func(), error) {
	chI, _ := execSlots.LoadOrStore(key, make(chan struct{}, execLimit()))
	ch := chI.(chan struct{})
	select {
	case ch <- struct{}{}:
		return func() { <-ch }, nil
	case <-ctx.Done():
		observability.SandboxExecRejectedTotal.Inc()
		return nil, ctx.Err()
	}
}
