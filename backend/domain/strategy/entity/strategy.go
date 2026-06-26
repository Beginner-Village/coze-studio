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

package entity

type CapabilityType = string

const (
	CapabilityTypeWorkflow  CapabilityType = "workflow"
	CapabilityTypePlugin    CapabilityType = "plugin"
	CapabilityTypeKnowledge CapabilityType = "knowledge"
	CapabilityTypePrompt    CapabilityType = "prompt"
)

const (
	StatusDraft     int32 = 0
	StatusPublished int32 = 1
)

type Strategy struct {
	ID, SpaceID, AppID, CreatorID int64
	Name, Description, IconURI    string
	Status                        int32
	Version                       string
	Scenarios                     []*Scenario // GetDetail 时填充
}

type Scenario struct {
	ID, StrategyID    int64
	Name, Description string
	SortOrder         int32
	Capabilities      []*Capability // GetDetail 时填充
}

type Capability struct {
	ID, StrategyID, ScenarioID    int64
	Type                          CapabilityType
	RefID, RefSubID               int64
	RefVersion                    string
	PromptContent, RetrieveConfig string
	AliasName, AliasDescription   string
	SortOrder                     int32
}
