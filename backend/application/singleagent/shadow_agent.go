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

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/bot_common"
	crossdomainSingleagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	aiproduct "github.com/ynet-dev/ynet-studio/backend/application/aiproduct"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
)

// GetDraft reads the super-agent draft identified by agentID and returns the
// minimal view needed by the publish flow.  It satisfies aiproduct.AgentReader.
func (s *SingleAgentApplicationService) GetDraft(ctx context.Context, agentID int64) (*aiproduct.AgentDraftView, error) {
	draft, err := s.DomainSVC.GetSingleAgentDraft(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("GetDraftView agentID=%d: %w", agentID, err)
	}
	if draft == nil {
		return nil, fmt.Errorf("GetDraftView agentID=%d: not found", agentID)
	}

	view := &aiproduct.AgentDraftView{}

	// Model
	if mi := draft.ModelInfo; mi != nil && mi.ModelId != nil {
		view.ModelID = strconv.FormatInt(*mi.ModelId, 10)
		view.ModelParams = buildModelParams(mi)
	}

	// Prompt
	if p := draft.Prompt; p != nil && p.Prompt != nil {
		view.Prompt = *p.Prompt
	}

	// Skills
	for _, ref := range draft.SkillInfoList {
		view.Skills = append(view.Skills, aiproduct.SnapshotSkill{
			SkillID: ref.SkillID,
			Name:    ref.SkillName,
		})
	}

	// Capabilities (super-agent tool config as map)
	if cfg := draft.SuperAgentToolConfig; cfg != nil {
		view.Capabilities = superAgentToolConfigToMap(cfg)
	}

	// MCP servers
	if cfg := draft.SuperAgentToolConfig; cfg != nil {
		for _, srv := range cfg.MCPServers {
			view.MCPServers = append(view.MCPServers, mcpServerToMap(srv))
		}
	}

	return view, nil
}

// CreateShadowDraft inserts a new single_agent_draft row of agent_type="super"
// that is pre-seeded from snapshot and linked to the given product/version.
// It satisfies aiproduct.ShadowAgentWriter.
func (s *SingleAgentApplicationService) CreateShadowDraft(
	ctx context.Context,
	spaceID, userID, productID int64,
	version string,
	snapshot map[string]any,
) (int64, error) {
	now := time.Now().UnixMilli()
	draft := &entity.SingleAgent{
		SingleAgent: &crossdomainSingleagent.SingleAgent{
			SpaceID:              spaceID,
			CreatorID:            userID,
			AgentType:            "super",
			SourceProductID:      productID,
			SourceProductVersion: version,
			Name:                 fmt.Sprintf("shadow-%d-%s", productID, version),
			Prompt:               &bot_common.PromptInfo{},
			OnboardingInfo:       &bot_common.OnboardingInfo{},
			Plugin:               []*bot_common.PluginInfo{},
			Workflow:             []*bot_common.WorkflowInfo{},
			SuggestReply:         &bot_common.SuggestReplyInfo{},
			JumpConfig:           &bot_common.JumpConfig{},
			Database:             []*bot_common.Database{},
			CreatedAt:            now,
			UpdatedAt:            now,
		},
	}

	// Apply model info from snapshot if present.
	if snapshot != nil {
		if m, ok := snapshot["model"].(map[string]any); ok {
			if idStr, ok := m["model_id"].(string); ok && idStr != "" {
				if idInt, err := strconv.ParseInt(idStr, 10, 64); err == nil {
					draft.ModelInfo = &bot_common.ModelInfo{}
					draft.ModelInfo.ModelId = &idInt
				}
			}
		}
		if prompt, ok := snapshot["prompt"].(string); ok && prompt != "" {
			draft.Prompt = &bot_common.PromptInfo{Prompt: strPtr(prompt)}
		}
	}

	agentID, err := s.DomainSVC.CreateSingleAgentDraft(ctx, userID, draft)
	if err != nil {
		return 0, fmt.Errorf("CreateShadowDraft productID=%d: %w", productID, err)
	}
	return agentID, nil
}

// buildModelParams converts a ModelInfo's numeric fields to a map[string]any.
func buildModelParams(mi *bot_common.ModelInfo) map[string]any {
	p := make(map[string]any)
	if mi.Temperature != nil {
		p["temperature"] = *mi.Temperature
	}
	if mi.MaxTokens != nil {
		p["max_tokens"] = *mi.MaxTokens
	}
	if mi.TopP != nil {
		p["top_p"] = *mi.TopP
	}
	if mi.TopK != nil {
		p["top_k"] = *mi.TopK
	}
	if mi.FrequencyPenalty != nil {
		p["frequency_penalty"] = *mi.FrequencyPenalty
	}
	if mi.PresencePenalty != nil {
		p["presence_penalty"] = *mi.PresencePenalty
	}
	return p
}

// superAgentToolConfigToMap serialises only the bool switches (nil → omitted).
func superAgentToolConfigToMap(cfg *crossdomainSingleagent.SuperAgentToolConfig) map[string]any {
	m := make(map[string]any)
	if cfg.Sandbox != nil {
		m["sandbox"] = *cfg.Sandbox
	}
	if cfg.WebSearch != nil {
		m["web_search"] = *cfg.WebSearch
	}
	if cfg.WebFetch != nil {
		m["web_fetch"] = *cfg.WebFetch
	}
	if cfg.RunBash != nil {
		m["run_bash"] = *cfg.RunBash
	}
	if cfg.DeepTask != nil {
		m["deep_task"] = *cfg.DeepTask
	}
	if cfg.SkillManage != nil {
		m["skill_manage"] = *cfg.SkillManage
	}
	return m
}

func mcpServerToMap(srv *crossdomainSingleagent.MCPServerConfig) map[string]any {
	if srv == nil {
		return nil
	}
	m := map[string]any{
		"name": srv.Name,
		"type": srv.Type,
	}
	if srv.Command != "" {
		m["command"] = srv.Command
	}
	if len(srv.Args) > 0 {
		m["args"] = srv.Args
	}
	if len(srv.Env) > 0 {
		m["env"] = srv.Env
	}
	if srv.URL != "" {
		m["url"] = srv.URL
	}
	if srv.TimeoutSec != 0 {
		m["timeout_sec"] = srv.TimeoutSec
	}
	if srv.Enabled != nil {
		m["enabled"] = *srv.Enabled
	}
	return m
}

func strPtr(s string) *string { return &s }
