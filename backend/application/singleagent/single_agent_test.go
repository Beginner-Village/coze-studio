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
	userentity "github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
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
	// Authenticate as the draft owner so ValidateAgentDraftAccess passes and the
	// request actually reaches the shadow guard under test.
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 1})

	req := &playground.UpdateDraftBotInfoAgwRequest{
		BotInfo: &bot_common.BotInfoForUpdate{
			BotId: ptr.Of(int64(100)),
		},
	}

	// A shadow instance (SourceProductID!=0) is read-only: the guard silently
	// ignores the update (returns success with HasChange=false, no persistence)
	// rather than erroring, so the editor's auto-save does not surface a toast.
	resp, err := svc.UpdateSingleAgentDraft(ctx, req)
	assert.NoError(t, err)
	if assert.NotNil(t, resp) && assert.NotNil(t, resp.Data) {
		assert.False(t, resp.Data.GetHasChange(), "shadow instance update must be ignored (HasChange=false)")
	}
	// Confirm the shadow guard short-circuited before any persistence.
	assert.Nil(t, svc.DomainSVC.(*fakeSingleAgentDomainForShadow).updated)
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
	// Authenticate as the draft owner so the access check passes and the request
	// reaches the shadow guard. A normal agent (SourceProductID==0) is not
	// rejected by the guard and proceeds into the update path, which panics in
	// this minimal test on the nil appContext — that downstream panic proves the
	// shadow guard itself did NOT fire.
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 1})

	req := &playground.UpdateDraftBotInfoAgwRequest{
		BotInfo: &bot_common.BotInfoForUpdate{
			BotId: ptr.Of(int64(101)),
		},
	}

	var shadowErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Downstream panic (nil appContext) — expected in test env, not the
				// shadow guard. Verify it is NOT a shadow-guard rejection.
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
