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

package conversation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	openauthentity "github.com/ynet-dev/ynet-studio/backend/domain/openauth/openapiauth/entity"
	userentity "github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

func TestResolveOpenapiRunActorUsesApiAuth(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{
		UserID:      77,
		ConnectorID: 1234,
	})

	actor, err := resolveOpenapiRunActor(ctx, nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, int64(77), actor.creatorID)
	assert.Equal(t, int64(1234), actor.connectorID)
	assert.False(t, actor.isDraft)
}

func TestResolveOpenapiRunActorUsesDraftForSuperAgentAppServerApiAuth(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{
		UserID:      77,
		ConnectorID: consts.APIConnectorID,
	})

	actor, err := resolveOpenapiRunActor(ctx, nil, map[string]string{
		"super_agent_app_server": "true",
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(77), actor.creatorID)
	assert.Equal(t, consts.CozeConnectorID, actor.connectorID)
	assert.True(t, actor.isDraft)
}

func TestResolveOpenapiRunActorIgnoresSuperAgentFlagForNonAPIConnector(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{
		UserID:      77,
		ConnectorID: 1234,
	})

	actor, err := resolveOpenapiRunActor(ctx, nil, map[string]string{
		"super_agent_app_server": "true",
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(77), actor.creatorID)
	assert.Equal(t, int64(1234), actor.connectorID)
	assert.False(t, actor.isDraft)
}

func TestResolveOpenapiRunActorUsesSessionWhenApiAuthMissing(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: 42})

	actor, err := resolveOpenapiRunActor(ctx, nil, nil)

	assert.NoError(t, err)
	assert.Equal(t, int64(42), actor.creatorID)
	assert.Equal(t, consts.CozeConnectorID, actor.connectorID)
	assert.True(t, actor.isDraft)
}

func TestResolveOpenapiRunActorAppliesWebSDKConnectorOverride(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{
		UserID:      77,
		ConnectorID: 1234,
	})
	connectorID := consts.WebSDKConnectorID

	actor, err := resolveOpenapiRunActor(ctx, &connectorID, map[string]string{
		"super_agent_app_server": "true",
	})

	assert.NoError(t, err)
	assert.Equal(t, int64(77), actor.creatorID)
	assert.Equal(t, consts.WebSDKConnectorID, actor.connectorID)
	assert.False(t, actor.isDraft)
}

func TestResolveOpenapiRunActorRequiresAuthOrSession(t *testing.T) {
	ctx := ctxcache.Init(context.Background())

	_, err := resolveOpenapiRunActor(ctx, nil, nil)

	assert.Error(t, err)
}
