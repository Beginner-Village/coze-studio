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

package vo

import (
	"errors"
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

func TestDetailedCausePrefersModelError(t *testing.T) {
	mce := &chatmodel.ModelCallError{HTTPStatus: 400, ProviderMessage: "invalid max_tokens", Raw: errors.New("raw")}
	got := DetailedCause(mce)
	if got != "model call failed [HTTP 400]: invalid max_tokens" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestDetailedCauseFallsBackToRoot(t *testing.T) {
	err := errors.New("plain root error")
	if DetailedCause(err) != "plain root error" {
		t.Fatalf("unexpected: %q", DetailedCause(err))
	}
}
