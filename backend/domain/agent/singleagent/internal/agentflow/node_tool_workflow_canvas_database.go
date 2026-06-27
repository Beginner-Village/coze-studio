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

// 画布插件/数据库类工具共享的小工具。
//
// 能力 E(数据库建表 + mock 行)暂未在此实现:application/memory 服务会经
// memory→search→singleagent→agent-service 反向 import 本 internal 包,形成导入环,
// 不能从 agentflow 直接调。正确做法是给 crossdatabase 合约加 CreateDatabase/
// AddDatabaseRecord/ListDatabase(包当前 domain 服务已具备),再在工具里走
// crossdatabase.DefaultSVC()——作为 B/C/D 验证通过后的紧接着的下一步。

import (
	"encoding/json"
	"strconv"
)

// wfCanvasSpaceID 优先取前端 Ext 传入的画布空间,否则回退 deps.SpaceID(B/C 复用)。
func wfCanvasSpaceID(deps superAgentToolDeps) int64 {
	if deps.Ext != nil {
		if s := wfExtValue(deps.Ext, workflowCanvasSpaceIDExtKey); s != "" {
			if id, err := strconv.ParseInt(s, 10, 64); err == nil && id != 0 {
				return id
			}
		}
	}
	return deps.SpaceID
}

func wfCanvasDBErr(msg string) string {
	b, _ := json.Marshal(map[string]string{"status": "error", "message": msg})
	return string(b)
}
