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
	"testing"
	"time"
)

func TestModelHTTPClientForDashScopeForcesIPv4(t *testing.T) {
	client := modelHTTPClient("https://dashscope.aliyuncs.com/compatible-mode/v1", 3*time.Second)
	if client == nil {
		t.Fatal("expected dashscope base_url to receive a custom HTTP client")
	}
	if client.Timeout != 3*time.Second {
		t.Fatalf("expected timeout to be preserved, got %s", client.Timeout)
	}
	if client.Transport == nil {
		t.Fatal("expected custom HTTP client to have a transport")
	}
}

func TestModelHTTPClientLeavesOtherHostsDefault(t *testing.T) {
	if client := modelHTTPClient("https://api.openai.com/v1", 3*time.Second); client != nil {
		t.Fatalf("expected non-dashscope base_url to use provider default HTTP client, got %#v", client)
	}
}

func TestShouldForceIPv4RejectsInvalidBaseURL(t *testing.T) {
	if shouldForceIPv4("://bad-url") {
		t.Fatal("expected invalid base_url not to force IPv4")
	}
}
