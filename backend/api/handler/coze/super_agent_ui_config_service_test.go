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

import "testing"

func TestSuperAgentUIEnabled(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{"", true},          // 未设置 → 默认启用（与沙箱默认开一致）
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"false", false},
		{"FALSE", false},
		{" false ", false},  // 含空白也应识别
		{"0", false},
		{"no", false},
		{"off", false},
		{"disabled", false},
		{"anything", true},  // 无法识别为「关」的一律视为开
	}
	for _, tc := range cases {
		t.Setenv("SANDBOX_ENABLED", tc.val)
		if got := superAgentUIEnabled(); got != tc.want {
			t.Errorf("SANDBOX_ENABLED=%q: got %v, want %v", tc.val, got, tc.want)
		}
	}
}
