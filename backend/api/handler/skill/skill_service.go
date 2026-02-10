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

package skill

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	"github.com/coze-dev/coze-studio/backend/api/internal/httputil"
	skillApp "github.com/coze-dev/coze-studio/backend/application/skill"
	"github.com/coze-dev/coze-studio/backend/domain/skill/entity"
)

type createSkillRequest struct {
	SpaceID     int64  `json:"space_id,string"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	IconURI     string `json:"icon_uri"`
}

type getSkillRequest struct {
	SkillID int64 `query:"skill_id,string"`
	SpaceID int64 `query:"space_id,string"`
}

type updateSkillRequest struct {
	SkillID     int64  `json:"skill_id,string"`
	SpaceID     int64  `json:"space_id,string"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	IconURI     string `json:"icon_uri"`
}

type deleteSkillRequest struct {
	SkillID int64 `json:"skill_id,string"`
	SpaceID int64 `json:"space_id,string"`
}

type listSkillsRequest struct {
	SpaceID  int64  `query:"space_id,string"`
	Page     int32  `query:"page"`
	PageSize int32  `query:"page_size"`
	Keyword  string `query:"keyword"`
}

type skillInfoResponse struct {
	SkillID     string `json:"skill_id"`
	SpaceID     string `json:"space_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	IconURI     string `json:"icon_uri"`
	CreatorID   string `json:"creator_id"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type response struct {
	Code int32       `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func entityToResponse(s *entity.Skill) *skillInfoResponse {
	if s == nil {
		return nil
	}
	return &skillInfoResponse{
		SkillID:     int64ToString(s.SkillID),
		SpaceID:     int64ToString(s.SpaceID),
		Name:        s.Name,
		Description: s.Description,
		Prompt:      s.Prompt,
		IconURI:     s.IconURI,
		CreatorID:   int64ToString(s.CreatorID),
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func int64ToString(v int64) string {
	return fmt.Sprintf("%d", v)
}

// CreateSkill handles POST /api/skill/create
func CreateSkill(ctx context.Context, c *app.RequestContext) {
	var req createSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.CreateSkill(ctx, req.SpaceID, req.Name, req.Description, req.Prompt, req.IconURI)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// GetSkill handles GET /api/skill/get
func GetSkill(ctx context.Context, c *app.RequestContext) {
	var req getSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.GetSkill(ctx, req.SkillID, req.SpaceID)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// UpdateSkill handles POST /api/skill/update
func UpdateSkill(ctx context.Context, c *app.RequestContext) {
	var req updateSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skill, err := skillApp.SkillApplicationSVC.UpdateSkill(ctx, req.SkillID, req.SpaceID, req.Name, req.Description, req.Prompt, req.IconURI)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{"skill_info": entityToResponse(skill)},
	})
}

// DeleteSkill handles POST /api/skill/delete
func DeleteSkill(ctx context.Context, c *app.RequestContext) {
	var req deleteSkillRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	err := skillApp.SkillApplicationSVC.DeleteSkill(ctx, req.SkillID, req.SpaceID)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
	})
}

// ListSkills handles GET /api/skill/list
func ListSkills(ctx context.Context, c *app.RequestContext) {
	var req listSkillsRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	skills, total, err := skillApp.SkillApplicationSVC.ListSkills(ctx, req.SpaceID, req.Page, req.PageSize, req.Keyword)
	if err != nil {
		httputil.InternalError(ctx, c, err)
		return
	}

	skillResponses := make([]*skillInfoResponse, 0, len(skills))
	for _, s := range skills {
		skillResponses = append(skillResponses, entityToResponse(s))
	}

	c.JSON(http.StatusOK, response{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{
			"skill_list": skillResponses,
			"total":      total,
		},
	})
}
