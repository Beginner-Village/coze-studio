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
	"errors"
	"strings"

	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

const superAgentAppServerFlag = "super_agent_app_server"

type openapiRunActor struct {
	creatorID   int64
	connectorID int64
	isDraft     bool
}

func resolveOpenapiRunActor(ctx context.Context, requestConnectorID *int64, extraParams map[string]string) (*openapiRunActor, error) {
	actor := &openapiRunActor{}
	if apiKeyInfo := ctxutil.GetApiAuthFromCtx(ctx); apiKeyInfo != nil {
		actor.creatorID = apiKeyInfo.UserID
		actor.connectorID = apiKeyInfo.ConnectorID
		if apiKeyInfo.ConnectorID == consts.APIConnectorID && isSuperAgentAppServerRun(extraParams) {
			actor.connectorID = consts.CozeConnectorID
			actor.isDraft = true
		}
	} else if userID := ctxutil.GetUIDFromCtx(ctx); userID != nil {
		actor.creatorID = *userID
		actor.connectorID = consts.CozeConnectorID
		actor.isDraft = true
	} else {
		return nil, errors.New("openapi agent run requires api auth or user session")
	}

	if requestConnectorID != nil && *requestConnectorID == consts.WebSDKConnectorID {
		actor.connectorID = *requestConnectorID
		actor.isDraft = false
	}

	return actor, nil
}

func isSuperAgentAppServerRun(extraParams map[string]string) bool {
	return strings.EqualFold(strings.TrimSpace(extraParams[superAgentAppServerFlag]), "true")
}
