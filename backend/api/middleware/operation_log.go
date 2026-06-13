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

		ev := buildEvent(rule, method, path, *uidPtr,
			query, bodyJSON, pathSegs, string(bodyBytes),
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
	rawBody string, statusCode int, clientIP string, durationMs int32,
	logID string, nowMs int64) *entity.Event {

	spaceID := oplogmw.ExtractSpaceID(query, nil, bodyJSON)
	if spaceID == 0 {
		spaceID = oplogmw.ParseInt64Public(pathSegs["space_id"])
	}

	resourceIDStr := oplogmw.ExtractBySpec(rule.ResourceIDFrom, query, nil, bodyJSON, pathSegs)
	resourceName := oplogmw.ExtractBySpec(rule.ResourceNameFrom, query, nil, bodyJSON, pathSegs)

	status := entity.StatusSuccess
	if statusCode >= 400 {
		status = entity.StatusFail
	}

	summary := rawBody
	if len(summary) > opLogMaxSummary {
		summary = summary[:opLogMaxSummary]
	}

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
		ClientIP:       clientIP,
		DurationMs:     durationMs,
		LogID:          logID,
		CreatedAt:      nowMs,
	}
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
