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

// OperationLog 是 operation_log 表的 gorm 持久化对象。
type OperationLog struct {
	ID             int64  `gorm:"column:id;primaryKey" json:"id"`
	SpaceID        int64  `gorm:"column:space_id" json:"space_id"`
	OperatorID     int64  `gorm:"column:operator_id" json:"operator_id"`
	Module         string `gorm:"column:module" json:"module"`
	ResourceType   int32  `gorm:"column:resource_type" json:"resource_type"`
	ResourceID     int64  `gorm:"column:resource_id" json:"resource_id"`
	ResourceName   string `gorm:"column:resource_name" json:"resource_name"`
	Action         string `gorm:"column:action" json:"action"`
	Description    string `gorm:"column:description" json:"description"`
	Method         string `gorm:"column:method" json:"method"`
	Path           string `gorm:"column:path" json:"path"`
	RequestSummary string `gorm:"column:request_summary" json:"request_summary"`
	Status         int32  `gorm:"column:status" json:"status"`
	ErrorCode      string `gorm:"column:error_code" json:"error_code"`
	ClientIP       string `gorm:"column:client_ip" json:"client_ip"`
	DurationMs     int32  `gorm:"column:duration_ms" json:"duration_ms"`
	LogID          string `gorm:"column:log_id" json:"log_id"`
	CreatedAt      int64  `gorm:"column:created_at" json:"created_at"`
}

func (OperationLog) TableName() string {
	return "operation_log"
}
