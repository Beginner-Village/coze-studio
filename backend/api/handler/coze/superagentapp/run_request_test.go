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

package superagentapp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ynet-dev/ynet-studio/backend/domain/openauth/openapiauth/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

func TestPrepareRunRequestDerivesUserFromOpenAPIAuth(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &entity.ApiKey{UserID: 42})

	got, err := PrepareRunRequest(ctx, &RunInput{AgentID: 123}, false)

	assert.NoError(t, err)
	assert.Equal(t, int64(123), got.BotID)
	assert.Equal(t, "api-user-42", got.User)
	assert.Equal(t, "true", got.ExtraParams[AppServerFlag])
	assert.Equal(t, "json", got.ExtraParams[TransportParam])
}

func TestPrepareRunRequestStillRequiresUserWithoutOpenAPIAuth(t *testing.T) {
	got, err := PrepareRunRequest(context.Background(), &RunInput{AgentID: 123}, false)

	assert.Nil(t, got)
	assert.ErrorContains(t, err, "user_id")
}
