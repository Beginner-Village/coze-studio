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
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The per-sandbox semaphore must cap concurrent holders at the limit and never let more
// than `limit` run at once. Run with -race to prove the slot bookkeeping is race-free.
func TestAcquireExecSlot_CapsConcurrency(t *testing.T) {
	const key = "sbx-limit-test"
	limit := execLimit()

	var inFlight int32
	var maxSeen int32
	var wg sync.WaitGroup
	for i := 0; i < limit*4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := acquireExecSlot(context.Background(), key)
			if err != nil {
				t.Errorf("unexpected acquire error: %v", err)
				return
			}
			cur := atomic.AddInt32(&inFlight, 1)
			for {
				prev := atomic.LoadInt32(&maxSeen)
				if cur <= prev || atomic.CompareAndSwapInt32(&maxSeen, prev, cur) {
					break
				}
			}
			time.Sleep(2 * time.Millisecond)
			atomic.AddInt32(&inFlight, -1)
			release()
		}()
	}
	wg.Wait()

	if maxSeen > int32(limit) {
		t.Fatalf("concurrency exceeded limit: saw %d concurrent, limit %d", maxSeen, limit)
	}
	if maxSeen == 0 {
		t.Fatalf("no concurrency observed; test is not exercising the semaphore")
	}
}

// A cancelled context while waiting for a full slot must return an error (rejection),
// not block forever.
func TestAcquireExecSlot_RejectsOnCtxCancel(t *testing.T) {
	const key = "sbx-reject-test"
	limit := execLimit()

	// Fill all slots and hold them.
	releases := make([]func(), 0, limit)
	for i := 0; i < limit; i++ {
		release, err := acquireExecSlot(context.Background(), key)
		if err != nil {
			t.Fatalf("acquire %d failed: %v", i, err)
		}
		releases = append(releases, release)
	}
	defer func() {
		for _, r := range releases {
			r()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	if _, err := acquireExecSlot(ctx, key); err == nil {
		t.Fatalf("acquire on a full sandbox with cancelled ctx must return an error")
	}
}
