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

package middleware

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	oplogmw "github.com/ynet-dev/ynet-studio/backend/api/middleware/operationlog"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/application/operationlog"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
)

const opLogMaxSummary = 512

// OperationLogMW collects write operations that hit the route whitelist and
// asynchronously persists them. It must be registered after SessionAuthMW so
// that the user id is present in the context.
func OperationLogMW() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		start := time.Now()
		method := string(ctx.Request.Header.Method())
		path := stripQuery(string(ctx.Request.URI().PathOriginal()))

		rule := oplogmw.Match(method, path)
		if rule == nil {
			ctx.Next(c)
			return
		}

		// Capture the request body before Next. Hertz buffers the body so this
		// read does not consume it for the downstream handler.
		bodyBytes := append([]byte(nil), ctx.Request.Body()...)

		ctx.Next(c)

		uidPtr := ctxutil.GetUIDFromCtx(c)
		if uidPtr == nil {
			return // unauthenticated write operations are not audited
		}

		query := map[string]string{}
		ctx.QueryArgs().VisitAll(func(k, v []byte) { query[string(k)] = string(v) })

		bodyJSON := map[string]any{}
		if len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &bodyJSON)
		}

		pathSegs := oplogmw.PathSegs(rule.PathPattern, path)

		// Copy the response body so the audit event is not affected by buffer
		// reuse after the handler chain returns.
		respBody := append([]byte(nil), ctx.Response.Body()...)

		ev := buildEvent(rule, method, path, *uidPtr,
			query, bodyJSON, pathSegs, string(bodyBytes), respBody,
			ctx.Response.StatusCode(), ctx.ClientIP(),
			int32(time.Since(start).Milliseconds()), getLogID(c),
			time.Now().UnixMilli())

		operationlog.OperationLogApplicationSVC.Collect(ev)
	}
}

// buildEvent assembles an audit event from already-parsed request data. It is a
// pure function so the assembly logic can be unit-tested without Hertz.
func buildEvent(rule *oplogmw.RouteRule, method, path string, uid int64,
	query map[string]string, bodyJSON map[string]any, pathSegs map[string]string,
	rawBody string, respBody []byte, statusCode int, clientIP string, durationMs int32,
	logID string, nowMs int64) *entity.Event {

	spaceID := oplogmw.ExtractSpaceID(query, nil, bodyJSON)
	if spaceID == 0 {
		spaceID = oplogmw.ParseInt64Public(pathSegs["space_id"])
	}

	resourceIDStr := oplogmw.ExtractBySpec(rule.ResourceIDFrom, query, nil, bodyJSON, pathSegs)
	resourceName := oplogmw.ExtractBySpec(rule.ResourceNameFrom, query, nil, bodyJSON, pathSegs)

	// Determine success/fail. Most APIs return HTTP 200 with a business code in
	// the response body: a non-zero top-level "code" means failure even though
	// the HTTP status is 2xx.
	status := entity.StatusSuccess
	errorCode := ""
	if statusCode >= 400 {
		status = entity.StatusFail
		errorCode = strconv.Itoa(statusCode)
	} else if code, ok := extractBizCode(respBody); ok && code != 0 {
		status = entity.StatusFail
		errorCode = strconv.FormatInt(code, 10)
	}

	summary := redactSummary([]byte(rawBody))

	return &entity.Event{
		SpaceID:        spaceID,
		OperatorID:     uid,
		Module:         rule.Module,
		ResourceType:   rule.ResourceType,
		ResourceID:     oplogmw.ParseInt64Public(resourceIDStr),
		ResourceName:   resourceName,
		Action:         rule.Action,
		Description:    rule.DescTemplate,
		Method:         method,
		Path:           path,
		RequestSummary: summary,
		Status:         status,
		ErrorCode:      errorCode,
		ClientIP:       clientIP,
		DurationMs:     durationMs,
		LogID:          logID,
		CreatedAt:      nowMs,
	}
}

// extractBizCode parses respBody as a JSON object and reads the top-level "code"
// field as an int64. Business error codes are large (9+ digits), so int64 is
// required. The value may be a JSON number or a numeric string. Returns ok=false
// when the body is not a JSON object or has no usable numeric "code" field; the
// caller treats that as success so audit parsing never affects the main flow.
func extractBizCode(respBody []byte) (code int64, ok bool) {
	if len(respBody) == 0 {
		return 0, false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(respBody, &obj); err != nil {
		return 0, false
	}
	raw, exists := obj["code"]
	if !exists {
		return 0, false
	}
	// Try JSON number first.
	var num json.Number
	if err := json.Unmarshal(raw, &num); err == nil {
		if i, err := num.Int64(); err == nil {
			return i, true
		}
	}
	// Try numeric string, e.g. "code":"700012345".
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if i, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
			return i, true
		}
	}
	return 0, false
}

// sensitiveKeySubstrings 是顶层 JSON key 名中若包含(不区分大小写)即需脱敏的子串。
var sensitiveKeySubstrings = []string{
	"password", "passwd", "secret", "token", "api_key", "apikey",
	"access_key", "private_key", "credential", "authorization",
}

// redactSummary 若 body 是 JSON 对象，则对顶层 key 名含敏感子串(不区分大小写)
// 的值替换为 "***"，再序列化并截断到 opLogMaxSummary；否则退回原始截断逻辑。
func redactSummary(bodyBytes []byte) string {
	var obj map[string]any
	if err := json.Unmarshal(bodyBytes, &obj); err != nil {
		return truncate(string(bodyBytes))
	}
	for k := range obj {
		lower := strings.ToLower(k)
		for _, s := range sensitiveKeySubstrings {
			if strings.Contains(lower, s) {
				obj[k] = "***"
				break
			}
		}
	}
	redacted, err := json.Marshal(obj)
	if err != nil {
		return truncate(string(bodyBytes))
	}
	return truncate(string(redacted))
}

func truncate(s string) string {
	if len(s) > opLogMaxSummary {
		return s[:opLogMaxSummary]
	}
	return s
}

func stripQuery(p string) string {
	if i := strings.IndexByte(p, '?'); i >= 0 {
		return p[:i]
	}
	return p
}

// getLogID reads the request log id set by SetLogIDMW (api/middleware/log.go),
// which stores it on the context under the "log-id" key.
func getLogID(c context.Context) string {
	if v := c.Value("log-id"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
