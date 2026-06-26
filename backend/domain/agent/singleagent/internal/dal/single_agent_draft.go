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

package dal

import (
	"context"
	"errors"

	"gorm.io/gen/field"
	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/bot_common"
	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"

	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/internal/dal/model"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/internal/dal/query"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/cache"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

type SingleAgentDraftDAO struct {
	idGen       idgen.IDGenerator
	dbQuery     *query.Query
	cacheClient cache.Cmdable
}

// Helper function to convert *string to string
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func NewSingleAgentDraftDAO(db *gorm.DB, idGen idgen.IDGenerator, cli cache.Cmdable) *SingleAgentDraftDAO {
	query.SetDefault(db)

	return &SingleAgentDraftDAO{
		idGen:       idGen,
		dbQuery:     query.Use(db),
		cacheClient: cli,
	}
}

func (sa *SingleAgentDraftDAO) Create(ctx context.Context, creatorID int64, draft *entity.SingleAgent) (draftID int64, err error) {
	id, err := sa.idGen.GenID(ctx)
	if err != nil {
		return 0, errorx.WrapByCode(err, errno.ErrAgentIDGenFailCode, errorx.KV("msg", "CreatePromptResource"))
	}

	return sa.CreateWithID(ctx, creatorID, id, draft)
}

func (sa *SingleAgentDraftDAO) CreateWithID(ctx context.Context, creatorID, agentID int64, draft *entity.SingleAgent) (draftID int64, err error) {
	po := sa.singleAgentDraftDo2Po(draft)
	po.AgentID = agentID
	po.CreatorID = creatorID

	err = sa.dbQuery.SingleAgentDraft.WithContext(ctx).Create(po)
	if err != nil {
		return 0, errorx.WrapByCode(err, errno.ErrAgentCreateDraftCode)
	}

	return agentID, nil
}

func (sa *SingleAgentDraftDAO) Get(ctx context.Context, agentID int64) (*entity.SingleAgent, error) {
	singleAgentDAOModel := sa.dbQuery.SingleAgentDraft
	singleAgent, err := sa.dbQuery.SingleAgentDraft.Where(singleAgentDAOModel.AgentID.Eq(agentID)).First()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrAgentGetCode)
	}

	do := sa.singleAgentDraftPo2Do(singleAgent)

	return do, nil
}

// GetBySourceProduct returns the most recently updated draft instance that the
// given creator materialised from the given source agent_app product, or
// (nil, nil) when none exists. It is used to make virtual-employee recruitment
// idempotent: re-recruiting the same product reuses the existing instance agent
// instead of leaking a fresh sandbox-backed draft each time.
func (sa *SingleAgentDraftDAO) GetBySourceProduct(ctx context.Context, creatorID, sourceProductID int64) (*entity.SingleAgent, error) {
	m := sa.dbQuery.SingleAgentDraft
	// source_product_id is not part of the generated query struct, so build an
	// ad-hoc field expression for it (table name matches the gen model).
	sourceProductIDField := field.NewInt64(m.TableName(), "source_product_id")

	singleAgent, err := m.WithContext(ctx).
		Where(m.CreatorID.Eq(creatorID), sourceProductIDField.Eq(sourceProductID)).
		Order(m.UpdatedAt.Desc()).
		First()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrAgentGetCode)
	}

	return sa.singleAgentDraftPo2Do(singleAgent), nil
}

func (sa *SingleAgentDraftDAO) MGet(ctx context.Context, agentIDs []int64) ([]*entity.SingleAgent, error) {
	sam := sa.dbQuery.SingleAgentDraft
	singleAgents, err := sam.WithContext(ctx).Where(sam.AgentID.In(agentIDs...)).Find()
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrAgentGetCode)
	}

	dos := make([]*entity.SingleAgent, 0, len(singleAgents))
	for _, singleAgent := range singleAgents {
		dos = append(dos, sa.singleAgentDraftPo2Do(singleAgent))
	}

	return dos, nil
}

func (sa *SingleAgentDraftDAO) Save(ctx context.Context, agentInfo *entity.SingleAgent) (err error) {
	po := sa.singleAgentDraftDo2Po(agentInfo)
	singleAgentDAOModel := sa.dbQuery.SingleAgentDraft

	err = singleAgentDAOModel.Where(singleAgentDAOModel.AgentID.Eq(agentInfo.AgentID)).Save(po)
	if err != nil {
		return errorx.WrapByCode(err, errno.ErrAgentUpdateCode)
	}

	return nil
}

func (sa *SingleAgentDraftDAO) Delete(ctx context.Context, spaceID, agentID int64) (err error) {
	po := sa.dbQuery.SingleAgentDraft
	_, err = po.WithContext(ctx).Where(po.AgentID.Eq(agentID), po.SpaceID.Eq(spaceID)).Delete()
	return err
}

func (sa *SingleAgentDraftDAO) ListBySpaceID(ctx context.Context, spaceID int64, limit int) ([]*entity.SingleAgent, error) {
	po := sa.dbQuery.SingleAgentDraft
	q := po.WithContext(ctx).Where(po.SpaceID.Eq(spaceID)).Order(po.ID.Asc())
	if limit > 0 {
		q = q.Limit(limit)
	}
	pos, err := q.Find()
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrAgentGetCode)
	}

	dos := make([]*entity.SingleAgent, 0, len(pos))
	for _, p := range pos {
		dos = append(dos, sa.singleAgentDraftPo2Do(p))
	}
	return dos, nil
}

func (sa *SingleAgentDraftDAO) singleAgentDraftPo2Do(po *model.SingleAgentDraft) *entity.SingleAgent {
	return &entity.SingleAgent{
		SingleAgent: &singleagent.SingleAgent{
			AgentID:                 po.AgentID,
			CreatorID:               po.CreatorID,
			SpaceID:                 po.SpaceID,
			Name:                    po.Name,
			Desc:                    getStringValue(po.Description),
			IconURI:                 po.IconURI,
			CreatedAt:               po.CreatedAt,
			UpdatedAt:               po.UpdatedAt,
			DeletedAt:               po.DeletedAt,
			ModelInfo:               po.ModelInfo,
			OnboardingInfo:          po.OnboardingInfo,
			Prompt:                  po.Prompt,
			Plugin:                  po.Plugin,
			Knowledge:               po.Knowledge,
			ExternalKnowledge:       po.ExternalKnowledge,
			Workflow:                po.Workflow,
			SuggestReply:            po.SuggestReply,
			JumpConfig:              po.JumpConfig,
			VariablesMetaID:         po.VariablesMetaID,
			BackgroundImageInfoList: po.BackgroundImageInfoList,
			Database:                po.DatabaseConfig,
			ShortcutCommand:         po.ShortcutCommand,
			BotMode:                 bot_common.BotMode(po.BotMode),
			LayoutInfo:              po.LayoutInfo,
			MemoryToolConfig:        po.MemoryToolConfig,
			BoundCards:              po.BoundCards,
			SkillInfoList:           skillPOsToSkillDOs(po.SkillInfoList),
			ForceToolReturn:         po.ForceToolReturn,
			AgentType:               ptr.From(po.AgentType),
			SuperAgentToolConfig:    po.SuperAgentToolConfig,
			SourceProductID:         po.SourceProductID,
			SourceProductVersion:    po.SourceProductVersion,
			Strategies:              po.StrategyConfig,
		},
	}
}

func (sa *SingleAgentDraftDAO) singleAgentDraftDo2Po(do *entity.SingleAgent) *model.SingleAgentDraft {
	return &model.SingleAgentDraft{
		AgentID:                 do.AgentID,
		CreatorID:               do.CreatorID,
		SpaceID:                 do.SpaceID,
		Name:                    do.Name,
		Description:             &do.Desc,
		IconURI:                 do.IconURI,
		CreatedAt:               do.CreatedAt,
		UpdatedAt:               do.UpdatedAt,
		DeletedAt:               do.DeletedAt,
		ModelInfo:               do.ModelInfo,
		OnboardingInfo:          do.OnboardingInfo,
		Prompt:                  do.Prompt,
		Plugin:                  do.Plugin,
		Knowledge:               do.Knowledge,
		ExternalKnowledge:       do.ExternalKnowledge,
		Workflow:                do.Workflow,
		SuggestReply:            do.SuggestReply,
		JumpConfig:              do.JumpConfig,
		VariablesMetaID:         do.VariablesMetaID,
		BackgroundImageInfoList: do.BackgroundImageInfoList,
		DatabaseConfig:          do.Database,
		ShortcutCommand:         do.ShortcutCommand,
		BotMode:                 int32(do.BotMode),
		LayoutInfo:              do.LayoutInfo,
		MemoryToolConfig:        do.MemoryToolConfig,
		BoundCards:              do.BoundCards,
		SkillInfoList:           skillDOsToPOs(do.SkillInfoList),
		ForceToolReturn:         do.ForceToolReturn,
		AgentType:               ptr.Of(do.AgentType),
		SuperAgentToolConfig:    do.SuperAgentToolConfig,
		SourceProductID:         do.SourceProductID,
		SourceProductVersion:    do.SourceProductVersion,
		StrategyConfig:          do.Strategies,
	}
}

func skillPOsToSkillDOs(pos []*model.SkillReference) []*singleagent.SkillReference {
	if len(pos) == 0 {
		return nil
	}
	result := make([]*singleagent.SkillReference, 0, len(pos))
	for _, po := range pos {
		result = append(result, &singleagent.SkillReference{
			SkillID:          po.SkillID,
			SkillName:        po.SkillName,
			SkillDescription: po.SkillDescription,
		})
	}
	return result
}

func skillDOsToPOs(dos []*singleagent.SkillReference) []*model.SkillReference {
	if len(dos) == 0 {
		return nil
	}
	result := make([]*model.SkillReference, 0, len(dos))
	for _, do := range dos {
		result = append(result, &model.SkillReference{
			SkillID:          do.SkillID,
			SkillName:        do.SkillName,
			SkillDescription: do.SkillDescription,
		})
	}
	return result
}
