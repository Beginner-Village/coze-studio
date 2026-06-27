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

package coze_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"

	"github.com/ynet-dev/ynet-studio/backend/api/handler/coze"
	aiproduct "github.com/ynet-dev/ynet-studio/backend/application/aiproduct"
)

// TestPublishAgentApp_NilSVC verifies that SuperAgentPublishAgentApp returns 500
// (not a panic) when AgentAppSVC is nil (sandbox disabled).
func TestPublishAgentApp_NilSVC(t *testing.T) {
	// Ensure SVC is nil for this test.
	orig := aiproduct.AgentAppSVC
	aiproduct.AgentAppSVC = nil
	defer func() { aiproduct.AgentAppSVC = orig }()

	h := server.Default()
	h.POST("/api/super-agent/agent-app/publish", coze.SuperAgentPublishAgentApp)

	body, _ := json.Marshal(map[string]any{
		"agent_id": "123",
		"space_id": "456",
		"user_id":  "789",
		"name":     "My Agent",
		"version":  "v1.0.0",
	})
	w := ut.PerformRequest(h.Engine, "POST", "/api/super-agent/agent-app/publish",
		&ut.Body{Body: bytes.NewBuffer(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestPublishAgentApp_MissingUserID verifies that SuperAgentPublishAgentApp
// returns 400 when caller cannot be resolved (no user_id, no session context).
func TestPublishAgentApp_MissingUserID(t *testing.T) {
	orig := aiproduct.AgentAppSVC
	aiproduct.AgentAppSVC = nil
	defer func() { aiproduct.AgentAppSVC = orig }()

	h := server.Default()
	h.POST("/api/super-agent/agent-app/publish", coze.SuperAgentPublishAgentApp)

	body, _ := json.Marshal(map[string]any{
		"agent_id": "123",
		"space_id": "456",
		// no user_id
		"name":    "My Agent",
		"version": "v1.0.0",
	})
	w := ut.PerformRequest(h.Engine, "POST", "/api/super-agent/agent-app/publish",
		&ut.Body{Body: bytes.NewBuffer(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	// user_id missing → 400 bad request
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestBuildStatusAgentApp_NilSVC verifies that SuperAgentAgentAppBuildStatus
// returns 500 (not a panic) when AgentAppSVC is nil.
func TestBuildStatusAgentApp_NilSVC(t *testing.T) {
	orig := aiproduct.AgentAppSVC
	aiproduct.AgentAppSVC = nil
	defer func() { aiproduct.AgentAppSVC = orig }()

	h := server.Default()
	h.POST("/api/super-agent/agent-app/build-status", coze.SuperAgentAgentAppBuildStatus)

	body, _ := json.Marshal(map[string]any{
		"product_id": "123",
		"version":    "v1.0.0",
		"user_id":    "789",
	})
	w := ut.PerformRequest(h.Engine, "POST", "/api/super-agent/agent-app/build-status",
		&ut.Body{Body: bytes.NewBuffer(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestRecruitAgentApp_NilSVC verifies that SuperAgentRecruitAgentApp
// returns 500 (not a panic) when AgentAppSVC is nil.
func TestRecruitAgentApp_NilSVC(t *testing.T) {
	orig := aiproduct.AgentAppSVC
	aiproduct.AgentAppSVC = nil
	defer func() { aiproduct.AgentAppSVC = orig }()

	h := server.Default()
	h.POST("/api/super-agent/agent-app/recruit", coze.SuperAgentRecruitAgentApp)

	body, _ := json.Marshal(map[string]any{
		"product_id": "123",
		"space_id":   "456",
		"user_id":    "789",
	})
	w := ut.PerformRequest(h.Engine, "POST", "/api/super-agent/agent-app/recruit",
		&ut.Body{Body: bytes.NewBuffer(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
