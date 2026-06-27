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

const TableNameStrategy = "strategy"

type Strategy struct {
	ID          int64          `gorm:"column:id;primaryKey;comment:ID" json:"id"`
	SpaceID     int64          `gorm:"column:space_id;not null;comment:Space ID" json:"space_id"`
	AppID       int64          `gorm:"column:app_id;comment:App ID" json:"app_id"`
	CreatorID   int64          `gorm:"column:creator_id;not null;comment:Creator ID" json:"creator_id"`
	Name        string         `gorm:"column:name;not null;comment:Strategy name" json:"name"`
	Description string         `gorm:"column:description;comment:Description / L1 hint" json:"description"`
	IconURI     string         `gorm:"column:icon_uri;comment:Icon Uri" json:"icon_uri"`
	Status      int32          `gorm:"column:status;not null;default:0;comment:0 draft 1 published" json:"status"`
	Version     string         `gorm:"column:version;comment:Published version" json:"version"`
	CreatedAt   int64          `gorm:"column:created_at;not null;autoCreateTime:milli" json:"created_at"`
	UpdatedAt   int64          `gorm:"column:updated_at;not null;autoUpdateTime:milli" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
}

func (*Strategy) TableName() string { return TableNameStrategy }
