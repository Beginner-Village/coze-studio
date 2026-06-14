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

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

var statusCodeRe = regexp.MustCompile(`(?i)status(?:\s+code)?[:\s]+(\d{3})`)

// extractModelHTTPError 从各家 eino-ext provider 的错误里尽力解析 HTTP 状态码与厂商消息。
// 解析不到状态码时返回 0；消息至少回退为 err.Error()。
func extractModelHTTPError(err error) (status int, providerMsg string) {
	if err == nil {
		return 0, ""
	}
	s := err.Error()
	if m := statusCodeRe.FindStringSubmatch(s); len(m) == 2 {
		status, _ = strconv.Atoi(m[1])
	}
	// 优先取 "message: ..." 之后的部分作为厂商消息
	if idx := strings.Index(strings.ToLower(s), "message:"); idx >= 0 {
		providerMsg = strings.TrimSpace(s[idx+len("message:"):])
	} else {
		providerMsg = s
	}
	return status, providerMsg
}

// wrapModelError 把原始模型错误富化为 ModelCallError；err 为 nil 时返回 nil。
func wrapModelError(err error) error {
	if err == nil {
		return nil
	}
	// 已经富化过则不重复包装
	if _, ok := chatmodel.AsModelCallError(err); ok {
		return err
	}
	status, msg := extractModelHTTPError(err)
	return &chatmodel.ModelCallError{HTTPStatus: status, ProviderMessage: msg, Raw: err}
}
