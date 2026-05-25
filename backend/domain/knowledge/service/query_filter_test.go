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

package service

import (
	"testing"
)

func TestIsJunkQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{"empty", "", true},
		{"single char ascii", "a", true},
		{"single char chinese", "好", true},
		{"two char ascii", "ab", true},      // 长度 < MinQueryLen=3
		{"two char chinese", "你好", true},   // 长度 < MinQueryLen=3
		{"repeat ascii", "aaa", true},
		{"repeat ascii longer", "aaaa", true},
		{"repeat chinese", "啊啊啊", true},
		{"all punct", "!!!", true},
		{"all punct mixed", "?!@", true},
		{"all whitespace", "   ", true},
		{"valid short", "abc", false},        // 3 字符且非重复 → 通过
		{"valid chinese", "知识库", false},
		{"valid mixed", "API key", false},
		{"valid english phrase", "vector embedding", false},
		{"valid leading whitespace", "  hello", false}, // trim 后非 junk
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJunkQuery(tt.query)
			if got != tt.want {
				t.Errorf("isJunkQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}
