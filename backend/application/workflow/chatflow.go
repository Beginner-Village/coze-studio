/*
 * Copyright 2025 coze-dev Authors
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

package workflow

import (
	"context"
	"fmt"
	"github.com/coze-dev/coze-studio/backend/types/consts"

	"runtime/debug"
	"strconv"

	"github.com/coze-dev/coze-studio/backend/api/model/ocean/cloud/workflow"
	"github.com/coze-dev/coze-studio/backend/application/base/ctxutil"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/pkg/errorx"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/maps"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/slices"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ternary"
	"github.com/coze-dev/coze-studio/backend/pkg/safego"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

func (w *ApplicationService) CreateApplicationConversationDef(ctx context.Context, req *workflow.CreateProjectConversationDefRequest) (resp *workflow.CreateProjectConversationDefResponse, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = safego.NewPanicErr(panicErr, debug.Stack())
		}

		if err != nil {
			err = vo.WrapIfNeeded(errno.ErrConversationOfAppOperationFail, err, errorx.KV("cause", vo.UnwrapRootErr(err).Error()))
		}
	}()

	var (
		spaceID = mustParseInt64(req.GetSpaceID())
		appID   = mustParseInt64(req.GetProjectID())
		userID  = ctxutil.MustGetUIDFromCtx(ctx)
	)

	if err := checkUserSpace(ctx, userID, spaceID); err != nil {
		return nil, err
	}

	uniqueID, err := GetWorkflowDomainSVC().CreateDraftConversationTemplate(ctx, &vo.CreateConversationTemplateMeta{
		AppID:   appID,
		SpaceID: spaceID,
		Name:    req.GetConversationName(),
		UserID:  userID,
	})
	if err != nil {
		return nil, err
	}

	return &workflow.CreateProjectConversationDefResponse{
		UniqueID: strconv.FormatInt(uniqueID, 10),
		SpaceID:  req.GetSpaceID(),
	}, err
}

func (w *ApplicationService) UpdateApplicationConversationDef(ctx context.Context, req *workflow.UpdateProjectConversationDefRequest) (resp *workflow.UpdateProjectConversationDefResponse, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = safego.NewPanicErr(panicErr, debug.Stack())
		}

		if err != nil {
			err = vo.WrapIfNeeded(errno.ErrConversationOfAppOperationFail, err, errorx.KV("cause", vo.UnwrapRootErr(err).Error()))
		}
	}()
	var (
		spaceID    = mustParseInt64(req.GetSpaceID())
		templateID = mustParseInt64(req.GetUniqueID())
		appID      = mustParseInt64(req.GetProjectID())
		userID     = ctxutil.MustGetUIDFromCtx(ctx)
	)

	if err := checkUserSpace(ctx, userID, spaceID); err != nil {
		return nil, err
	}

	err = GetWorkflowDomainSVC().UpdateDraftConversationTemplateName(ctx, appID, userID, templateID, req.GetConversationName())
	if err != nil {
		return nil, err
	}
	return &workflow.UpdateProjectConversationDefResponse{}, err
}

func (w *ApplicationService) DeleteApplicationConversationDef(ctx context.Context, req *workflow.DeleteProjectConversationDefRequest) (resp *workflow.DeleteProjectConversationDefResponse, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = safego.NewPanicErr(panicErr, debug.Stack())
		}

		if err != nil {
			err = vo.WrapIfNeeded(errno.ErrConversationOfAppOperationFail, err, errorx.KV("cause", vo.UnwrapRootErr(err).Error()))
		}
	}()
	var (
		appID      = mustParseInt64(req.GetProjectID())
		templateID = mustParseInt64(req.GetUniqueID())
	)
	if err := checkUserSpace(ctx, ctxutil.MustGetUIDFromCtx(ctx), mustParseInt64(req.GetSpaceID())); err != nil {
		return nil, err
	}
	if req.GetCheckOnly() {
		wfs, err := GetWorkflowDomainSVC().CheckWorkflowsToReplace(ctx, appID, templateID)
		if err != nil {
			return nil, err
		}
		resp = &workflow.DeleteProjectConversationDefResponse{NeedReplace: make([]*workflow.Workflow, 0)}
		for _, wf := range wfs {
			resp.NeedReplace = append(resp.NeedReplace, &workflow.Workflow{
				Name:       wf.Name,
				URL:        wf.IconURL,
				WorkflowID: strconv.FormatInt(wf.ID, 10),
			})
		}
		return resp, nil
	}

	wfID2ConversationName, err := maps.TransformKeyWithErrorCheck(req.GetReplace(), func(k1 string) (int64, error) {
		return strconv.ParseInt(k1, 10, 64)
	})

	rowsAffected, err := GetWorkflowDomainSVC().DeleteDraftConversationTemplate(ctx, templateID, wfID2ConversationName)
	if err != nil {
		return nil, err
	}
	if rowsAffected > 0 {
		return &workflow.DeleteProjectConversationDefResponse{
			Success: true,
		}, err
	}

	rowsAffected, err = GetWorkflowDomainSVC().DeleteDynamicConversation(ctx, vo.Draft, templateID)
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("delete conversation failed")
	}

	return &workflow.DeleteProjectConversationDefResponse{
		Success: true,
	}, nil

}

func (w *ApplicationService) ListApplicationConversationDef(ctx context.Context, req *workflow.ListProjectConversationRequest) (resp *workflow.ListProjectConversationResponse, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = safego.NewPanicErr(panicErr, debug.Stack())
		}

		if err != nil {
			err = vo.WrapIfNeeded(errno.ErrConversationOfAppOperationFail, err, errorx.KV("cause", vo.UnwrapRootErr(err).Error()))
		}
	}()
	var connectorID int64
	if len(req.GetConnectorID()) != 0 {
		connectorID = mustParseInt64(req.GetConnectorID())
	} else {
		connectorID = consts.CozeConnectorID
	}
	var (
		page                 = mustParseInt64(ternary.IFElse(req.GetCursor() == "", "0", req.GetCursor()))
		size                 = req.GetLimit()
		userID               = ctxutil.MustGetUIDFromCtx(ctx)
		spaceID              = mustParseInt64(req.GetSpaceID())
		appID                = mustParseInt64(req.GetProjectID())
		version              = req.ProjectVersion
		listConversationMeta = vo.ListConversationMeta{
			APPID:       appID,
			UserID:      userID,
			ConnectorID: connectorID,
		}
	)

	if err := checkUserSpace(ctx, userID, spaceID); err != nil {
		return nil, err
	}

	env := ternary.IFElse(req.GetCreateEnv() == workflow.CreateEnv_Draft, vo.Draft, vo.Online)
	if req.GetCreateMethod() == workflow.CreateMethod_ManualCreate {
		templates, err := GetWorkflowDomainSVC().ListConversationTemplate(ctx, env, &vo.ListConversationTemplatePolicy{
			AppID: appID,
			Page: &vo.Page{
				Page: int32(page),
				Size: int32(size),
			},
			NameLike: ternary.IFElse(len(req.GetNameLike()) == 0, nil, ptr.Of(req.GetNameLike())),
			Version:  version,
		})
		if err != nil {
			return nil, err
		}

		stsConversations, err := GetWorkflowDomainSVC().MGetStaticConversation(ctx, env, userID, connectorID, slices.Transform(templates, func(a *entity.ConversationTemplate) int64 {
			return a.TemplateID
		}))
		if err != nil {
			return nil, err
		}
		stsConversationMap := slices.ToMap(stsConversations, func(e *entity.StaticConversation) (int64, *entity.StaticConversation) {
			return e.TemplateID, e
		})

		resp = &workflow.ListProjectConversationResponse{Data: make([]*workflow.ProjectConversation, 0)}
		for _, tmpl := range templates {
			conversationID := ""
			if c, ok := stsConversationMap[tmpl.TemplateID]; ok {
				conversationID = strconv.FormatInt(c.ConversationID, 10)
			}
			resp.Data = append(resp.Data, &workflow.ProjectConversation{
				UniqueID:         strconv.FormatInt(tmpl.TemplateID, 10),
				ConversationName: tmpl.Name,
				ConversationID:   conversationID,
			})
		}
	}

	if req.GetCreateMethod() == workflow.CreateMethod_NodeCreate {
		dyConversations, err := GetWorkflowDomainSVC().ListDynamicConversation(ctx, env, &vo.ListConversationPolicy{
			ListConversationMeta: listConversationMeta,
			Page: &vo.Page{
				Page: int32(page),
				Size: int32(size),
			},
			NameLike: ternary.IFElse(len(req.GetNameLike()) == 0, nil, ptr.Of(req.GetNameLike())),
		})
		if err != nil {
			return nil, err
		}
		resp = &workflow.ListProjectConversationResponse{Data: make([]*workflow.ProjectConversation, 0, len(dyConversations))}
		resp.Data = append(resp.Data, slices.Transform(dyConversations, func(a *entity.DynamicConversation) *workflow.ProjectConversation {
			return &workflow.ProjectConversation{
				UniqueID:         strconv.FormatInt(a.ID, 10),
				ConversationName: a.Name,
				ConversationID:   strconv.FormatInt(a.ConversationID, 10),
			}
		})...)

	}

	return resp, nil
}
