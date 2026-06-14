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

package chatmodel

import "fmt"

// ModelCallError 富化后的模型调用错误：携带 HTTP 状态码与厂商原始报错。
// 作为错误链上的一环（Unwrap 返回底层 Raw），其 Error() 文本已包含状态码与厂商消息，
// 供调试态就地展示。生产态由上层决定是否脱敏，不直接回显本类型内容。
type ModelCallError struct {
	HTTPStatus      int    // 0 表示未解析到
	ProviderMessage string // 厂商返回的 message/body（可能为空）
	Raw             error  // 原始底层错误
}

func (e *ModelCallError) Error() string {
	switch {
	case e.HTTPStatus > 0 && e.ProviderMessage != "":
		return fmt.Sprintf("model call failed [HTTP %d]: %s", e.HTTPStatus, e.ProviderMessage)
	case e.HTTPStatus > 0:
		return fmt.Sprintf("model call failed [HTTP %d]: %v", e.HTTPStatus, e.Raw)
	case e.ProviderMessage != "":
		return fmt.Sprintf("model call failed: %s", e.ProviderMessage)
	default:
		return fmt.Sprintf("model call failed: %v", e.Raw)
	}
}

func (e *ModelCallError) Unwrap() error { return e.Raw }

// AsModelCallError 在错误链中查找 ModelCallError。
func AsModelCallError(err error) (*ModelCallError, bool) {
	for err != nil {
		if mce, ok := err.(*ModelCallError); ok {
			return mce, true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return nil, false
		}
		err = u.Unwrap()
	}
	return nil, false
}
