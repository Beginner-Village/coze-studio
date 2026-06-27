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

package conv

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
)

// base64DataURLPattern 匹配 data:<mime>;base64,<payload> 形式的内嵌资源。
// payload 至少 24 个 base64 字符才视为需要脱敏的大内容（图片/文件），
// 以免误伤短小的合法取值。同时覆盖标准(+/)与 URL-safe(-_)字母表。
var base64DataURLPattern = regexp.MustCompile(`data:([a-zA-Z0-9.+-]+/[a-zA-Z0-9.+-]+)?;base64,([A-Za-z0-9+/=_-]{24,})`)

// redactBase64DataURLs 把字符串里的 base64 data URL 载荷替换为只含 MIME、
// 字节长度与短 hash 的占位符。多模态输入会把内网图片转成 data URL 直接喂给
// 模型，调试日志（DebugJsonToStr）若原样打印请求会泄露整张图片的 base64，
// 这里统一脱敏，只保留可排查所需的元信息。
func redactBase64DataURLs(s string) string {
	return base64DataURLPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := base64DataURLPattern.FindStringSubmatch(match)
		mime := sub[1]
		if mime == "" {
			mime = "application/octet-stream"
		}
		payload := sub[2]
		sum := sha256.Sum256([]byte(payload))
		return fmt.Sprintf("data:%s;base64,[redacted b64len=%d sha256=%s]",
			mime, len(payload), hex.EncodeToString(sum[:])[:12])
	})
}
