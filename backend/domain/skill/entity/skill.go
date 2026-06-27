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

// Skill status values (maps to the `status tinyint default 1` DB column).
const (
	// SkillStatusActive marks a skill that is enabled and loadable.
	SkillStatusActive int8 = 1
)

const (
	// SkillPublishScopePrivate keeps a skill visible only in its owning space.
	SkillPublishScopePrivate int8 = 1
	// SkillPublishScopeSpace publishes a skill to the owning space marketplace.
	SkillPublishScopeSpace int8 = 2
	// SkillPublishScopeGlobal publishes a skill to the global marketplace.
	SkillPublishScopeGlobal int8 = 3
)

const (
	// SkillReviewStatusPending marks a globally-published skill awaiting platform review.
	SkillReviewStatusPending int8 = 1
	// SkillReviewStatusApproved marks a skill that is approved and may appear in the marketplace.
	SkillReviewStatusApproved int8 = 2
	// SkillReviewStatusRejected marks a skill whose global publish was rejected.
	SkillReviewStatusRejected int8 = 3
)

// Skill is the domain entity for skill.
type Skill struct {
	SkillID     int64
	SpaceID     int64
	Name        string
	Description string
	Prompt      string
	// Files 是技能文件夹的完整内容树:相对路径 -> 文件内容(含 SKILL.md、scripts/*、references/*、assets/*)。
	// 「真·文件夹技能」(对齐 Anthropic / LangChain Agent Skills)的存储,运行时整棵同步到沙箱
	// /skills/<name>/,agent 用 read_file/run_bash 自行渐进式读取。
	Files            map[string]string
	Metadata         SkillMetadata
	IconURI          string
	CreatorID        int64
	Status           int8
	PublishScope     int8
	PublishedVersion int64
	PublishedAt      int64
	PublishedBy      int64
	// Version is the current content version of the skill. It starts at 1 on
	// create and is incremented on every successful update. Each value maps to
	// an immutable SkillVersion snapshot.
	Version   int64
	CreatedAt int64
	UpdatedAt int64
	// ReviewStatus tracks platform review for global-scope publishing. Space/private
	// scope skills are auto-approved. See SkillReviewStatus* constants.
	ReviewStatus int8
	ReviewNote   string
	ReviewerID   int64
	ReviewedAt   int64
}

type SkillMetadata struct {
	Version   string
	Category  string
	Tags      []string
	Platforms []string
}

// SkillVersion is an immutable snapshot of a skill's content at a given version.
// It lets callers read a pinned version instead of always reading the latest
// prompt, which is required for stable agent behavior after publish.
type SkillVersion struct {
	SkillID     int64
	Version     int64
	Name        string
	Description string
	Prompt      string
	Files       map[string]string
	IconURI     string
	// ContentHash is the sha256 hex digest of Prompt and Files.
	ContentHash string
	CreatedAt   int64
}

// SkillReference is a lightweight reference for Bot binding.
type SkillReference struct {
	SkillID          int64
	SkillName        string
	SkillDescription string
	// Version optionally pins the bound skill to a specific snapshot version.
	// Zero means "use latest". Reserved for future use; not yet enforced.
	Version int64
}

// ListRequest is the request for listing skills.
type ListRequest struct {
	SpaceID  int64
	Page     int32
	PageSize int32
	Keyword  string
}

// MarketplaceListRequest is the request for listing published skills.
type MarketplaceListRequest struct {
	SpaceID  int64
	Scope    int8
	Page     int32
	PageSize int32
	Keyword  string
}

// PendingReviewListRequest is the request for listing skills awaiting review.
type PendingReviewListRequest struct {
	SpaceID  int64
	Page     int32
	PageSize int32
}

// ListResponse is the response for listing skills.
type ListResponse struct {
	Skills []*Skill
	Total  int32
}
