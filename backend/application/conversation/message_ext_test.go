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

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/common"
	messageModel "github.com/ynet-dev/ynet-studio/backend/api/model/conversation/message"
	agentrunEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
	convEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/entity"
	messageEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/entity"
	userEntity "github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

type messageListConversationDomainFake struct {
	current           *convEntity.Conversation
	byID              map[int64]*convEntity.Conversation
	getByIDCalledWith int64
	currentCalled     bool
}

func (f *messageListConversationDomainFake) Create(context.Context, *convEntity.CreateMeta) (*convEntity.Conversation, error) {
	return nil, nil
}

func (f *messageListConversationDomainFake) GetByID(_ context.Context, id int64) (*convEntity.Conversation, error) {
	f.getByIDCalledWith = id
	return f.byID[id], nil
}

func (f *messageListConversationDomainFake) NewConversationCtx(context.Context, *convEntity.NewConversationCtxRequest) (*convEntity.NewConversationCtxResponse, error) {
	return nil, nil
}

func (f *messageListConversationDomainFake) GetCurrentConversation(context.Context, *convEntity.GetCurrent) (*convEntity.Conversation, error) {
	f.currentCalled = true
	return f.current, nil
}

func (f *messageListConversationDomainFake) Delete(context.Context, int64) error {
	return nil
}

func (f *messageListConversationDomainFake) List(context.Context, *convEntity.ListMeta) ([]*convEntity.Conversation, bool, error) {
	return nil, false, nil
}

func (f *messageListConversationDomainFake) UpdateExt(context.Context, int64, string) error {
	return nil
}

type messageListMessageDomainFake struct {
	lastList *messageEntity.ListMeta
}

func (f *messageListMessageDomainFake) List(_ context.Context, req *messageEntity.ListMeta) (*messageEntity.ListResult, error) {
	f.lastList = req
	return &messageEntity.ListResult{Direction: req.Direction}, nil
}

func (f *messageListMessageDomainFake) ListWithoutPair(context.Context, *messageEntity.ListMeta) (*messageEntity.ListResult, error) {
	return nil, nil
}

func (f *messageListMessageDomainFake) PreCreate(context.Context, *messageEntity.Message) (*messageEntity.Message, error) {
	return nil, nil
}

func (f *messageListMessageDomainFake) Create(context.Context, *messageEntity.Message) (*messageEntity.Message, error) {
	return nil, nil
}

func (f *messageListMessageDomainFake) GetByRunIDs(context.Context, int64, []int64) ([]*messageEntity.Message, error) {
	return nil, nil
}

func (f *messageListMessageDomainFake) GetByID(context.Context, int64) (*messageEntity.Message, error) {
	return nil, nil
}

func (f *messageListMessageDomainFake) Edit(context.Context, *messageEntity.Message) (*messageEntity.Message, error) {
	return nil, nil
}

func (f *messageListMessageDomainFake) Delete(context.Context, *messageEntity.DeleteMeta) error {
	return nil
}

func (f *messageListMessageDomainFake) Broken(context.Context, *messageEntity.BrokenMeta) error {
	return nil
}

type messageListRunDomainFake struct{}

func (f *messageListRunDomainFake) AgentRun(context.Context, *agentrunEntity.AgentRunMeta) (*schema.StreamReader[*agentrunEntity.AgentRunResponse], error) {
	return nil, nil
}

func (f *messageListRunDomainFake) Delete(context.Context, []int64) error {
	return nil
}

func (f *messageListRunDomainFake) Create(context.Context, *agentrunEntity.AgentRunMeta) (*agentrunEntity.RunRecordMeta, error) {
	return nil, nil
}

func (f *messageListRunDomainFake) GetByID(context.Context, int64) (*agentrunEntity.RunRecordMeta, error) {
	return nil, nil
}

func (f *messageListRunDomainFake) List(context.Context, *agentrunEntity.ListRunRecordMeta) ([]*agentrunEntity.RunRecordMeta, error) {
	return nil, nil
}

func TestBuildDExt2ApiExtPreservesToolCallID(t *testing.T) {
	extra := buildDExt2ApiExt(map[string]string{
		"call_id":        "call-web-search-1",
		"plugin_request": `{"query":"GLM-5.2"}`,
		"tool_name":      "web_search",
	})

	assert.Equal(t, "call-web-search-1", extra.CallID)
	assert.Equal(t, `{"query":"GLM-5.2"}`, extra.PluginRequest)
	assert.Equal(t, "web_search", extra.ToolName)
}

func TestGetMessageListUsesRequestedConversationID(t *testing.T) {
	const (
		userID                  = int64(42)
		agentID                 = int64(456)
		currentConversationID   = int64(111)
		requestedConversationID = int64(222)
		requestedSectionID      = int64(2220)
	)

	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userEntity.Session{UserID: userID})

	conversationFake := &messageListConversationDomainFake{
		current: &convEntity.Conversation{
			ID:        currentConversationID,
			SectionID: 1110,
			AgentID:   agentID,
			CreatorID: userID,
		},
		byID: map[int64]*convEntity.Conversation{
			requestedConversationID: {
				ID:        requestedConversationID,
				SectionID: requestedSectionID,
				AgentID:   agentID,
				CreatorID: userID,
			},
		},
	}
	messageFake := &messageListMessageDomainFake{}
	svc := &ConversationApplicationService{
		ConversationDomainSVC: conversationFake,
		MessageDomainSVC:      messageFake,
		AgentRunDomainSVC:     &messageListRunDomainFake{},
	}

	scene := common.Scene_SceneOpenApi
	resp, err := svc.GetMessageList(ctx, &messageModel.GetMessageListRequest{
		BotID:          "456",
		ConversationID: "222",
		Scene:          &scene,
		Count:          10,
		Cursor:         "0",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, messageFake.lastList)
	assert.False(t, conversationFake.currentCalled)
	assert.Equal(t, requestedConversationID, conversationFake.getByIDCalledWith)
	assert.Equal(t, "222", resp.ConversationID)
	assert.Equal(t, requestedConversationID, messageFake.lastList.ConversationID)
	assert.Equal(t, requestedSectionID, messageFake.lastList.SectionID)
}
