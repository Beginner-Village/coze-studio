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

import "testing"

func TestEffectiveMinScore(t *testing.T) {
	tests := []struct {
		name     string
		strategy float64
		want     float64
	}{
		{"low strategy below floor", 0.01, 0.3},
		{"medium strategy below floor", 0.2, 0.3},
		{"strategy at floor", 0.3, 0.3},
		{"strategy above floor", 0.5, 0.5},
		{"strategy very high", 0.9, 0.9},
		{"strategy zero", 0.0, 0.3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveMinScore(tt.strategy)
			if got != tt.want {
				t.Errorf("effectiveMinScore(%v) = %v, want %v", tt.strategy, got, tt.want)
			}
		})
	}
}
