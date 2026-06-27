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

import "fmt"

type AIProductType string

const (
	AIProductTypeModel         AIProductType = "model"
	AIProductTypeMCPServer     AIProductType = "mcp_server"
	AIProductTypeStandardSkill AIProductType = "standard_skill"
	AIProductTypeAgentApp      AIProductType = "agent_app"
)

type AIProductVisibility string

const (
	AIProductVisibilityPrivate AIProductVisibility = "private"
	AIProductVisibilitySpace   AIProductVisibility = "space"
	AIProductVisibilityGlobal  AIProductVisibility = "global"
)

type AIProductStatus string

const (
	AIProductStatusDraft      AIProductStatus = "draft"
	AIProductStatusReviewing  AIProductStatus = "reviewing"
	AIProductStatusPublished  AIProductStatus = "published"
	AIProductStatusDeprecated AIProductStatus = "deprecated"
	AIProductStatusArchived   AIProductStatus = "archived"
)

type AIProductInstallationStatus string

const (
	AIProductInstallationActive      AIProductInstallationStatus = "active"
	AIProductInstallationDisabled    AIProductInstallationStatus = "disabled"
	AIProductInstallationUninstalled AIProductInstallationStatus = "uninstalled"
)

const (
	AIProductAuditActionCreate         = "product_create"
	AIProductAuditActionUpdate         = "product_update"
	AIProductAuditActionPublish        = "product_publish"
	AIProductAuditActionReviewApprove  = "product_review_approve"
	AIProductAuditActionReviewReject   = "product_review_reject"
	AIProductAuditActionInstall        = "product_install"
	AIProductAuditActionUninstall      = "product_uninstall"
	AIProductAuditActionUpgrade        = "product_upgrade"
	AIProductAuditActionSessionBind    = "session_runtime_bind"
	AIProductAuditActionSessionUpdate  = "session_runtime_update"
	AIProductAuditActionRuntimeResolve = "runtime_resolve"
	AIProductAuditActionSkillInject    = "runtime_skill_inject"
)

const (
	SourceRefTypeSkill = "skill"
)

type Product struct {
	ID               int64
	ProductID        int64
	SpaceID          int64
	CreatorID        int64
	Name             string
	Description      string
	Type             AIProductType
	Status           AIProductStatus
	Visibility       AIProductVisibility
	IconURI          string
	CoverURI         string
	Document         string
	Feature          map[string]any
	SourceRefType    string
	SourceRefID      int64
	LatestVersion    string
	PublishedVersion string
	Official         bool
	Featured         bool
	InstallCount     int64
	DownloadCount    int64
	CreatedAt        int64
	UpdatedAt        int64
}

type ProductVersion struct {
	ID              int64
	ProductID       int64
	Version         string
	SourceVersion   string
	Status          AIProductStatus
	ReviewStatus    string
	ReviewNote      string
	ReviewerID      int64
	ContentHash     string
	FeatureSnapshot map[string]any
	PublishedAt     int64
	CreatedAt       int64
	UpdatedAt       int64
}

type ProductInstallation struct {
	ID             int64
	InstallationID int64
	ProductID      int64
	ProductVersion string
	TargetSpaceID  int64
	TargetUserID   int64
	InstalledBy    int64
	Status         AIProductInstallationStatus
	InstallMode    string
	RuntimeConfig  map[string]any
	CreatedAt      int64
	UpdatedAt      int64
}

type AuditLog struct {
	ID             int64
	AuditID        int64
	ProductID      int64
	InstallationID int64
	SpaceID        int64
	UserID         int64
	Action         string
	TargetType     string
	TargetID       string
	Detail         map[string]any
	CreatedAt      int64
}

type SessionRuntimeConfig struct {
	ID               int64
	ConversationID   int64
	AgentID          int64
	SpaceID          int64
	ModelProductID   int64
	MCPProductIDs    []int64
	SkillProductIDs  []int64
	ToolPolicy       map[string]any
	ContextPolicy    map[string]any
	ResolvedSnapshot map[string]any
	CreatedBy        int64
	CreatedAt        int64
	UpdatedAt        int64
}

type ListProductsRequest struct {
	SpaceID    int64
	UserID     int64
	Type       AIProductType
	Visibility AIProductVisibility
	Status     AIProductStatus
	Installed  *bool
	Keyword    string
	Page       int32
	PageSize   int32
}

type ListProductsResult struct {
	Products []*Product
	Total    int32
}

type ListInstallationsRequest struct {
	SpaceID   int64
	UserID    int64
	ProductID int64
	Type      AIProductType
	Status    AIProductInstallationStatus
}

type ListAuditsRequest struct {
	ProductID int64
	SpaceID   int64
	UserID    int64
	Action    string
	Page      int32
	PageSize  int32
}

func InstallationKey(productID, spaceID, userID int64) string {
	return fmt.Sprintf("%d:%d:%d", productID, spaceID, userID)
}
