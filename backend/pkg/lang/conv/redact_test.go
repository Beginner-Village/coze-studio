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

package conv

import (
	"strings"
	"testing"
)

func TestDebugJsonToStrRedactsBase64DataURL(t *testing.T) {
	payload := strings.Repeat("A", 300)
	v := map[string]any{
		"role": "user",
		"url":  "data:image/png;base64," + payload,
	}

	out := DebugJsonToStr(v)

	if strings.Contains(out, payload) {
		t.Fatalf("base64 payload leaked through DebugJsonToStr: %s", out)
	}
	if !strings.Contains(out, "data:image/png;base64,[redacted b64len=300") {
		t.Fatalf("expected redaction marker with mime + length, got: %s", out)
	}
	if !strings.Contains(out, "sha256=") {
		t.Fatalf("expected sha256 metadata, got: %s", out)
	}
	// 非 base64 字段保持原样
	if !strings.Contains(out, `"role":"user"`) {
		t.Fatalf("non-image field was altered: %s", out)
	}
}

func TestRedactBase64DataURLsLeavesPlainTextUnchanged(t *testing.T) {
	in := `{"content":"请直接读这张图,只回答:title=..., code=..."}`
	if got := redactBase64DataURLs(in); got != in {
		t.Fatalf("plain text without data URL must be unchanged, got: %s", got)
	}
}

func TestRedactBase64DataURLsHandlesMultiple(t *testing.T) {
	p1 := strings.Repeat("A", 200)
	p2 := strings.Repeat("B", 120) + "=="
	in := "a data:image/jpeg;base64," + p1 + " b data:application/pdf;base64," + p2 + " c"

	out := redactBase64DataURLs(in)

	if strings.Contains(out, p1) || strings.Contains(out, p2) {
		t.Fatalf("a base64 payload leaked: %s", out)
	}
	if strings.Count(out, "[redacted b64len=") != 2 {
		t.Fatalf("expected exactly 2 redactions, got: %s", out)
	}
	if !strings.HasPrefix(out, "a ") || !strings.HasSuffix(out, " c") {
		t.Fatalf("surrounding text altered: %s", out)
	}
}
