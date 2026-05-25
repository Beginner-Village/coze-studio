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

package space

import "github.com/ynet-dev/ynet-studio/backend/api/model/base"

type ResyncESRequest struct {
	SpaceID int64      `thrift:"space_id,1,required" form:"space_id" json:"space_id,string"`
	Base    *base.Base `thrift:"Base,255,optional" json:"Base,omitempty"`
}

type ResyncESCounts struct {
	ProjectDraft     int `json:"project_draft"`
	CozeResource     int `json:"coze_resource"`
	KbEntries        int `json:"kb_entries"`
	SliceReindexJobs int `json:"slice_reindex_jobs"`
}

type ResyncESResponse struct {
	Code   int64           `json:"code"`
	Msg    string          `json:"msg"`
	Counts *ResyncESCounts `json:"counts"`
}
