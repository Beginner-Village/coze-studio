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

package model

import "gorm.io/gorm"

const TableNameStrategyCapability = "strategy_capability"

type StrategyCapability struct {
	ID               int64          `gorm:"column:id;primaryKey" json:"id"`
	StrategyID       int64          `gorm:"column:strategy_id;not null" json:"strategy_id"`
	ScenarioID       int64          `gorm:"column:scenario_id;not null" json:"scenario_id"`
	Type             string         `gorm:"column:type;not null" json:"type"`
	RefID            int64          `gorm:"column:ref_id" json:"ref_id"`
	RefSubID         int64          `gorm:"column:ref_sub_id" json:"ref_sub_id"`
	RefVersion       string         `gorm:"column:ref_version" json:"ref_version"`
	PromptContent    string         `gorm:"column:prompt_content" json:"prompt_content"`
	RetrieveConfig   string         `gorm:"column:retrieve_config" json:"retrieve_config"`
	AliasName        string         `gorm:"column:alias_name" json:"alias_name"`
	AliasDescription string         `gorm:"column:alias_description" json:"alias_description"`
	SortOrder        int32          `gorm:"column:sort_order;not null;default:0" json:"sort_order"`
	CreatedAt        int64          `gorm:"column:created_at;not null;autoCreateTime:milli" json:"created_at"`
	UpdatedAt        int64          `gorm:"column:updated_at;not null;autoUpdateTime:milli" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
}

func (*StrategyCapability) TableName() string { return TableNameStrategyCapability }
