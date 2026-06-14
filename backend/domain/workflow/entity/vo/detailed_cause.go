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

package vo

import (
	workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

// SanitizedCauseMsg release 态对外的脱敏占位（绝不含原始错误文本）。
const SanitizedCauseMsg = "internal error, please check the workflow configuration or contact the administrator"

// CauseForMode 按执行模式返回错误 cause 文案：
// debug/node_debug 返回详细根因（含 HTTP 状态+厂商报错），release 返回脱敏文案。
func CauseForMode(mode workflowModel.ExecuteMode, err error) string {
	if mode == workflowModel.ExecuteModeDebug || mode == workflowModel.ExecuteModeNodeDebug {
		return DetailedCause(err)
	}
	return SanitizedCauseMsg
}

// DetailedCause 返回用于调试展示的详细根因文本：
// 错误链中若存在 ModelCallError，优先用其富化文本（含 HTTP 状态码 + 厂商报错）；
// 否则回退到最深层根因。
func DetailedCause(err error) string {
	if err == nil {
		return ""
	}
	if mce, ok := chatmodel.AsModelCallError(err); ok {
		return mce.Error()
	}
	return UnwrapRootErr(err).Error()
}
