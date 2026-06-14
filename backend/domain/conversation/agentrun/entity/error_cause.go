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

package entity

import "github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"

// SanitizedCauseMsg 非调试态对外的脱敏占位（绝不含原始错误文本/厂商报错）。
const SanitizedCauseMsg = "internal error, please check the agent configuration or contact the administrator"

// CauseForDebug 按是否调试态返回智能体运行错误的对外文案：
// 调试态（搭建者在编排页调试，isDraft=true）返回详细根因（含 HTTP 状态 + 厂商报错），
// 非调试态（线上/已发布对话）返回脱敏文案，避免泄露厂商原始错误。
func CauseForDebug(isDebug bool, err error) string {
	if err == nil {
		return ""
	}
	if isDebug {
		return detailedCause(err)
	}
	return SanitizedCauseMsg
}

// detailedCause 返回用于调试展示的详细根因文本：
// 错误链中若存在 ModelCallError，优先用其富化文本（含 HTTP 状态码 + 厂商报错）；
// 否则回退到原始错误文本。
func detailedCause(err error) string {
	if mce, ok := chatmodel.AsModelCallError(err); ok {
		return mce.Error()
	}
	return err.Error()
}
