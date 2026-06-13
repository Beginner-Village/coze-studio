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

// Status 操作结果
type Status int32

const (
	StatusSuccess Status = 1
	StatusFail    Status = 2
)

// OperationLog 一条审计记录（落库 + 查询返回）
type OperationLog struct {
	ID             int64
	SpaceID        int64
	OperatorID     int64
	Module         string
	ResourceType   int32
	ResourceID     int64
	ResourceName   string
	Action         string
	Description    string
	Method         string
	Path           string
	RequestSummary string
	Status         Status
	ErrorCode      string
	ClientIP       string
	DurationMs     int32
	LogID          string
	CreatedAt      int64 // 毫秒
}

// Event 中间件采集后投递给 worker 的事件（未分配 ID）
type Event = OperationLog

// ListFilter 查询过滤条件
type ListFilter struct {
	SpaceID      int64
	OperatorID   *int64
	ResourceType *int32
	Action       *string
	StartTime    *int64 // 毫秒
	EndTime      *int64 // 毫秒
	Keyword      *string
	Page         int32
	PageSize     int32
}
