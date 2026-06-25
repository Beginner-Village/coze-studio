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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// SandboxKeyFor 由 connector/agent/user_id 组合出稳定且容器名安全的沙箱 key。
// 这是超级智能体「每个 (连接器, 智能体, 用户) 一个独立沙箱空间」的寻址依据，
// agent 运行时(agentflow)与沙箱空间管理 API 必须用同一套算法,才能对上同一个沙箱。
//
// key 形如 "a<agentID>-u<hash>":把 agentID 作为可还原前缀编入,这样删除智能体时
// 可按前缀 "a<agentID>-" 枚举出该智能体下「所有用户/所有连接器」的容器与数据目录,
// 一次性清理干净(详见 SandboxKeyPrefixForAgent)。
func SandboxKeyFor(connectorID, agentID int64, userID string) string {
	raw := fmt.Sprintf("%d_%d_%s", connectorID, agentID, userID)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("a%d-u%s", agentID, hex.EncodeToString(sum[:])[:20])
}

// SandboxKeyPrefixForAgent 返回某 agentID 所有沙箱 key 的公共前缀(含尾部 "-"),
// 用于按前缀枚举/清理该智能体下全部用户的沙箱容器与数据目录。
func SandboxKeyPrefixForAgent(agentID int64) string {
	return fmt.Sprintf("a%d-", agentID)
}
