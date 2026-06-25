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

import "sync"

// 生产硬化：同一沙箱文件的并发写会互相覆盖/损坏（典型受害者是长期记忆 USER.md/MEMORY.md
// 与 context-summary.json —— 主 run、复盘 fork、上下文压缩可能同时写）。这里用进程内
// per-(sandboxKey,path) 互斥锁把「同一个文件」的写串行化；不同文件、不同沙箱仍并发。
//
// 说明：这是单实例锁。多副本部署需换成分布式锁（Redis），但当前 super-agent 单容器运行，
// 进程内锁已消除绝大多数并发写损坏。read-modify-write 的 edit_file 尤其依赖它保持原子。
var sandboxFileLocks sync.Map // key = sandboxKey + "\x00" + path  ->  *sync.Mutex

// lockSandboxFile 锁定某个沙箱文件并返回解锁函数（defer 调用）。
func lockSandboxFile(sandboxKey, path string) func() {
	k := sandboxKey + "\x00" + path
	mu, _ := sandboxFileLocks.LoadOrStore(k, &sync.Mutex{})
	m := mu.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}
