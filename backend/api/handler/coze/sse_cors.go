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
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
)

var sseAllowedOrigins = sync.OnceValue(func() map[string]struct{} {
	set := make(map[string]struct{})
	for _, o := range strings.Split(os.Getenv("YNET_LOOP_ALLOWED_ORIGINS"), ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			set[o] = struct{}{}
		}
	}
	return set
})

// setSSECORSHeaders 仅当请求 Origin 在白名单（YNET_LOOP_ALLOWED_ORIGINS）内或与请求同源时，
// 回显该 Origin 并加 Vary: Origin；否则不设置任何 CORS 头。
func setSSECORSHeaders(c *app.RequestContext) {
	origin := string(c.Request.Header.Peek("Origin"))
	if origin == "" {
		return
	}
	allowed := false
	if _, ok := sseAllowedOrigins()[origin]; ok {
		allowed = true
	} else if u, err := url.Parse(origin); err == nil && u.Host != "" && u.Host == string(c.Host()) {
		allowed = true
	}
	if !allowed {
		return
	}
	c.Response.Header.Set("Access-Control-Allow-Origin", origin)
	c.Response.Header.Add("Vary", "Origin")
}
