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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/run"
	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
)

const (
	AppServerFlag         = "super_agent_app_server"
	CapabilitiesParam     = "super_agent_capabilities"
	TransportParam        = "super_agent_transport"
	SpaceIDParam          = "super_agent_space_id"
	AgentIDParam          = "super_agent_agent_id"
	ClientIDParam         = "super_agent_client_id"
	CapabilitiesParamText = "sessions,messages,sandbox,workspace,skills,products,runtime_config,harness,artifacts,approvals,traces"
	ProtocolVersion       = "super-agent.app-server.v1"
)

type RunInput struct {
	SpaceID            int64
	AgentID            int64
	BotID              int64
	ConversationID     *int64
	User               string
	AdditionalMessages []*run.EnterMessage
	CustomVariables    map[string]string
	MetaData           map[string]string
	CustomConfig       *run.CustomConfig
	ExtraParams        map[string]string
	ConnectorID        *int64
	ShortcutCommand    *run.ShortcutCommandDetail
	ClientID           string
}

func PrepareRunRequest(ctx context.Context, req *RunInput, stream bool) (*run.ChatV3Request, error) {
	botID := req.BotID
	if botID == 0 {
		botID = req.AgentID
	}
	if botID == 0 {
		return nil, errors.New("agent_id or bot_id is required")
	}

	user := resolveRunUser(ctx, req.User)
	if user == "" {
		return nil, errors.New("user_id is required")
	}

	extraParams := make(map[string]string, len(req.ExtraParams)+6)
	for k, v := range req.ExtraParams {
		extraParams[k] = v
	}
	extraParams[AppServerFlag] = "true"
	extraParams[CapabilitiesParam] = CapabilitiesParamText
	if req.SpaceID > 0 {
		extraParams[SpaceIDParam] = strconv.FormatInt(req.SpaceID, 10)
	}
	extraParams[AgentIDParam] = strconv.FormatInt(botID, 10)
	if strings.TrimSpace(req.ClientID) != "" {
		extraParams[ClientIDParam] = strings.TrimSpace(req.ClientID)
	}
	if stream {
		extraParams[TransportParam] = "stream"
	} else {
		extraParams[TransportParam] = "json"
	}

	return &run.ChatV3Request{
		BotID:              botID,
		ConversationID:     req.ConversationID,
		User:               user,
		Stream:             ptr.Of(stream),
		AdditionalMessages: req.AdditionalMessages,
		CustomVariables:    req.CustomVariables,
		MetaData:           req.MetaData,
		CustomConfig:       req.CustomConfig,
		ExtraParams:        extraParams,
		ConnectorID:        req.ConnectorID,
		ShortcutCommand:    req.ShortcutCommand,
	}, nil
}

func resolveRunUser(ctx context.Context, explicit string) string {
	if user := strings.TrimSpace(explicit); user != "" {
		return user
	}
	apiAuth := ctxutil.GetApiAuthFromCtx(ctx)
	if apiAuth != nil && apiAuth.UserID != 0 {
		return fmt.Sprintf("api-user-%d", apiAuth.UserID)
	}
	return ""
}
