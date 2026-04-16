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

import "encoding/json"

// --- Create Release ---

type CreateReleaseRequest struct {
	SpaceID     int64  `json:"space_id,string" path:"space_id"`
	Version     string `json:"version,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Description string `json:"description,omitempty"`
	SyncType    string `json:"sync_type"`
}

type CreateReleaseResponse struct {
	Code int32              `json:"code"`
	Msg  string             `json:"msg"`
	Data *CreateReleaseData `json:"data,omitempty"`
}

type CreateReleaseData struct {
	Version     string `json:"version"`
	Status      string `json:"status"`
	PackageSize int64  `json:"package_size"`
	ContentHash string `json:"content_hash"`
}

// --- List Releases ---

type ListReleasesResponse struct {
	Code int32             `json:"code"`
	Msg  string            `json:"msg"`
	Data *ListReleasesData `json:"data,omitempty"`
}

type ListReleasesData struct {
	Items []*ReleaseItem `json:"items"`
	Total int64          `json:"total"`
}

type ReleaseItem struct {
	Version       string `json:"version"`
	Tag           string `json:"tag,omitempty"`
	Description   string `json:"description,omitempty"`
	SyncType      string `json:"sync_type"`
	ParentVersion string `json:"parent_version,omitempty"`
	PackageSize   int64  `json:"package_size"`
	Status        string `json:"status"`
	ContentHash   string `json:"content_hash"`
	CreatedBy     int64  `json:"created_by,string"`
	PublishedAt   *int64 `json:"published_at,omitempty"`
	CreatedAt     int64  `json:"created_at"`
}

// --- Get Release Detail ---

type GetReleaseResponse struct {
	Code int32           `json:"code"`
	Msg  string          `json:"msg"`
	Data *GetReleaseData `json:"data,omitempty"`
}

type GetReleaseData struct {
	ReleaseItem
	DownloadURL  string          `json:"download_url"`
	URLExpiresAt int64           `json:"url_expires_at"`
	Statistics   json.RawMessage `json:"statistics"`
	Manifest     json.RawMessage `json:"manifest"`
}

// --- Version Diff ---

type VersionDiffResponse struct {
	Code int32            `json:"code"`
	Msg  string           `json:"msg"`
	Data *VersionDiffData `json:"data,omitempty"`
}

type VersionDiffData struct {
	FromVersion string                    `json:"from_version"`
	ToVersion   string                    `json:"to_version"`
	Added       map[string][]ResourceRef  `json:"added"`
	Modified    map[string][]ResourceRef  `json:"modified"`
	Removed     map[string][]ResourceRef  `json:"removed"`
}

type ResourceRef struct {
	ID   int64  `json:"id,string"`
	Name string `json:"name"`
}

// --- Publish / Deprecate ---

type PublishReleaseResponse struct {
	Code int32  `json:"code"`
	Msg  string `json:"msg"`
}

// --- Rollback ---

type RollbackRequest struct {
	SpaceID       int64  `json:"space_id,string" path:"space_id"`
	TargetVersion string `json:"target_version"`
}

type RollbackResponse struct {
	Code int32         `json:"code"`
	Msg  string        `json:"msg"`
	Data *RollbackData `json:"data,omitempty"`
}

type RollbackData struct {
	RolledBackFrom string `json:"rolled_back_from"`
	RolledBackTo   string `json:"rolled_back_to"`
	SnapshotKey    string `json:"snapshot_key"`
}

// --- Current Version ---

type CurrentVersionResponse struct {
	Code int32               `json:"code"`
	Msg  string              `json:"msg"`
	Data *CurrentVersionData `json:"data,omitempty"`
}

type CurrentVersionData struct {
	Version    string `json:"version"`
	ImportedAt int64  `json:"imported_at"`
}
