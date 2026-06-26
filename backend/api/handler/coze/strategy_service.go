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

package coze

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	apiModel "github.com/ynet-dev/ynet-studio/backend/api/model/data/strategy"
	strategyApp "github.com/ynet-dev/ynet-studio/backend/application/strategy"
)

// ---- Strategy handlers ----

// CreateStrategy creates a new strategy resource.
// @router /api/strategy/create [POST]
func CreateStrategy(ctx context.Context, c *app.RequestContext) {
	var req apiModel.CreateStrategyRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.CreateStrategy(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// GetStrategyDetail returns the full tree: strategy + scenarios + capabilities.
// @router /api/strategy/get_detail [POST]
func GetStrategyDetail(ctx context.Context, c *app.RequestContext) {
	var req apiModel.GetStrategyDetailRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.GetStrategyDetail(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// UpdateStrategy updates a strategy's metadata.
// @router /api/strategy/update [POST]
func UpdateStrategy(ctx context.Context, c *app.RequestContext) {
	var req apiModel.UpdateStrategyRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.UpdateStrategy(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// DeleteStrategy deletes a strategy and all its descendants.
// @router /api/strategy/delete [POST]
func DeleteStrategy(ctx context.Context, c *app.RequestContext) {
	var req apiModel.DeleteStrategyRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.DeleteStrategy(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// PublishStrategy publishes a strategy (sets status=Published, stamps version).
// @router /api/strategy/publish [POST]
func PublishStrategy(ctx context.Context, c *app.RequestContext) {
	var req apiModel.PublishStrategyRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.PublishStrategy(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// ---- Scenario handlers ----

// CreateScenario creates a scenario under a strategy.
// @router /api/strategy/scenario/create [POST]
func CreateScenario(ctx context.Context, c *app.RequestContext) {
	var req apiModel.CreateScenarioRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.CreateScenario(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// UpdateScenario updates a scenario's fields.
// @router /api/strategy/scenario/update [POST]
func UpdateScenario(ctx context.Context, c *app.RequestContext) {
	var req apiModel.UpdateScenarioRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.UpdateScenario(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// DeleteScenario deletes a scenario and all its capabilities.
// @router /api/strategy/scenario/delete [POST]
func DeleteScenario(ctx context.Context, c *app.RequestContext) {
	var req apiModel.DeleteScenarioRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.DeleteScenario(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// ---- Capability handlers ----

// AddCapability adds a capability to a scenario.
// @router /api/strategy/capability/add [POST]
func AddCapability(ctx context.Context, c *app.RequestContext) {
	var req apiModel.AddCapabilityRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.AddCapability(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// UpdateCapability updates a capability's fields.
// @router /api/strategy/capability/update [POST]
func UpdateCapability(ctx context.Context, c *app.RequestContext) {
	var req apiModel.UpdateCapabilityRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.UpdateCapability(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}

// DeleteCapability removes a capability from a scenario.
// @router /api/strategy/capability/delete [POST]
func DeleteCapability(ctx context.Context, c *app.RequestContext) {
	var req apiModel.DeleteCapabilityRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	resp, err := strategyApp.StrategyApplicationSVC.DeleteCapability(ctx, &req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(consts.StatusOK, resp)
}
