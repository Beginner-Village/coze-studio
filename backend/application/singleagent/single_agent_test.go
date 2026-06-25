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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/bot_common"
	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/api/model/playground"
	agententity "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	agentservice "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/service"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
)

// fakeSingleAgentDomainForShadow implements agentservice.SingleAgent minimally.
type fakeSingleAgentDomainForShadow struct {
	agentservice.SingleAgent
	draft   *agententity.SingleAgent
	updated *agententity.SingleAgent
}

func (f *fakeSingleAgentDomainForShadow) GetSingleAgentDraft(_ context.Context, _ int64) (*agententity.SingleAgent, error) {
	return f.draft, nil
}

func (f *fakeSingleAgentDomainForShadow) UpdateSingleAgentDraft(_ context.Context, info *agententity.SingleAgent) error {
	f.updated = info
	return nil
}

// newTestSingleAgentSVC creates a minimal SingleAgentApplicationService for unit tests.
func newTestSingleAgentSVC(draft *agententity.SingleAgent) *SingleAgentApplicationService {
	return &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomainForShadow{draft: draft},
	}
}

// TestShadowUpdateRejectedWhenSourceProductIDNonZero verifies that
// UpdateSingleAgentDraft returns an error when the draft's SourceProductID != 0
// (i.e. the draft is a virtual-employee instance shadow — read-only config).
func TestShadowUpdateRejectedWhenSourceProductIDNonZero(t *testing.T) {
	draft := &agententity.SingleAgent{
		SingleAgent: &crossagent.SingleAgent{
			AgentID:         100,
			CreatorID:       1,
			SpaceID:         1,
			SourceProductID: 42, // non-zero → shadow instance
			AgentType:       "super",
		},
	}
	svc := newTestSingleAgentSVC(draft)
	ctx := context.Background()

	req := &playground.UpdateDraftBotInfoAgwRequest{
		BotInfo: &bot_common.BotInfoForUpdate{
			BotId: ptr.Of(int64(100)),
		},
	}

	_, err := svc.UpdateSingleAgentDraft(ctx, req)
	assert.Error(t, err, "UpdateSingleAgentDraft must return error for shadow instance (SourceProductID!=0)")
}

// TestShadowUpdateAllowedWhenSourceProductIDZero verifies that a normal agent
// (SourceProductID == 0) is NOT rejected by the shadow guard (it may fail further
// down for unrelated reasons like missing session, but must not fail with the
// shadow-reject error).
func TestShadowUpdateAllowedWhenSourceProductIDZero(t *testing.T) {
	draft := &agententity.SingleAgent{
		SingleAgent: &crossagent.SingleAgent{
			AgentID:         101,
			CreatorID:       1,
			SpaceID:         1,
			SourceProductID: 0, // normal agent
			Name:            "normal",
		},
	}
	svc := newTestSingleAgentSVC(draft)
	ctx := context.Background()

	req := &playground.UpdateDraftBotInfoAgwRequest{
		BotInfo: &bot_common.BotInfoForUpdate{
			BotId: ptr.Of(int64(101)),
		},
	}

	// The normal agent proceeds past the shadow guard and reaches MustGetUIDFromCtx
	// which panics in test (no session). We catch that panic here to prove the
	// guard itself did NOT fire.
	var shadowErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Panic from MustGetUIDFromCtx — expected in test env, not the shadow guard.
				// Verify it is NOT a shadow-guard error.
				msg := fmt.Sprintf("%v", r)
				if strings.Contains(msg, "read-only shadow") || strings.Contains(msg, "SourceProductID") {
					shadowErr = fmt.Errorf("unexpected shadow rejection: %v", r)
				}
			}
		}()
		_, shadowErr = svc.UpdateSingleAgentDraft(ctx, req)
	}()
	if shadowErr != nil {
		t.Fatalf("normal agent must not be shadow-rejected: %v", shadowErr)
	}
}
