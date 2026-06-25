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
	"runtime"
	"sync"
	"testing"
)

// lockSandboxFile must serialize concurrent writers to the SAME (sandbox,path) so a
// read-modify-write (like edit_file on USER.md/MEMORY.md) can't lose updates. Run with
// -race to also prove there's no data race on the shared state.
func TestLockSandboxFile_SerializesSameFile(t *testing.T) {
	const n = 100
	counter := 0
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := lockSandboxFile("sbx-key", "/workspace/.agent/USER.md")
			defer unlock()
			c := counter // read-modify-write under lock
			runtime.Gosched()
			counter = c + 1
		}()
	}
	wg.Wait()
	if counter != n {
		t.Fatalf("lock failed to serialize same-file writers: got %d, want %d", counter, n)
	}
}

// Different files (or different sandboxes) use independent locks and must not deadlock
// when held simultaneously.
func TestLockSandboxFile_DistinctFilesIndependent(t *testing.T) {
	u1 := lockSandboxFile("sbx-key", "/workspace/a.txt")
	u2 := lockSandboxFile("sbx-key", "/workspace/b.txt")
	u3 := lockSandboxFile("other-key", "/workspace/a.txt")
	u3()
	u2()
	u1()
}
