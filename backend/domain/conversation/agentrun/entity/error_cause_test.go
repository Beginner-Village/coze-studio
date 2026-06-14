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

package entity

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

func TestCauseForDebug_NilError(t *testing.T) {
	if got := CauseForDebug(true, nil); got != "" {
		t.Fatalf("nil error should yield empty cause, got %q", got)
	}
	if got := CauseForDebug(false, nil); got != "" {
		t.Fatalf("nil error should yield empty cause, got %q", got)
	}
}

func TestCauseForDebug_DebugExposesModelError(t *testing.T) {
	mce := &chatmodel.ModelCallError{
		HTTPStatus:      400,
		ProviderMessage: "Incorrect API key sk-secret",
		Raw:             errors.New("raw"),
	}

	cause := CauseForDebug(true, mce)
	if !strings.Contains(cause, "400") {
		t.Errorf("debug cause should contain HTTP status, got %q", cause)
	}
	if !strings.Contains(cause, "Incorrect API key") {
		t.Errorf("debug cause should contain provider message, got %q", cause)
	}
}

func TestCauseForDebug_DebugExposesWrappedModelError(t *testing.T) {
	mce := &chatmodel.ModelCallError{HTTPStatus: 429, ProviderMessage: "rate limited", Raw: errors.New("raw")}
	wrapped := fmt.Errorf("agent stream execute: %w", mce)

	cause := CauseForDebug(true, wrapped)
	if !strings.Contains(cause, "429") || !strings.Contains(cause, "rate limited") {
		t.Errorf("debug cause should surface wrapped model error detail, got %q", cause)
	}
}

func TestCauseForDebug_DebugFallsBackToErrorText(t *testing.T) {
	err := errors.New("plain agent error")
	if got := CauseForDebug(true, err); got != "plain agent error" {
		t.Errorf("debug cause should fall back to error text, got %q", got)
	}
}

func TestCauseForDebug_ReleaseRedacts(t *testing.T) {
	cases := []error{
		&chatmodel.ModelCallError{HTTPStatus: 400, ProviderMessage: "Incorrect API key sk-secret", Raw: errors.New("raw")},
		errors.New("some internal stack trace with sk-secret token"),
	}

	for _, err := range cases {
		cause := CauseForDebug(false, err)
		if cause != SanitizedCauseMsg {
			t.Errorf("release cause should equal SanitizedCauseMsg %q, got %q", SanitizedCauseMsg, cause)
		}
		for _, leak := range []string{"sk-secret", "Incorrect API key", "400", "stack trace"} {
			if strings.Contains(cause, leak) {
				t.Errorf("release cause must not leak %q, got %q", leak, cause)
			}
		}
	}
}
