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

package release

import "encoding/json"

const (
	StatusDraft      = "draft"
	StatusPublished  = "published"
	StatusDeprecated = "deprecated"
)

// SpaceRelease corresponds to the space_release table
type SpaceRelease struct {
	ID            uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	SpaceID       int64           `gorm:"column:space_id"`
	Version       string          `gorm:"column:version"`
	Tag           *string         `gorm:"column:tag"`
	Description   *string         `gorm:"column:description"`
	SyncType      string          `gorm:"column:sync_type"`
	ParentVersion *string         `gorm:"column:parent_version"`
	Manifest      json.RawMessage `gorm:"column:manifest;type:json"`
	Statistics    json.RawMessage `gorm:"column:statistics;type:json"`
	PackageKey    string          `gorm:"column:package_key"`
	PackageSize   int64           `gorm:"column:package_size"`
	ContentHash   string          `gorm:"column:content_hash"`
	Status        string          `gorm:"column:status"`
	CreatedBy     int64           `gorm:"column:created_by"`
	PublishedAt   *int64          `gorm:"column:published_at"`
	CreatedAt     int64           `gorm:"column:created_at"`
	UpdatedAt     int64           `gorm:"column:updated_at"`
}

func (SpaceRelease) TableName() string {
	return "space_release"
}

// CreateReleaseRequest is the application-layer request to create a release
type CreateReleaseRequest struct {
	SpaceID     int64
	UserID      int64
	Version     string // user-specified or auto-incremented
	Tag         string
	Description string
	SyncType    string // "full" / "incremental"
}

// ReleaseDetail contains release info plus a temporary download URL
type ReleaseDetail struct {
	Release     *SpaceRelease
	DownloadURL string
	ExpiresAt   int64
}

// VersionDiff represents the difference between two versions
type VersionDiff struct {
	FromVersion string                      `json:"from_version"`
	ToVersion   string                      `json:"to_version"`
	Added       map[string][]ResourceSummary `json:"added"`
	Modified    map[string][]ResourceSummary `json:"modified"`
	Removed     map[string][]ResourceSummary `json:"removed"`
}

// ResourceSummary is a lightweight reference to a resource in a diff
type ResourceSummary struct {
	ID   int64  `json:"id,string"`
	Name string `json:"name"`
}
