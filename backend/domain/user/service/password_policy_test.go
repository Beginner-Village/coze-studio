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

func TestValidatePasswordStrength(t *testing.T) {
	cases := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid strong", "Abcdef1!", false},
		{"valid complex", "Str0ng#Pass2026", false},
		{"too short", "Ab1!xy", true},
		{"no upper", "abcdef1!", true},
		{"no lower", "ABCDEF1!", true},
		{"no digit", "Abcdefg!", true},
		{"no special", "Abcdefg1", true},
		{"empty", "", true},
		{"weak dictionary", "Password1!", true},
		{"weak admin", "Admin@123", true},
		{"too long", "Aa1!" + string(make([]byte, 80)), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validatePasswordStrength(c.password)
			if (err != nil) != c.wantErr {
				t.Fatalf("validatePasswordStrength(%q) err=%v, wantErr=%v", c.password, err, c.wantErr)
			}
		})
	}
}
