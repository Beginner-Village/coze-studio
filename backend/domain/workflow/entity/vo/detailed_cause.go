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

import "github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"

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
