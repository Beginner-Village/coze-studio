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

package observability

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

// HTTPRequestsMiddleware records RED metrics (rate, errors, duration)
// for every Hertz request. Path is normalized to keep label cardinality
// bounded (e.g. /users/123 -> /users/:id).
func HTTPRequestsMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		HTTPRequestsInFlight.Inc()
		defer HTTPRequestsInFlight.Dec()

		start := time.Now()
		c.Next(ctx)
		dur := time.Since(start).Seconds()

		method := string(c.Request.Method())
		path := normalizePath(string(c.Request.URI().Path()))
		status := strconv.Itoa(c.Response.StatusCode())

		HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		HTTPRequestDuration.WithLabelValues(method, path, status).Observe(dur)
	}
}

// normalizePath strips numeric/UUID segments to keep label cardinality bounded.
func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	parts := strings.Split(p, "/")
	for i, seg := range parts {
		if seg == "" {
			continue
		}
		if _, err := strconv.ParseInt(seg, 10, 64); err == nil {
			parts[i] = ":id"
			continue
		}
		// UUID length: 36 chars, 4 dashes (e.g. 00000000-0000-4000-8000-000000000000)
		if len(seg) == 36 && strings.Count(seg, "-") == 4 {
			parts[i] = ":uuid"
			continue
		}
	}
	return strings.Join(parts, "/")
}
