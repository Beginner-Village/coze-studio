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

const TableNameStrategyScenario = "strategy_scenario"

type StrategyScenario struct {
	ID          int64          `gorm:"column:id;primaryKey" json:"id"`
	StrategyID  int64          `gorm:"column:strategy_id;not null" json:"strategy_id"`
	Name        string         `gorm:"column:name;not null" json:"name"`
	Description string         `gorm:"column:description" json:"description"`
	SortOrder   int32          `gorm:"column:sort_order;not null;default:0" json:"sort_order"`
	CreatedAt   int64          `gorm:"column:created_at;not null;autoCreateTime:milli" json:"created_at"`
	UpdatedAt   int64          `gorm:"column:updated_at;not null;autoUpdateTime:milli" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
}

func (*StrategyScenario) TableName() string { return TableNameStrategyScenario }
