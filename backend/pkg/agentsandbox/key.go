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
func SandboxKeyFor(connectorID, agentID int64, userID string) string {
	raw := fmt.Sprintf("%d_%d_%s", connectorID, agentID, userID)
	sum := sha256.Sum256([]byte(raw))
	return "u" + hex.EncodeToString(sum[:])[:24]
}
