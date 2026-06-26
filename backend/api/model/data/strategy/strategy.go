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

package strategy

// ---------- shared sub-models ----------

// CapabilityInfo mirrors entity.Capability for HTTP transport.
type CapabilityInfo struct {
	ID               int64  `json:"id,string"`
	StrategyID       int64  `json:"strategy_id,string"`
	ScenarioID       int64  `json:"scenario_id,string"`
	Type             string `json:"type"`
	RefID            int64  `json:"ref_id,string"`
	RefSubID         int64  `json:"ref_sub_id,string,omitempty"`
	RefVersion       string `json:"ref_version,omitempty"`
	PromptContent    string `json:"prompt_content,omitempty"`
	RetrieveConfig   string `json:"retrieve_config,omitempty"`
	AliasName        string `json:"alias_name,omitempty"`
	AliasDescription string `json:"alias_description,omitempty"`
	SortOrder        int32  `json:"sort_order"`
}

// ScenarioInfo mirrors entity.Scenario for HTTP transport.
type ScenarioInfo struct {
	ID           int64             `json:"id,string"`
	StrategyID   int64             `json:"strategy_id,string"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	SortOrder    int32             `json:"sort_order"`
	Capabilities []*CapabilityInfo `json:"capabilities,omitempty"`
}

// StrategyInfo mirrors entity.Strategy for HTTP transport.
type StrategyInfo struct {
	ID          int64           `json:"id,string"`
	SpaceID     int64           `json:"space_id,string"`
	AppID       int64           `json:"app_id,string,omitempty"`
	CreatorID   int64           `json:"creator_id,string"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	IconURI     string          `json:"icon_uri,omitempty"`
	Status      int32           `json:"status"`
	Version     string          `json:"version,omitempty"`
	Scenarios   []*ScenarioInfo `json:"scenarios,omitempty"`
}

// ---------- Strategy ----------

type CreateStrategyRequest struct {
	SpaceID     int64  `json:"space_id,string" vd:"$>0"`
	Name        string `json:"name" vd:"len($)>0"`
	Description string `json:"description,omitempty"`
	IconURI     string `json:"icon_uri,omitempty"`
}

type CreateStrategyResponse struct {
	Code int64         `json:"code"`
	Msg  string        `json:"msg"`
	Data *StrategyInfo `json:"data,omitempty"`
}

type GetStrategyDetailRequest struct {
	ID int64 `json:"id,string" vd:"$>0"`
}

type GetStrategyDetailResponse struct {
	Code int64         `json:"code"`
	Msg  string        `json:"msg"`
	Data *StrategyInfo `json:"data,omitempty"`
}

type UpdateStrategyRequest struct {
	ID          int64  `json:"id,string" vd:"$>0"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	IconURI     string `json:"icon_uri,omitempty"`
}

type UpdateStrategyResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

type DeleteStrategyRequest struct {
	ID int64 `json:"id,string" vd:"$>0"`
}

type DeleteStrategyResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

type PublishStrategyRequest struct {
	ID      int64  `json:"id,string" vd:"$>0"`
	Version string `json:"version,omitempty"`
}

type PublishStrategyResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// ---------- Scenario ----------

type CreateScenarioRequest struct {
	StrategyID  int64  `json:"strategy_id,string" vd:"$>0"`
	Name        string `json:"name" vd:"len($)>0"`
	Description string `json:"description,omitempty"`
	SortOrder   int32  `json:"sort_order,omitempty"`
}

type CreateScenarioResponse struct {
	Code int64         `json:"code"`
	Msg  string        `json:"msg"`
	Data *ScenarioInfo `json:"data,omitempty"`
}

type UpdateScenarioRequest struct {
	ID          int64  `json:"id,string" vd:"$>0"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	SortOrder   int32  `json:"sort_order,omitempty"`
}

type UpdateScenarioResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

type DeleteScenarioRequest struct {
	ID int64 `json:"id,string" vd:"$>0"`
}

type DeleteScenarioResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// ---------- Capability ----------

type AddCapabilityRequest struct {
	ScenarioID       int64  `json:"scenario_id,string" vd:"$>0"`
	StrategyID       int64  `json:"strategy_id,string" vd:"$>0"`
	Type             string `json:"type" vd:"len($)>0"`
	RefID            int64  `json:"ref_id,string" vd:"$>0"`
	RefSubID         int64  `json:"ref_sub_id,string,omitempty"`
	RefVersion       string `json:"ref_version,omitempty"`
	PromptContent    string `json:"prompt_content,omitempty"`
	RetrieveConfig   string `json:"retrieve_config,omitempty"`
	AliasName        string `json:"alias_name,omitempty"`
	AliasDescription string `json:"alias_description,omitempty"`
	SortOrder        int32  `json:"sort_order,omitempty"`
}

type AddCapabilityResponse struct {
	Code int64           `json:"code"`
	Msg  string          `json:"msg"`
	Data *CapabilityInfo `json:"data,omitempty"`
}

type UpdateCapabilityRequest struct {
	ID               int64  `json:"id,string" vd:"$>0"`
	RefVersion       string `json:"ref_version,omitempty"`
	PromptContent    string `json:"prompt_content,omitempty"`
	RetrieveConfig   string `json:"retrieve_config,omitempty"`
	AliasName        string `json:"alias_name,omitempty"`
	AliasDescription string `json:"alias_description,omitempty"`
	SortOrder        int32  `json:"sort_order,omitempty"`
}

type UpdateCapabilityResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

type DeleteCapabilityRequest struct {
	ID int64 `json:"id,string" vd:"$>0"`
}

type DeleteCapabilityResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}
