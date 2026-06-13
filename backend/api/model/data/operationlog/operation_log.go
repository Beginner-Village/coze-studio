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

package operationlog

import "github.com/ynet-dev/ynet-studio/backend/api/model/base"

// ListRequest is the body for POST /api/operation_log/list. Owner/Admin only.
type ListRequest struct {
	SpaceID      int64      `form:"space_id" json:"space_id,string"`
	OperatorID   *int64     `form:"operator_id" json:"operator_id,string,omitempty"`
	ResourceType *int32     `form:"resource_type" json:"resource_type,omitempty"`
	Action       *string    `form:"action" json:"action,omitempty"`
	StartTime    *int64     `form:"start_time" json:"start_time,omitempty"`
	EndTime      *int64     `form:"end_time" json:"end_time,omitempty"`
	Keyword      *string    `form:"keyword" json:"keyword,omitempty"`
	Page         int32      `form:"page" json:"page"`
	PageSize     int32      `form:"page_size" json:"page_size"`
	Base         *base.Base `json:"Base,omitempty"`
}

// LogItemDTO is one operation-log row returned to the client.
type LogItemDTO struct {
	ID           int64  `json:"id,string"`
	OperatorID   int64  `json:"operator_id,string"`
	OperatorName string `json:"operator_name"`
	Module       string `json:"module"`
	ResourceType int32  `json:"resource_type"`
	ResourceID   int64  `json:"resource_id,string"`
	ResourceName string `json:"resource_name"`
	Action       string `json:"action"`
	Description  string `json:"description"`
	Status       int32  `json:"status"`
	ClientIP     string `json:"client_ip"`
	CreatedAt    int64  `json:"created_at"`
}

// ListResponse wraps the paginated operation-log query result.
type ListResponse struct {
	Code  int64        `json:"code"`
	Msg   string       `json:"msg"`
	Logs  []LogItemDTO `json:"logs"`
	Total int64        `json:"total"`
}
