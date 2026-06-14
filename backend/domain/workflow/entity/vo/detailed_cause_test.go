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
	"strings"
	"testing"

	workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
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

func TestCauseForMode_ReleaseRedactsProviderError(t *testing.T) {
	err := &chatmodel.ModelCallError{
		HTTPStatus:      400,
		ProviderMessage: "Incorrect API key sk-secret",
		Raw:             errors.New("raw"),
	}

	cause := CauseForMode(workflowModel.ExecuteModeRelease, err)
	if cause != SanitizedCauseMsg {
		t.Errorf("expected release cause to equal SanitizedCauseMsg %q, got %q", SanitizedCauseMsg, cause)
	}
	for _, leak := range []string{"Incorrect API key", "400", "sk-secret"} {
		if strings.Contains(cause, leak) {
			t.Errorf("release cause must not contain %q, got %q", leak, cause)
		}
	}
}

func TestCauseForMode_DebugExposesProviderError(t *testing.T) {
	err := &chatmodel.ModelCallError{
		HTTPStatus:      400,
		ProviderMessage: "Incorrect API key sk-secret",
		Raw:             errors.New("raw"),
	}

	for _, mode := range []workflowModel.ExecuteMode{
		workflowModel.ExecuteModeDebug,
		workflowModel.ExecuteModeNodeDebug,
	} {
		cause := CauseForMode(mode, err)
		if !strings.Contains(cause, "400") {
			t.Errorf("mode %v: expected cause to contain HTTP status %q, got %q", mode, "400", cause)
		}
		if !strings.Contains(cause, "Incorrect API key") {
			t.Errorf("mode %v: expected cause to contain provider message, got %q", mode, cause)
		}
	}
}
