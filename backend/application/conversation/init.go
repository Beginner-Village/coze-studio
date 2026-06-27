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
	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/application/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/repository"
	agentrun "github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/service"
	convRepo "github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/repository"
	conversation "github.com/ynet-dev/ynet-studio/backend/domain/conversation/conversation/service"
	msgRepo "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/repository"
	message "github.com/ynet-dev/ynet-studio/backend/domain/conversation/message/service"
	shortcutRepo "github.com/ynet-dev/ynet-studio/backend/domain/shortcutcmd/repository"
	"github.com/ynet-dev/ynet-studio/backend/domain/shortcutcmd/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/cache"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/imagex"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/modelmgr"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/storage"
)

type ServiceComponents struct {
	IDGen     idgen.IDGenerator
	DB        *gorm.DB
	TosClient storage.Storage
	ImageX    imagex.ImageX
	ModelMgr  modelmgr.Manager
	Cache     cache.Cmdable

	SingleAgentDomainSVC singleagent.SingleAgent
}

func InitService(s *ServiceComponents) *ConversationApplicationService {
	mDomainComponents := &message.Components{
		MessageRepo: msgRepo.NewMessageRepo(s.DB, s.IDGen),
	}
	messageDomainSVC := message.NewService(mDomainComponents)

	cDomainComponents := &conversation.Components{
		ConversationRepo: convRepo.NewConversationRepo(s.DB, s.IDGen),
	}

	conversationDomainSVC := conversation.NewService(cDomainComponents)

	arDomainComponents := &agentrun.Components{
		RunRecordRepo: repository.NewRunRecordRepo(s.DB, s.IDGen),
		ImagexSVC:     s.ImageX,
		TosClient:     s.TosClient,
		Cache:         s.Cache,
	}

	agentRunDomainSVC := agentrun.NewService(arDomainComponents)
	components := &service.Components{
		ShortCutCmdRepo: shortcutRepo.NewShortCutCmdRepo(s.DB, s.IDGen),
	}
	shortcutCmdDomainSVC := service.NewShortcutCommandService(components)

	ConversationSVC.AgentRunDomainSVC = agentRunDomainSVC
	ConversationSVC.MessageDomainSVC = messageDomainSVC
	ConversationSVC.ConversationDomainSVC = conversationDomainSVC
	ConversationSVC.appContext = s
	ConversationSVC.ShortcutDomainSVC = shortcutCmdDomainSVC

	// 🔥 关键修复：为OpenAPI服务也初始化appContext，以便访问ImageX服务
	ConversationOpenAPISVC.appContext = s
	ConversationOpenAPISVC.ShortcutDomainSVC = shortcutCmdDomainSVC

	return ConversationSVC
}
