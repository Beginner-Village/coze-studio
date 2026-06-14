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
	"errors"
	"testing"
)

func TestExtractModelHTTPError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantMsgSub string
	}{
		{"openai-style", errors.New("error, status code: 400, message: invalid 'max_tokens'"), 400, "invalid 'max_tokens'"},
		{"status-only", errors.New("request failed with status 429 Too Many Requests"), 429, "Too Many Requests"},
		{"no-status", errors.New("connection refused"), 0, "connection refused"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, msg := extractModelHTTPError(c.err)
			if status != c.wantStatus {
				t.Fatalf("status: want %d got %d", c.wantStatus, status)
			}
			if c.wantMsgSub != "" && !contains(msg, c.wantMsgSub) {
				t.Fatalf("msg %q does not contain %q", msg, c.wantMsgSub)
			}
		})
	}
}

func contains(s, sub string) bool { return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0) }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
