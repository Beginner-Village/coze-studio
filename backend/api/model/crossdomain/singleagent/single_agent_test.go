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

package singleagent

import "testing"

func TestSkillExecutionEnabled(t *testing.T) {
	var nilCfg *SuperAgentToolConfig
	if !nilCfg.SkillExecutionEnabled() {
		t.Fatal("nil config -> skill execution enabled by default (backward compatible)")
	}
	if !(&SuperAgentToolConfig{}).SkillExecutionEnabled() {
		t.Fatal("unset field -> enabled by default")
	}
	off := false
	if (&SuperAgentToolConfig{SkillExecution: &off}).SkillExecutionEnabled() {
		t.Fatal("SkillExecution=false -> disabled")
	}
	on := true
	if !(&SuperAgentToolConfig{SkillExecution: &on}).SkillExecutionEnabled() {
		t.Fatal("SkillExecution=true -> enabled")
	}
}

func TestRedactedForReadMasksMCPEnv(t *testing.T) {
	enabled := true
	cfg := &SuperAgentToolConfig{
		Sandbox: &enabled,
		MCPServers: []*MCPServerConfig{
			{
				Name: "srv1",
				Type: "streamable_http",
				URL:  "https://mcp.example.com",
				Env:  map[string]string{"Authorization": "Bearer super-secret", "X-Api-Key": "k123"},
			},
		},
	}
	red := cfg.RedactedForRead()

	// The original config must not be mutated.
	if cfg.MCPServers[0].Env["Authorization"] != "Bearer super-secret" {
		t.Fatalf("RedactedForRead must not mutate the original: %v", cfg.MCPServers[0].Env)
	}
	// Redacted copy: keys preserved (so the UI can show which vars exist) but no
	// real secret leaked.
	renv := red.MCPServers[0].Env
	if _, ok := renv["Authorization"]; !ok {
		t.Fatal("redacted env must keep the key so the UI can show it")
	}
	if renv["Authorization"] == "Bearer super-secret" || renv["X-Api-Key"] == "k123" {
		t.Fatalf("redacted env must NOT contain the real secret: %v", renv)
	}
	// Non-secret fields copied verbatim.
	if red.MCPServers[0].URL != "https://mcp.example.com" || red.MCPServers[0].Name != "srv1" {
		t.Fatal("non-secret fields must be preserved in the redacted copy")
	}
	if red.Sandbox == nil || *red.Sandbox != true {
		t.Fatal("bool switches must be preserved in the redacted copy")
	}
}

func TestRedactedForReadNil(t *testing.T) {
	var cfg *SuperAgentToolConfig
	if cfg.RedactedForRead() != nil {
		t.Fatal("nil config redacts to nil")
	}
}

func TestMergeMCPServerSecretsRestoresAndUpdates(t *testing.T) {
	old := &SuperAgentToolConfig{
		MCPServers: []*MCPServerConfig{
			{Name: "srv1", Env: map[string]string{"Authorization": "Bearer OLD", "Keep": "oldkeep"}},
		},
	}
	// Client sends back the redaction sentinel for Authorization (untouched) but a
	// real new value for Keep (changed by the user).
	incoming := &SuperAgentToolConfig{
		MCPServers: []*MCPServerConfig{
			{Name: "srv1", Env: map[string]string{"Authorization": mcpSecretRedacted, "Keep": "newkeep"}},
		},
	}
	MergeMCPServerSecrets(incoming, old)
	got := incoming.MCPServers[0].Env
	if got["Authorization"] != "Bearer OLD" {
		t.Fatalf("redacted secret must be restored from old config, got %q", got["Authorization"])
	}
	if got["Keep"] != "newkeep" {
		t.Fatalf("a real new value must be kept, got %q", got["Keep"])
	}
}

func TestMergeMCPServerSecretsDropsSentinelWithoutCounterpart(t *testing.T) {
	// A sentinel value with no stored counterpart (e.g. a brand-new server) must
	// be dropped, never persisted literally.
	incoming := &SuperAgentToolConfig{
		MCPServers: []*MCPServerConfig{
			{Name: "brand-new", Env: map[string]string{"Authorization": mcpSecretRedacted}},
		},
	}
	MergeMCPServerSecrets(incoming, nil)
	if _, ok := incoming.MCPServers[0].Env["Authorization"]; ok {
		t.Fatalf("sentinel with no old counterpart must be dropped, got %v", incoming.MCPServers[0].Env)
	}
}
