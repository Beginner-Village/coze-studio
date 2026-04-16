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

type SyncExportRequest struct {
	SpaceID   int64  `json:"space_id,string" path:"space_id"`
	Mode      string `json:"mode"`
	SinceTime int64  `json:"since_time,omitempty"`
}

type SyncImportConfirmRequest struct {
	SpaceID     int64  `json:"space_id,string" path:"space_id"`
	ImportToken string `json:"import_token"`
}

type SyncExportResponse struct {
	Code int32           `json:"code"`
	Msg  string          `json:"msg"`
	Data *SyncExportData `json:"data,omitempty"`
}

type SyncExportData struct {
	DownloadURL string `json:"download_url"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
	ExpiresAt   int64  `json:"expires_at"`
}

type SyncImportPreviewResponse struct {
	Code int32                  `json:"code"`
	Msg  string                 `json:"msg"`
	Data *SyncImportPreviewData `json:"data,omitempty"`
}

type SyncImportPreviewData struct {
	ImportToken     string    `json:"import_token"`
	Plan            *SyncPlan `json:"plan"`
	Warnings        []string  `json:"warnings"`
	TokenExpiresAt  int64     `json:"token_expires_at"`
	IncomingVersion string    `json:"incoming_version,omitempty"`
	CurrentVersion  string    `json:"current_version,omitempty"`
}

type SyncPlan struct {
	Create map[string]int `json:"create"`
	Update map[string]int `json:"update"`
	Delete map[string]int `json:"delete"`
}

type SyncImportConfirmResponse struct {
	Code int32                  `json:"code"`
	Msg  string                 `json:"msg"`
	Data *SyncImportConfirmData `json:"data,omitempty"`
}

type SyncImportConfirmData struct {
	AgentsCreated    int        `json:"agents_created"`
	PluginsCreated   int        `json:"plugins_created"`
	WorkflowsCreated int        `json:"workflows_created"`
	VariablesCreated int        `json:"variables_created"`
	KnowledgeCreated int        `json:"knowledge_created"`
	Plan             *SyncPlan  `json:"plan"`
}

type SyncLastExportResponse struct {
	Code int32               `json:"code"`
	Msg  string              `json:"msg"`
	Data *SyncLastExportData `json:"data,omitempty"`
}

type SyncLastExportData struct {
	ExportTime int64  `json:"export_time"`
	SyncType   string `json:"sync_type"`
}

type SyncHistoryResponse struct {
	Code int32              `json:"code"`
	Msg  string             `json:"msg"`
	Data []*SyncHistoryItem `json:"data,omitempty"`
}

type SyncHistoryItem struct {
	ID                  uint64  `json:"id,string"`
	SourceSpaceID       int64   `json:"source_space_id,string"`
	TargetSpaceID       int64   `json:"target_space_id,string"`
	SyncType            string  `json:"sync_type"`
	Version             *string `json:"version,omitempty"`
	ExportTime          int64   `json:"export_time"`
	ImportTime          *int64  `json:"import_time,omitempty"`
	Status              int8    `json:"status"`
	SnapshotKey         *string `json:"snapshot_key,omitempty"`
	RollbackFromVersion *string `json:"rollback_from_version,omitempty"`
}
