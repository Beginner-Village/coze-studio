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

package coze

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

type superAgentUIConfigData struct {
	// SuperAgentEnabled 报告是否向前端暴露「超级智能体」入口（创建入口 / 超级体编辑器 /
	// 工作流自动搭建面板）。与后端沙箱总开关 SANDBOX_ENABLED 联动，运维改一个 env 即可，
	// 前端无需重新构建。
	SuperAgentEnabled bool `json:"super_agent_enabled"`
}

type superAgentUIConfigResponse struct {
	Code int64                   `json:"code"`
	Msg  string                  `json:"msg"`
	Data *superAgentUIConfigData `json:"data"`
}

// SuperAgentUIConfig 暴露一份供前端运行时决定超级体入口显隐的最小配置。
// 无鉴权（同 manifest / openapi.json），任何前端页面启动时都可拉取。
// @router /api/super-agent/ui-config [GET]
func SuperAgentUIConfig(_ context.Context, c *app.RequestContext) {
	c.JSON(http.StatusOK, &superAgentUIConfigResponse{
		Code: 0,
		Msg:  "success",
		Data: &superAgentUIConfigData{
			SuperAgentEnabled: superAgentUIEnabled(),
		},
	})
}

// superAgentUIEnabled 与 appinfra.sandboxEnabledByConfig 采用同一判定：默认启用，
// 仅当 SANDBOX_ENABLED 显式为 false/0/no/off/disabled 时关闭。二者共读同一个 env，
// 保证「后端沙箱是否启用」与「前端是否展示超级体入口」始终一致。
func superAgentUIEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SANDBOX_ENABLED"))) {
	case "false", "0", "no", "off", "disabled":
		return false
	default:
		return true
	}
}
