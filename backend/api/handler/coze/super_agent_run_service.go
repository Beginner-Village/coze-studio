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

package coze

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/sse"
	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/api/handler/coze/superagentapp"
	"github.com/ynet-dev/ynet-studio/backend/api/handler/coze/superagenttrace"
	messageModel "github.com/ynet-dev/ynet-studio/backend/api/model/conversation/message"
	"github.com/ynet-dev/ynet-studio/backend/api/model/conversation/run"
	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/application/conversation"
	singleagentapp "github.com/ynet-dev/ynet-studio/backend/application/singleagent"
	productEntity "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
	agentrunEntity "github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
	skillEntity "github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	sseImpl "github.com/ynet-dev/ynet-studio/backend/infra/impl/sse"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/pkg/observability"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

const (
	superAgentProtocolVersion                   = superagentapp.ProtocolVersion
	superAgentOpenAPIVersion                    = "3.1.0"
	superAgentOpenAPITitle                      = "Super Agent App Server API"
	superAgentOpenAPISchemaRoute                = "GET /api/super-agent/openapi.json"
	superAgentWorkspaceMaxReadBytes             = 25 * 1024 * 1024
	superAgentSandboxMaxTimeoutSec              = 300
	superAgentSandboxOutputMaxBytes             = 64 * 1024
	superAgentHarnessContextStrategy            = "tool-output-offload+session-summary"
	superAgentHarnessSummaryVersion             = "v1"
	superAgentHarnessCompactTrigger             = "history_bytes_exceeded"
	superAgentHarnessContextPath                = "/workspace/.agent/context-summary.json"
	superAgentHarnessSessionSummaryPathTemplate = "/workspace/.agent/sessions/{conversation_id}/context-summary.json"
	superAgentHarnessContextClearRouteValue     = "POST /api/super-agent/harness/context/clear"
	superAgentHarnessDefaultCompactMaxBytes     = 160 * 1024
	superAgentHarnessDefaultRecentMessages      = 16
	superAgentHarnessSummaryMaxRunes            = 1600
)

type superAgentWorkspaceRoot struct {
	Path     string `json:"path"`
	Label    string `json:"label"`
	Readonly bool   `json:"readonly"`
}

type superAgentManifestData struct {
	ProtocolVersion    string                          `json:"protocol_version"`
	Capabilities       []string                        `json:"capabilities"`
	Auth               superAgentManifestAuth          `json:"auth"`
	Stream             superAgentStreamContract        `json:"stream"`
	Sessions           superAgentSessionContract       `json:"sessions"`
	RuntimeConfig      superAgentRuntimeConfigContract `json:"runtime_config"`
	Messages           superAgentMessageContract       `json:"messages"`
	Trace              superAgentTraceContract         `json:"trace"`
	Approvals          superAgentApprovalContract      `json:"approvals"`
	Artifacts          superAgentArtifactContract      `json:"artifacts"`
	Products           superAgentProductContract       `json:"products"`
	Skills             superAgentSkillContract         `json:"skills"`
	SkillPublishScopes map[string]int8                 `json:"skill_publish_scopes"`
	WorkspaceRoots     []superAgentWorkspaceRoot       `json:"workspace_roots"`
	Workspace          superAgentWorkspaceContract     `json:"workspace"`
	Sandbox            superAgentSandboxContract       `json:"sandbox"`
	Harness            superAgentHarnessContract       `json:"harness"`
	ExternalAPI        superAgentExternalAPIContract   `json:"external_api"`
	OpenAPI            superAgentOpenAPIContract       `json:"openapi"`
	Routes             map[string]string               `json:"routes"`
}

type superAgentManifestAuth struct {
	Type   string `json:"type"`
	Header string `json:"header"`
	Scheme string `json:"scheme"`
}

type superAgentStreamContract struct {
	Transport   string   `json:"transport"`
	ContentType string   `json:"content_type"`
	Events      []string `json:"events"`
	DoneEvent   string   `json:"done_event"`
	ErrorEvent  string   `json:"error_event"`
}

type superAgentSessionContract struct {
	ListRoute      string                             `json:"list_route"`
	CreateRoute    string                             `json:"create_route"`
	GetRoute       string                             `json:"get_route"`
	RenameRoute    string                             `json:"rename_route"`
	DeleteRoute    string                             `json:"delete_route"`
	TitleExtKey    string                             `json:"title_ext_key"`
	MaxTitleLength int                                `json:"max_title_length"`
	RequestSchemas map[string]superAgentRequestSchema `json:"request_schemas"`
}

type superAgentMessageContract struct {
	ListRoute      string                             `json:"list_route"`
	MaxPageSize    int                                `json:"max_page_size"`
	OrderValues    []string                           `json:"order_values"`
	RequestSchemas map[string]superAgentRequestSchema `json:"request_schemas"`
}

type superAgentRuntimeConfigContract struct {
	GetRoute       string                             `json:"get_route"`
	UpdateRoute    string                             `json:"update_route"`
	DeleteRoute    string                             `json:"delete_route"`
	BindableTypes  []string                           `json:"bindable_types"`
	SnapshotFields []string                           `json:"snapshot_fields"`
	RequestSchemas map[string]superAgentRequestSchema `json:"request_schemas"`
}

type superAgentApprovalContract struct {
	ListRoute      string                             `json:"list_route"`
	ResolveRoute   string                             `json:"resolve_route"`
	DecisionRoot   string                             `json:"decision_root"`
	DecisionFormat string                             `json:"decision_format"`
	StatusValues   []string                           `json:"status_values"`
	DecisionValues []string                           `json:"decision_values"`
	ResponseFields []string                           `json:"response_fields"`
	RequestSchemas map[string]superAgentRequestSchema `json:"request_schemas"`
}

type superAgentWorkspaceContract struct {
	IdentifierFields []string                           `json:"identifier_fields"`
	ReadableRoots    []string                           `json:"readable_roots"`
	WritableRoots    []string                           `json:"writable_roots"`
	ReadMaxBytes     int64                              `json:"read_max_bytes"`
	BinaryEncoding   string                             `json:"binary_encoding"`
	WriteRoute       string                             `json:"write_route"`
	MoveRoute        string                             `json:"move_route"`
	MkdirRoute       string                             `json:"mkdir_route"`
	StatRoute        string                             `json:"stat_route"`
	GrepRoute        string                             `json:"grep_route"`
	GlobRoute        string                             `json:"glob_route"`
	EditRoute        string                             `json:"edit_route"`
	PatchRoute       string                             `json:"patch_route"`
	RequestSchemas   map[string]superAgentRequestSchema `json:"request_schemas"`
}

type superAgentSandboxContract struct {
	ExecRoute      string                             `json:"exec_route"`
	DefaultWorkdir string                             `json:"default_workdir"`
	MaxTimeoutSec  int                                `json:"max_timeout_sec"`
	OutputMaxBytes int                                `json:"output_max_bytes"`
	RequestSchemas map[string]superAgentRequestSchema `json:"request_schemas"`
}

type superAgentRequestSchema struct {
	Required      []string            `json:"required"`
	RequiredOneOf [][]string          `json:"required_one_of,omitempty"`
	Optional      []string            `json:"optional"`
	Aliases       map[string][]string `json:"aliases,omitempty"`
}

type superAgentArtifactContract struct {
	Root             string                             `json:"root"`
	ListRoute        string                             `json:"list_route"`
	DownloadRoute    string                             `json:"download_route"`
	DeleteRoute      string                             `json:"delete_route"`
	MoveRoute        string                             `json:"move_route"`
	MetadataFields   []string                           `json:"metadata_fields"`
	PreviewableMIMEs []string                           `json:"previewable_mimes"`
	MaxListItems     int32                              `json:"max_list_items"`
	RequestSchemas   map[string]superAgentRequestSchema `json:"request_schemas"`
}

type superAgentSkillContract struct {
	EntryFile      string                             `json:"entry_file"`
	FileRoots      []string                           `json:"file_roots"`
	PublishScopes  map[string]int8                    `json:"publish_scopes"`
	Routes         map[string]string                  `json:"routes"`
	RequestSchemas map[string]superAgentRequestSchema `json:"request_schemas"`
	Assets         superAgentSkillAssetContract       `json:"assets"`
	AgentTool      superAgentSkillToolContract        `json:"agent_tool"`
}

type superAgentSkillAssetContract struct {
	Root            string   `json:"root"`
	ListRoute       string   `json:"list_route"`
	GetRoute        string   `json:"get_route"`
	UpsertRoute     string   `json:"upsert_route"`
	DeleteRoute     string   `json:"delete_route"`
	AllowedMIMEs    []string `json:"allowed_mimes"`
	ContentEncoding string   `json:"content_encoding"`
}

type superAgentSkillToolContract struct {
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

type superAgentProductContract struct {
	ProductTypes            []string                           `json:"product_types"`
	VisibilityValues        []string                           `json:"visibility_values"`
	StatusValues            []string                           `json:"status_values"`
	SourceRefTypes          []string                           `json:"source_ref_types"`
	ListRoute               string                             `json:"list_route"`
	GetRoute                string                             `json:"get_route"`
	InstallRoute            string                             `json:"install_route"`
	UpgradeRoute            string                             `json:"upgrade_route"`
	UninstallRoute          string                             `json:"uninstall_route"`
	MarketplaceListRoute    string                             `json:"marketplace_list_route"`
	MarketplaceGetRoute     string                             `json:"marketplace_get_route"`
	MarketplaceInstallRoute string                             `json:"marketplace_install_route"`
	RequestSchemas          map[string]superAgentRequestSchema `json:"request_schemas"`
}

type superAgentHarnessContract struct {
	DeliverableRoot         string                         `json:"deliverable_root"`
	PlanPath                string                         `json:"plan_path"`
	SessionPlanPathTemplate string                         `json:"session_plan_path_template"`
	ToolOutputRoot          string                         `json:"tool_output_root"`
	StateRoute              string                         `json:"state_route"`
	StateFields             []string                       `json:"state_fields"`
	PlanUpdateRoute         string                         `json:"plan_update_route"`
	PlanRoute               string                         `json:"plan_route"`
	ToolOutputsRoute        string                         `json:"tool_outputs_route"`
	ToolOutputFields        []string                       `json:"tool_output_entry_fields"`
	CleanupRoute            string                         `json:"cleanup_route"`
	ContextClearRoute       string                         `json:"context_clear_route"`
	ContextPolicy           superAgentHarnessContextPolicy `json:"context_policy"`
	SnapshotRoute           string                         `json:"snapshot_route"`
	SnapshotFields          []string                       `json:"snapshot_fields"`
	ResumeRoute             string                         `json:"resume_route"`
	ResumeFields            []string                       `json:"resume_fields"`
	SkillRuntimeRoot        string                         `json:"skill_runtime_root"`
	SkillEntryFile          string                         `json:"skill_entry_file"`
	SkillFileRoots          []string                       `json:"skill_file_roots"`
	Tools                   []superAgentHarnessTool        `json:"tools"`
}

type superAgentHarnessContextPolicy struct {
	Strategy                   string `json:"strategy"`
	SummaryVersion             string `json:"summary_version"`
	Trigger                    string `json:"trigger"`
	MaxBytes                   int    `json:"max_bytes"`
	RecentMessages             int    `json:"recent_messages"`
	SummaryMaxRunes            int    `json:"summary_max_runes"`
	SummaryPath                string `json:"summary_path"`
	SessionSummaryPathTemplate string `json:"session_summary_path_template"`
	ClearRoute                 string `json:"clear_route"`
}

type superAgentHarnessTool struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	Available string `json:"available"`
	Mutates   bool   `json:"mutates"`
}

type superAgentExternalAPIContract struct {
	BasePath             string                             `json:"base_path"`
	ProtocolVersion      string                             `json:"protocol_version"`
	Auth                 superAgentManifestAuth             `json:"auth"`
	SchemaRoute          string                             `json:"schema_route"`
	SessionAuthSupported bool                               `json:"session_auth_supported"`
	Transports           []string                           `json:"transports"`
	IdentifierFields     []string                           `json:"identifier_fields"`
	Capabilities         []string                           `json:"capabilities"`
	EntryRoutes          map[string]string                  `json:"entry_routes"`
	RequestSchemas       map[string]superAgentRequestSchema `json:"request_schemas"`
	ClientMetadata       map[string]string                  `json:"client_metadata"`
}

type superAgentOpenAPIContract struct {
	Version     string                       `json:"version"`
	Title       string                       `json:"title"`
	SchemaRoute string                       `json:"schema_route"`
	Operations  []superAgentOpenAPIOperation `json:"operations"`
}

type superAgentOpenAPIOperation struct {
	OperationID   string `json:"operation_id"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	Category      string `json:"category"`
	Transport     string `json:"transport"`
	Mutates       bool   `json:"mutates"`
	RequestSchema string `json:"request_schema"`
}

type superAgentManifestResponse struct {
	Code int                     `json:"code"`
	Msg  string                  `json:"msg"`
	Data *superAgentManifestData `json:"data"`
}

func buildSuperAgentOpenAPIContract() superAgentOpenAPIContract {
	return superAgentOpenAPIContract{
		Version:     superAgentOpenAPIVersion,
		Title:       superAgentOpenAPITitle,
		SchemaRoute: superAgentOpenAPISchemaRoute,
		Operations:  superAgentOpenAPIOperations(),
	}
}

func superAgentOpenAPIOperations() []superAgentOpenAPIOperation {
	return []superAgentOpenAPIOperation{
		{OperationID: "runs.create", Method: http.MethodPost, Path: "/api/super-agent/runs/create", Category: "runs", Transport: "json", Mutates: true, RequestSchema: "RunsCreateRequest"},
		{OperationID: "runs.get", Method: http.MethodPost, Path: "/api/super-agent/runs/get", Category: "runs", Transport: "json", Mutates: false, RequestSchema: "RunsGetRequest"},
		{OperationID: "runs.list", Method: http.MethodPost, Path: "/api/super-agent/runs/list", Category: "runs", Transport: "json", Mutates: false, RequestSchema: "RunsListRequest"},
		{OperationID: "runs.reply", Method: http.MethodPost, Path: "/api/super-agent/runs/reply", Category: "runs", Transport: "json", Mutates: true, RequestSchema: "RunsReplyRequest"},
		{OperationID: "runs.stream", Method: http.MethodPost, Path: "/api/super-agent/runs/stream", Category: "runs", Transport: "sse", Mutates: true, RequestSchema: "RunsStreamRequest"},
		{OperationID: "runs.cancel", Method: http.MethodPost, Path: "/api/super-agent/runs/cancel", Category: "runs", Transport: "json", Mutates: true, RequestSchema: "RunsCancelRequest"},
		{OperationID: "sessions.create", Method: http.MethodPost, Path: "/api/super-agent/sessions/create", Category: "sessions", Transport: "json", Mutates: true, RequestSchema: "SessionsCreateRequest"},
		{OperationID: "sessions.get", Method: http.MethodPost, Path: "/api/super-agent/sessions/get", Category: "sessions", Transport: "json", Mutates: false, RequestSchema: "SessionsGetRequest"},
		{OperationID: "sessions.list", Method: http.MethodPost, Path: "/api/super-agent/sessions/list", Category: "sessions", Transport: "json", Mutates: false, RequestSchema: "SessionsListRequest"},
		{OperationID: "sessions.rename", Method: http.MethodPost, Path: "/api/super-agent/sessions/rename", Category: "sessions", Transport: "json", Mutates: true, RequestSchema: "SessionsRenameRequest"},
		{OperationID: "sessions.delete", Method: http.MethodPost, Path: "/api/super-agent/sessions/delete", Category: "sessions", Transport: "json", Mutates: true, RequestSchema: "SessionsDeleteRequest"},
		{OperationID: "runtime_config.get", Method: http.MethodPost, Path: "/api/super-agent/runtime-config/get", Category: "runtime_config", Transport: "json", Mutates: false, RequestSchema: "RuntimeConfigGetRequest"},
		{OperationID: "runtime_config.update", Method: http.MethodPost, Path: "/api/super-agent/runtime-config/update", Category: "runtime_config", Transport: "json", Mutates: true, RequestSchema: "RuntimeConfigUpdateRequest"},
		{OperationID: "runtime_config.delete", Method: http.MethodPost, Path: "/api/super-agent/runtime-config/delete", Category: "runtime_config", Transport: "json", Mutates: true, RequestSchema: "RuntimeConfigDeleteRequest"},
		{OperationID: "messages.list", Method: http.MethodPost, Path: "/api/super-agent/messages/list", Category: "messages", Transport: "json", Mutates: false, RequestSchema: "MessagesListRequest"},
		{OperationID: "approvals.list", Method: http.MethodPost, Path: "/api/super-agent/approvals/list", Category: "approvals", Transport: "json", Mutates: false, RequestSchema: "ApprovalsListRequest"},
		{OperationID: "approvals.resolve", Method: http.MethodPost, Path: "/api/super-agent/approvals/resolve", Category: "approvals", Transport: "json", Mutates: true, RequestSchema: "ApprovalsResolveRequest"},
		{OperationID: "harness.state", Method: http.MethodPost, Path: "/api/super-agent/harness/state", Category: "harness", Transport: "json", Mutates: false, RequestSchema: "HarnessStateRequest"},
		{OperationID: "harness.plan", Method: http.MethodPost, Path: "/api/super-agent/harness/plan", Category: "harness", Transport: "json", Mutates: true, RequestSchema: "HarnessPlanRequest"},
		{OperationID: "harness.tool_outputs", Method: http.MethodPost, Path: "/api/super-agent/harness/tool-outputs", Category: "harness", Transport: "json", Mutates: false, RequestSchema: "HarnessToolOutputsRequest"},
		{OperationID: "harness.cleanup", Method: http.MethodPost, Path: "/api/super-agent/harness/cleanup", Category: "harness", Transport: "json", Mutates: true, RequestSchema: "HarnessCleanupRequest"},
		{OperationID: "harness.context_clear", Method: http.MethodPost, Path: "/api/super-agent/harness/context/clear", Category: "harness", Transport: "json", Mutates: true, RequestSchema: "HarnessContextClearRequest"},
		{OperationID: "harness.resume", Method: http.MethodPost, Path: "/api/super-agent/harness/resume", Category: "harness", Transport: "json", Mutates: false, RequestSchema: "HarnessResumeRequest"},
		{OperationID: "harness.snapshot", Method: http.MethodPost, Path: "/api/super-agent/harness/snapshot", Category: "harness", Transport: "json", Mutates: false, RequestSchema: "HarnessSnapshotRequest"},
		{OperationID: "workspace.list", Method: http.MethodPost, Path: "/api/super-agent/workspace/list", Category: "workspace", Transport: "json", Mutates: false, RequestSchema: "WorkspaceListRequest"},
		{OperationID: "workspace.read", Method: http.MethodPost, Path: "/api/super-agent/workspace/read", Category: "workspace", Transport: "json", Mutates: false, RequestSchema: "WorkspaceReadRequest"},
		{OperationID: "workspace.upload", Method: http.MethodPost, Path: "/api/super-agent/workspace/upload", Category: "workspace", Transport: "json", Mutates: true, RequestSchema: "WorkspaceUploadRequest"},
		{OperationID: "workspace.write", Method: http.MethodPost, Path: "/api/super-agent/workspace/write", Category: "workspace", Transport: "json", Mutates: true, RequestSchema: "WorkspaceWriteRequest"},
		{OperationID: "workspace.download", Method: http.MethodPost, Path: "/api/super-agent/workspace/download", Category: "workspace", Transport: "json", Mutates: false, RequestSchema: "WorkspaceDownloadRequest"},
		{OperationID: "workspace.delete", Method: http.MethodPost, Path: "/api/super-agent/workspace/delete", Category: "workspace", Transport: "json", Mutates: true, RequestSchema: "WorkspaceDeleteRequest"},
		{OperationID: "workspace.move", Method: http.MethodPost, Path: "/api/super-agent/workspace/move", Category: "workspace", Transport: "json", Mutates: true, RequestSchema: "WorkspaceMoveRequest"},
		{OperationID: "workspace.mkdir", Method: http.MethodPost, Path: "/api/super-agent/workspace/mkdir", Category: "workspace", Transport: "json", Mutates: true, RequestSchema: "WorkspaceMkdirRequest"},
		{OperationID: "workspace.stat", Method: http.MethodPost, Path: "/api/super-agent/workspace/stat", Category: "workspace", Transport: "json", Mutates: false, RequestSchema: "WorkspaceStatRequest"},
		{OperationID: "workspace.grep", Method: http.MethodPost, Path: "/api/super-agent/workspace/grep", Category: "workspace", Transport: "json", Mutates: false, RequestSchema: "WorkspaceGrepRequest"},
		{OperationID: "workspace.glob", Method: http.MethodPost, Path: "/api/super-agent/workspace/glob", Category: "workspace", Transport: "json", Mutates: false, RequestSchema: "WorkspaceGlobRequest"},
		{OperationID: "workspace.edit", Method: http.MethodPost, Path: "/api/super-agent/workspace/edit", Category: "workspace", Transport: "json", Mutates: true, RequestSchema: "WorkspaceEditRequest"},
		{OperationID: "workspace.patch", Method: http.MethodPost, Path: "/api/super-agent/workspace/patch", Category: "workspace", Transport: "json", Mutates: true, RequestSchema: "WorkspacePatchRequest"},
		{OperationID: "sandbox.exec", Method: http.MethodPost, Path: "/api/super-agent/sandbox/exec", Category: "sandbox", Transport: "json", Mutates: true, RequestSchema: "SandboxExecRequest"},
		{OperationID: "traces.get", Method: http.MethodPost, Path: "/api/super-agent/traces/get", Category: "traces", Transport: "json", Mutates: false, RequestSchema: "TracesGetRequest"},
		{OperationID: "artifacts.list", Method: http.MethodPost, Path: "/api/super-agent/artifacts/list", Category: "artifacts", Transport: "json", Mutates: false, RequestSchema: "ArtifactsListRequest"},
		{OperationID: "artifacts.download", Method: http.MethodPost, Path: "/api/super-agent/artifacts/download", Category: "artifacts", Transport: "json", Mutates: false, RequestSchema: "ArtifactsDownloadRequest"},
		{OperationID: "artifacts.delete", Method: http.MethodPost, Path: "/api/super-agent/artifacts/delete", Category: "artifacts", Transport: "json", Mutates: true, RequestSchema: "ArtifactsDeleteRequest"},
		{OperationID: "artifacts.move", Method: http.MethodPost, Path: "/api/super-agent/artifacts/move", Category: "artifacts", Transport: "json", Mutates: true, RequestSchema: "ArtifactsMoveRequest"},
		{OperationID: "skills.create", Method: http.MethodPost, Path: "/api/super-agent/skills/create", Category: "skills", Transport: "json", Mutates: true, RequestSchema: "SkillsCreateRequest"},
		{OperationID: "skills.validate_package", Method: http.MethodPost, Path: "/api/super-agent/skills/validate-package", Category: "skills", Transport: "json", Mutates: false, RequestSchema: "SkillsValidatePackageRequest"},
		{OperationID: "skills.import", Method: http.MethodPost, Path: "/api/super-agent/skills/import", Category: "skills", Transport: "json", Mutates: true, RequestSchema: "SkillsImportRequest"},
		{OperationID: "skills.import_runtime", Method: http.MethodPost, Path: "/api/super-agent/skills/import-runtime", Category: "skills", Transport: "json", Mutates: true, RequestSchema: "SkillsImportRuntimeRequest"},
		{OperationID: "skills.export", Method: http.MethodPost, Path: "/api/super-agent/skills/export", Category: "skills", Transport: "json", Mutates: false, RequestSchema: "SkillsExportRequest"},
		{OperationID: "skills.get", Method: http.MethodGet, Path: "/api/super-agent/skills/get", Category: "skills", Transport: "json", Mutates: false, RequestSchema: "SkillsGetRequest"},
		{OperationID: "skills.update", Method: http.MethodPost, Path: "/api/super-agent/skills/update", Category: "skills", Transport: "json", Mutates: true, RequestSchema: "SkillsUpdateRequest"},
		{OperationID: "skills.delete", Method: http.MethodPost, Path: "/api/super-agent/skills/delete", Category: "skills", Transport: "json", Mutates: true, RequestSchema: "SkillsDeleteRequest"},
		{OperationID: "skills.publish", Method: http.MethodPost, Path: "/api/super-agent/skills/publish", Category: "skills", Transport: "json", Mutates: true, RequestSchema: "SkillsPublishRequest"},
		{OperationID: "skills.list", Method: http.MethodGet, Path: "/api/super-agent/skills/list", Category: "skills", Transport: "json", Mutates: false, RequestSchema: "SkillsListRequest"},
		{OperationID: "skills.assets.list", Method: http.MethodGet, Path: "/api/super-agent/skills/assets/list", Category: "skills", Transport: "json", Mutates: false, RequestSchema: "SkillsAssetsListRequest"},
		{OperationID: "skills.assets.get", Method: http.MethodGet, Path: "/api/super-agent/skills/assets/get", Category: "skills", Transport: "json", Mutates: false, RequestSchema: "SkillsAssetsGetRequest"},
		{OperationID: "skills.assets.upsert", Method: http.MethodPost, Path: "/api/super-agent/skills/assets/upsert", Category: "skills", Transport: "json", Mutates: true, RequestSchema: "SkillsAssetsUpsertRequest"},
		{OperationID: "skills.assets.delete", Method: http.MethodPost, Path: "/api/super-agent/skills/assets/delete", Category: "skills", Transport: "json", Mutates: true, RequestSchema: "SkillsAssetsDeleteRequest"},
		{OperationID: "skills.marketplace.list", Method: http.MethodGet, Path: "/api/super-agent/marketplace/list", Category: "marketplace", Transport: "json", Mutates: false, RequestSchema: "SkillsMarketplaceListRequest"},
		{OperationID: "skills.marketplace.get", Method: http.MethodGet, Path: "/api/super-agent/marketplace/get", Category: "marketplace", Transport: "json", Mutates: false, RequestSchema: "SkillsMarketplaceGetRequest"},
		{OperationID: "skills.marketplace.install", Method: http.MethodPost, Path: "/api/super-agent/marketplace/install", Category: "marketplace", Transport: "json", Mutates: true, RequestSchema: "SkillsMarketplaceInstallRequest"},
		{OperationID: "products.list", Method: http.MethodPost, Path: "/api/super-agent/products/list", Category: "products", Transport: "json", Mutates: false, RequestSchema: "ProductsListRequest"},
		{OperationID: "products.get", Method: http.MethodPost, Path: "/api/super-agent/products/get", Category: "products", Transport: "json", Mutates: false, RequestSchema: "ProductsGetRequest"},
		{OperationID: "products.install", Method: http.MethodPost, Path: "/api/super-agent/products/install", Category: "products", Transport: "json", Mutates: true, RequestSchema: "ProductsInstallRequest"},
		{OperationID: "products.upgrade", Method: http.MethodPost, Path: "/api/super-agent/products/upgrade", Category: "products", Transport: "json", Mutates: true, RequestSchema: "ProductsUpgradeRequest"},
		{OperationID: "products.uninstall", Method: http.MethodPost, Path: "/api/super-agent/products/uninstall", Category: "products", Transport: "json", Mutates: true, RequestSchema: "ProductsUninstallRequest"},
		{OperationID: "marketplace.products.list", Method: http.MethodPost, Path: "/api/super-agent/marketplace/products/list", Category: "marketplace", Transport: "json", Mutates: false, RequestSchema: "MarketplaceProductsListRequest"},
		{OperationID: "marketplace.products.get", Method: http.MethodPost, Path: "/api/super-agent/marketplace/products/get", Category: "marketplace", Transport: "json", Mutates: false, RequestSchema: "MarketplaceProductsGetRequest"},
		{OperationID: "marketplace.products.install", Method: http.MethodPost, Path: "/api/super-agent/marketplace/products/install", Category: "marketplace", Transport: "json", Mutates: true, RequestSchema: "MarketplaceProductsInstallRequest"},
	}
}

func superAgentOpenAPIRequestSchemas() map[string]superAgentRequestSchema {
	return map[string]superAgentRequestSchema{
		"RunsCreateRequest":                 {Required: []string{}, RequiredOneOf: [][]string{{"agent_id", "bot_id"}}, Optional: []string{"space_id", "user_id", "additional_messages", "custom_variables", "meta_data", "custom_config", "extra_params", "connector_id", "shortcut_command", "client_id"}},
		"RunsStreamRequest":                 {Required: []string{}, RequiredOneOf: [][]string{{"agent_id", "bot_id"}}, Optional: []string{"space_id", "user_id", "additional_messages", "custom_variables", "meta_data", "custom_config", "extra_params", "connector_id", "shortcut_command", "client_id"}},
		"RunsReplyRequest":                  {Required: []string{"conversation_id"}, RequiredOneOf: [][]string{{"agent_id", "bot_id"}}, Optional: []string{"space_id", "user_id", "additional_messages", "custom_variables", "meta_data", "custom_config", "extra_params", "connector_id", "shortcut_command", "client_id"}},
		"RunsGetRequest":                    {Required: []string{"run_id"}, Optional: []string{"space_id", "agent_id", "bot_id", "user_id", "client_id"}},
		"RunsListRequest":                   {Required: []string{"conversation_id"}, Optional: []string{"space_id", "agent_id", "bot_id", "limit", "order_by", "before_id", "after_id", "user_id", "client_id"}},
		"RunsCancelRequest":                 {Required: []string{"run_id"}, Optional: []string{"space_id", "agent_id", "bot_id", "user_id", "client_id"}},
		"SessionsCreateRequest":             {Required: []string{}, RequiredOneOf: [][]string{{"agent_id", "bot_id"}}, Optional: []string{"space_id", "connector_id", "title", "user_id", "client_id"}},
		"SessionsGetRequest":                {Required: []string{"conversation_id"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "include_snapshot", "run_id", "trace_limit", "run_limit", "message_limit", "artifact_limit", "workspace_path", "workspace_recursive", "include_harness", "include_messages", "include_runs", "include_workspace", "include_context", "include_tool_outputs", "include_tool_output_content", "include_artifacts", "include_trace", "include_approvals", "user_id", "client_id"}},
		"SessionsListRequest":               {Required: []string{}, RequiredOneOf: [][]string{{"agent_id", "bot_id"}}, Optional: []string{"space_id", "connector_id", "page", "page_size", "user_id", "client_id"}},
		"SessionsRenameRequest":             {Required: []string{"conversation_id", "title"}, Optional: []string{"user_id", "client_id"}},
		"SessionsDeleteRequest":             {Required: []string{"conversation_id"}, Optional: []string{"user_id", "client_id"}},
		"RuntimeConfigGetRequest":           {Required: []string{"conversation_id"}, Optional: []string{"user_id", "client_id"}},
		"RuntimeConfigUpdateRequest":        {Required: []string{"conversation_id", "space_id"}, Optional: []string{"agent_id", "bot_id", "model_product_id", "mcp_product_ids", "skill_product_ids", "tool_policy", "context_policy", "user_id", "client_id"}},
		"RuntimeConfigDeleteRequest":        {Required: []string{"conversation_id"}, Optional: []string{"user_id", "client_id"}},
		"MessagesListRequest":               {Required: []string{"conversation_id"}, Optional: []string{"limit", "before_id", "after_id", "order_by", "user_id", "client_id"}},
		"ApprovalsListRequest":              {Required: []string{"conversation_id"}, Optional: []string{"space_id", "agent_id", "bot_id", "run_id", "status", "limit", "user_id", "client_id"}},
		"ApprovalsResolveRequest":           {Required: []string{"approval_id", "decision"}, Optional: []string{"space_id", "agent_id", "bot_id", "conversation_id", "run_id", "note", "user_id", "client_id"}},
		"HarnessStateRequest":               {Required: []string{}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "conversation_id"}},
		"HarnessPlanRequest":                {Required: []string{}, RequiredOneOf: [][]string{{"plan", "items"}}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "conversation_id"}},
		"HarnessToolOutputsRequest":         {Required: []string{}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "conversation_id", "path", "recursive", "include_content"}},
		"HarnessCleanupRequest":             {Required: []string{}, RequiredOneOf: [][]string{{"paths"}, {"keep_latest"}}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "paths", "keep_latest", "prefix", "dry_run"}},
		"HarnessContextClearRequest":        {Required: []string{}, RequiredOneOf: [][]string{{"agent_id", "bot_id"}}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "conversation_id", "dry_run"}},
		"HarnessResumeRequest":              {Required: []string{}, RequiredOneOf: [][]string{{"conversation_id"}}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "message_limit", "artifact_limit", "include_messages", "include_harness", "include_context", "include_tool_outputs", "include_tool_output_content", "include_artifacts", "include_trace", "include_approvals", "run_id", "trace_limit"}},
		"HarnessSnapshotRequest":            {Required: []string{}, RequiredOneOf: [][]string{{"conversation_id", "agent_id", "bot_id"}}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "conversation_id", "run_id", "trace_limit", "run_limit", "message_limit", "artifact_limit", "workspace_path", "workspace_recursive", "include_harness", "include_messages", "include_runs", "include_workspace", "include_context", "include_tool_outputs", "include_tool_output_content", "include_artifacts", "include_trace", "include_approvals", "include_resume"}},
		"WorkspaceListRequest":              {Required: []string{}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "path", "recursive"}},
		"WorkspaceReadRequest":              {Required: []string{"path"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "encoding"}},
		"WorkspaceUploadRequest":            {Required: []string{"path", "content"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "mime", "encoding", "is_base64"}},
		"WorkspaceWriteRequest":             {Required: []string{"path", "content"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "encoding", "is_base64"}},
		"WorkspaceDownloadRequest":          {Required: []string{"path"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id"}},
		"WorkspaceDeleteRequest":            {Required: []string{"path"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id"}},
		"WorkspaceMoveRequest":              {Required: []string{}, RequiredOneOf: [][]string{{"path", "from_path"}, {"target_path", "to_path"}}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id"}},
		"WorkspaceMkdirRequest":             {Required: []string{"path"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id"}},
		"WorkspaceStatRequest":              {Required: []string{"path"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id"}},
		"WorkspaceGrepRequest":              {Required: []string{"pattern"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "path", "case_sensitive", "include", "exclude"}},
		"WorkspaceGlobRequest":              {Required: []string{"pattern"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "path"}},
		"WorkspaceEditRequest":              {Required: []string{"path", "old_string", "new_string"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id"}},
		"WorkspacePatchRequest":             {Required: []string{"patch"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "workdir", "work_dir"}, Aliases: map[string][]string{"workdir": {"work_dir"}}},
		"SandboxExecRequest":                {Required: []string{"command"}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "workdir", "work_dir", "timeout_sec"}, Aliases: map[string][]string{"workdir": {"work_dir"}}},
		"TracesGetRequest":                  {Required: []string{"conversation_id"}, Optional: []string{"space_id", "agent_id", "bot_id", "run_id", "limit"}},
		"ArtifactsListRequest":              {Required: []string{}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "path", "limit"}},
		"ArtifactsDownloadRequest":          {Required: []string{}, RequiredOneOf: [][]string{{"artifact_id", "path"}}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id"}},
		"ArtifactsDeleteRequest":            {Required: []string{}, RequiredOneOf: [][]string{{"artifact_id", "path"}}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id"}},
		"ArtifactsMoveRequest":              {Required: []string{"target_path"}, RequiredOneOf: [][]string{{"artifact_id", "path"}}, Optional: []string{"space_id", "agent_id", "bot_id", "connector_id"}},
		"SkillsCreateRequest":               {Required: []string{"space_id", "name", "files"}, Optional: []string{"description", "prompt", "icon_uri"}},
		"SkillsValidatePackageRequest":      {Required: []string{"content"}, Optional: []string{"filename"}},
		"SkillsImportRequest":               {Required: []string{"space_id", "content"}, Optional: []string{"filename", "icon_uri"}},
		"SkillsImportRuntimeRequest":        {Required: []string{"agent_id", "name"}, Optional: []string{"bot_id", "connector_id", "skill_id", "icon_uri", "publish_scope"}},
		"SkillsExportRequest":               {Required: []string{"space_id", "skill_id"}, Optional: []string{}},
		"SkillsGetRequest":                  {Required: []string{"space_id", "skill_id"}, Optional: []string{}},
		"SkillsUpdateRequest":               {Required: []string{"space_id", "skill_id"}, Optional: []string{"name", "description", "prompt", "icon_uri", "files"}},
		"SkillsDeleteRequest":               {Required: []string{"space_id", "skill_id"}, Optional: []string{}},
		"SkillsPublishRequest":              {Required: []string{"space_id", "skill_id", "scope"}, Optional: []string{}},
		"SkillsListRequest":                 {Required: []string{"space_id"}, Optional: []string{"page", "page_size", "keyword"}},
		"SkillsAssetsListRequest":           {Required: []string{"space_id", "skill_id"}, Optional: []string{}},
		"SkillsAssetsGetRequest":            {Required: []string{"space_id", "skill_id", "path"}, Optional: []string{}},
		"SkillsAssetsUpsertRequest":         {Required: []string{"space_id", "skill_id", "path", "content"}, Optional: []string{"mime"}},
		"SkillsAssetsDeleteRequest":         {Required: []string{"space_id", "skill_id", "path"}, Optional: []string{}},
		"SkillsMarketplaceListRequest":      {Required: []string{"space_id"}, Optional: []string{"scope", "page", "page_size", "keyword"}},
		"SkillsMarketplaceGetRequest":       {Required: []string{"space_id", "skill_id"}, Optional: []string{}},
		"SkillsMarketplaceInstallRequest":   {Required: []string{"space_id", "skill_id"}, Optional: []string{}},
		"ProductsListRequest":               {Required: []string{"space_id"}, Optional: []string{"type", "page", "page_size", "keyword"}},
		"ProductsGetRequest":                {Required: []string{"space_id", "product_id"}, Optional: []string{}},
		"ProductsInstallRequest":            {Required: []string{"space_id", "product_id"}, Optional: []string{"version"}},
		"ProductsUpgradeRequest":            {Required: []string{"space_id", "product_id"}, Optional: []string{}},
		"ProductsUninstallRequest":          {Required: []string{"space_id", "product_id"}, Optional: []string{}},
		"MarketplaceProductsListRequest":    {Required: []string{"space_id"}, Optional: []string{"type", "page", "page_size", "keyword"}},
		"MarketplaceProductsGetRequest":     {Required: []string{"space_id", "product_id"}, Optional: []string{}},
		"MarketplaceProductsInstallRequest": {Required: []string{"space_id", "product_id"}, Optional: []string{"version"}},
	}
}

func superAgentExternalAPICapabilities() []string {
	return []string{
		"sessions",
		"messages",
		"sandbox",
		"workspace",
		"skills",
		"products",
		"runtime_config",
		"harness",
		"artifacts",
		"approvals",
		"traces",
	}
}

func superAgentExternalAPIEntryRoutes() map[string]string {
	routes := map[string]string{
		"create": "POST /api/super-agent/runs/create",
		"stream": "POST /api/super-agent/runs/stream",
		"reply":  "POST /api/super-agent/runs/reply",
		"get":    "POST /api/super-agent/runs/get",
		"list":   "POST /api/super-agent/runs/list",
		"cancel": "POST /api/super-agent/runs/cancel",
	}
	for _, operation := range superAgentOpenAPIOperations() {
		routes[operation.OperationID] = operation.Method + " " + operation.Path
	}
	return routes
}

func superAgentExternalAPIRequestSchemas() map[string]superAgentRequestSchema {
	openAPISchemas := superAgentOpenAPIRequestSchemas()
	requestSchemas := map[string]superAgentRequestSchema{
		"create": openAPISchemas["RunsCreateRequest"],
		"stream": openAPISchemas["RunsStreamRequest"],
		"reply":  openAPISchemas["RunsReplyRequest"],
		"get":    openAPISchemas["RunsGetRequest"],
		"list":   openAPISchemas["RunsListRequest"],
		"cancel": openAPISchemas["RunsCancelRequest"],
	}
	for _, operation := range superAgentOpenAPIOperations() {
		requestSchemas[operation.OperationID] = openAPISchemas[operation.RequestSchema]
	}
	return requestSchemas
}

func buildSuperAgentOpenAPIDocument() map[string]interface{} {
	paths := map[string]interface{}{}
	schemas := map[string]interface{}{}
	requestSchemas := superAgentOpenAPIRequestSchemas()

	for _, operation := range superAgentOpenAPIOperations() {
		schemaName := operation.RequestSchema
		requestSchema := requestSchemas[schemaName]
		schemas[schemaName] = superAgentOpenAPIJSONSchema(requestSchema)

		method := strings.ToLower(operation.Method)
		pathItem, ok := paths[operation.Path].(map[string]interface{})
		if !ok {
			pathItem = map[string]interface{}{}
			paths[operation.Path] = pathItem
		}

		operationDoc := map[string]interface{}{
			"operationId": operation.OperationID,
			"tags":        []string{operation.Category},
			"summary":     operation.OperationID,
			"x-transport": operation.Transport,
			"x-mutates":   operation.Mutates,
			"responses": map[string]interface{}{
				"200": map[string]interface{}{"description": "Success"},
			},
		}
		if operation.Method == http.MethodGet {
			operationDoc["parameters"] = superAgentOpenAPIParameters(requestSchema)
		} else {
			operationDoc["requestBody"] = map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"$ref": "#/components/schemas/" + schemaName,
						},
					},
				},
			}
		}
		pathItem[method] = operationDoc
	}

	return map[string]interface{}{
		"openapi": superAgentOpenAPIVersion,
		"info": map[string]interface{}{
			"title":   superAgentOpenAPITitle,
			"version": superAgentProtocolVersion,
		},
		"servers": []map[string]string{{"url": "/api/super-agent"}},
		"security": []map[string][]string{
			{"BearerAuth": {}},
		},
		"paths": paths,
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"BearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "Bearer",
				},
			},
			"schemas": schemas,
		},
	}
}

func superAgentOpenAPIJSONSchema(schema superAgentRequestSchema) map[string]interface{} {
	properties := map[string]interface{}{}
	addProperty := func(name string) {
		if name == "" {
			return
		}
		if _, ok := properties[name]; ok {
			return
		}
		properties[name] = map[string]interface{}{
			"type": "string",
		}
	}

	for _, name := range schema.Required {
		addProperty(name)
	}
	for _, group := range schema.RequiredOneOf {
		for _, name := range group {
			addProperty(name)
		}
	}
	for _, name := range schema.Optional {
		addProperty(name)
	}
	for name, aliases := range schema.Aliases {
		addProperty(name)
		for _, alias := range aliases {
			addProperty(alias)
		}
	}

	return map[string]interface{}{
		"type":                 "object",
		"required":             schema.Required,
		"properties":           properties,
		"additionalProperties": true,
		"x-required-one-of":    schema.RequiredOneOf,
		"x-field-aliases":      schema.Aliases,
		"x-optional-fields":    schema.Optional,
	}
}

func superAgentOpenAPIParameters(schema superAgentRequestSchema) []map[string]interface{} {
	parameters := make([]map[string]interface{}, 0, len(schema.Required)+len(schema.Optional))
	addParameter := func(name string, required bool) {
		if name == "" {
			return
		}
		for _, parameter := range parameters {
			if parameter["name"] == name {
				return
			}
		}
		parameters = append(parameters, map[string]interface{}{
			"name":     name,
			"in":       "query",
			"required": required,
			"schema": map[string]interface{}{
				"type": "string",
			},
		})
	}

	for _, name := range schema.Required {
		addParameter(name, true)
	}
	for _, group := range schema.RequiredOneOf {
		for _, name := range group {
			addParameter(name, false)
		}
	}
	for _, name := range schema.Optional {
		addParameter(name, false)
	}

	return parameters
}

func buildSuperAgentHarnessContextPolicy() superAgentHarnessContextPolicy {
	return superAgentHarnessContextPolicy{
		Strategy:                   superAgentHarnessContextStrategy,
		SummaryVersion:             superAgentHarnessSummaryVersion,
		Trigger:                    superAgentHarnessCompactTrigger,
		MaxBytes:                   superAgentEnvInt("AGENT_CONTEXT_COMPACT_MAX_BYTES", superAgentHarnessDefaultCompactMaxBytes),
		RecentMessages:             superAgentEnvInt("AGENT_CONTEXT_COMPACT_RECENT_MESSAGES", superAgentHarnessDefaultRecentMessages),
		SummaryMaxRunes:            superAgentHarnessSummaryMaxRunes,
		SummaryPath:                superAgentHarnessContextPath,
		SessionSummaryPathTemplate: superAgentHarnessSessionSummaryPathTemplate,
		ClearRoute:                 superAgentHarnessContextClearRouteValue,
	}
}

func superAgentEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// SuperAgentOpenAPI exposes a machine-readable App Server schema for SDK and external callers.
// @router /api/super-agent/openapi.json [GET]
func SuperAgentOpenAPI(_ context.Context, c *app.RequestContext) {
	c.JSON(http.StatusOK, buildSuperAgentOpenAPIDocument())
}

// SuperAgentManifest exposes the stable App Server discovery contract.
// @router /api/super-agent/manifest [GET]
func SuperAgentManifest(_ context.Context, c *app.RequestContext) {
	c.JSON(http.StatusOK, &superAgentManifestResponse{
		Code: 0,
		Msg:  "success",
		Data: &superAgentManifestData{
			ProtocolVersion: superAgentProtocolVersion,
			Capabilities: []string{
				"sessions",
				"messages",
				"sandbox",
				"workspace",
				"skills",
				"products",
				"runtime_config",
				"harness",
				"artifacts",
				"approvals",
				"traces",
			},
			Auth: superAgentManifestAuth{
				Type:   "bearer",
				Header: "Authorization",
				Scheme: "Bearer",
			},
			Stream: superAgentStreamContract{
				Transport:   "sse",
				ContentType: "text/event-stream",
				Events: []string{
					string(agentrunEntity.RunEventAck),
					string(agentrunEntity.RunEventCreated),
					string(agentrunEntity.RunEventInProgress),
					string(agentrunEntity.RunEventMessageDelta),
					string(agentrunEntity.RunEventMessageCompleted),
					string(agentrunEntity.RunEventCompleted),
					string(agentrunEntity.RunEventFailed),
					string(agentrunEntity.RunEventCancelled),
					string(agentrunEntity.RunEventStreamDone),
					string(agentrunEntity.RunEventError),
				},
				DoneEvent:  string(agentrunEntity.RunEventStreamDone),
				ErrorEvent: string(agentrunEntity.RunEventError),
			},
			Sessions: superAgentSessionContract{
				ListRoute:      "POST /api/super-agent/sessions/list",
				CreateRoute:    "POST /api/super-agent/sessions/create",
				GetRoute:       "POST /api/super-agent/sessions/get",
				RenameRoute:    "POST /api/super-agent/sessions/rename",
				DeleteRoute:    "POST /api/super-agent/sessions/delete",
				TitleExtKey:    superAgentSessionTitleExtKey,
				MaxTitleLength: superAgentSessionMaxTitleLen,
				RequestSchemas: map[string]superAgentRequestSchema{
					"create": {
						Required:      []string{},
						RequiredOneOf: [][]string{{"agent_id", "bot_id"}},
						Optional:      []string{"space_id", "connector_id", "title", "user_id", "client_id"},
					},
					"get": {
						Required: []string{"conversation_id"},
						Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "include_snapshot", "run_id", "trace_limit", "run_limit", "message_limit", "artifact_limit", "workspace_path", "workspace_recursive", "include_harness", "include_messages", "include_runs", "include_workspace", "include_context", "include_tool_outputs", "include_tool_output_content", "include_artifacts", "include_trace", "include_approvals", "user_id", "client_id"},
					},
					"list": {
						Required:      []string{},
						RequiredOneOf: [][]string{{"agent_id", "bot_id"}},
						Optional:      []string{"space_id", "connector_id", "page", "page_size", "user_id", "client_id"},
					},
					"rename": {
						Required: []string{"conversation_id", "title"},
						Optional: []string{"user_id", "client_id"},
					},
					"delete": {
						Required: []string{"conversation_id"},
						Optional: []string{"user_id", "client_id"},
					},
				},
			},
			RuntimeConfig: superAgentRuntimeConfigContract{
				GetRoute:    "POST /api/super-agent/runtime-config/get",
				UpdateRoute: "POST /api/super-agent/runtime-config/update",
				DeleteRoute: "POST /api/super-agent/runtime-config/delete",
				BindableTypes: []string{
					string(productEntity.AIProductTypeModel),
					string(productEntity.AIProductTypeMCPServer),
					string(productEntity.AIProductTypeStandardSkill),
				},
				SnapshotFields: []string{
					"conversation_id",
					"agent_id",
					"space_id",
					"model",
					"mcp_servers",
					"skills",
					"tool_policy",
					"context_policy",
				},
				RequestSchemas: map[string]superAgentRequestSchema{
					"get": {
						Required: []string{"conversation_id"},
						Optional: []string{"user_id", "client_id"},
					},
					"update": {
						Required: []string{"conversation_id", "space_id"},
						Optional: []string{"agent_id", "bot_id", "model_product_id", "mcp_product_ids", "skill_product_ids", "tool_policy", "context_policy", "user_id", "client_id"},
					},
					"delete": {
						Required: []string{"conversation_id"},
						Optional: []string{"user_id", "client_id"},
					},
				},
			},
			Messages: superAgentMessageContract{
				ListRoute:   "POST /api/super-agent/messages/list",
				MaxPageSize: superAgentMessageMaxPageSize,
				OrderValues: []string{messageModel.OrderByAsc, messageModel.OrderByDesc},
				RequestSchemas: map[string]superAgentRequestSchema{
					"list": {
						Required: []string{"conversation_id"},
						Optional: []string{"limit", "before_id", "after_id", "order_by", "user_id", "client_id"},
					},
				},
			},
			Trace: superAgentTraceContract{
				Events: []string{
					string(agentrunEntity.RunEventCreated),
					string(agentrunEntity.RunEventInProgress),
					string(agentrunEntity.RunEventCompleted),
					string(agentrunEntity.RunEventFailed),
					string(agentrunEntity.RunEventCancelled),
					string(agentrunEntity.RunEventRequiredAction),
					string(agentrunEntity.RunEventAck),
					string(agentrunEntity.RunEventMessageCompleted),
					superagenttrace.EventToolStarted,
					superagenttrace.EventToolCompleted,
					superagenttrace.EventToolFailed,
					superagenttrace.EventPlanUpdated,
					superagenttrace.EventContextCompacted,
				},
				MaxPageSize: superAgentTraceMaxPageSize,
			},
			Approvals: superAgentApprovalContract{
				ListRoute:      "POST /api/super-agent/approvals/list",
				ResolveRoute:   "POST /api/super-agent/approvals/resolve",
				DecisionRoot:   superAgentApprovalDecisionRoot,
				DecisionFormat: "json",
				StatusValues:   []string{"pending", "approved", "rejected", "cancelled"},
				DecisionValues: []string{"approve", "reject", "cancel"},
				ResponseFields: []string{"approval_id", "run_id", "decision", "status", "resolved", "cancelled", "note", "decision_path", "persisted"},
				RequestSchemas: map[string]superAgentRequestSchema{
					"list": {
						Required: []string{"conversation_id"},
						Optional: []string{"space_id", "agent_id", "bot_id", "run_id", "status", "limit", "user_id", "client_id"},
					},
					"resolve": {
						Required: []string{"approval_id", "decision"},
						Optional: []string{"space_id", "agent_id", "bot_id", "conversation_id", "run_id", "note", "user_id", "client_id"},
					},
				},
			},
			Artifacts: superAgentArtifactContract{
				Root:          "/outputs",
				ListRoute:     "POST /api/super-agent/artifacts/list",
				DownloadRoute: "POST /api/super-agent/artifacts/download",
				DeleteRoute:   "POST /api/super-agent/artifacts/delete",
				MoveRoute:     "POST /api/super-agent/artifacts/move",
				MetadataFields: []string{
					"artifact_id",
					"name",
					"path",
					"kind",
					"preview_type",
					"summary",
					"size",
					"mtime",
					"mime",
					"sha256",
					"previewable",
					"downloadable",
					"download_route",
				},
				PreviewableMIMEs: []string{
					"text/plain",
					"text/markdown",
					"text/html",
					"text/csv",
					"application/json",
					"application/pdf",
					"image/png",
					"image/jpeg",
					"image/webp",
				},
				MaxListItems: 200,
				RequestSchemas: map[string]superAgentRequestSchema{
					"list": {
						Required: []string{},
						Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "path", "limit"},
					},
					"download": {
						Required:      []string{},
						RequiredOneOf: [][]string{{"artifact_id", "path"}},
						Optional:      []string{"space_id", "agent_id", "bot_id", "connector_id"},
					},
					"delete": {
						Required:      []string{},
						RequiredOneOf: [][]string{{"artifact_id", "path"}},
						Optional:      []string{"space_id", "agent_id", "bot_id", "connector_id"},
					},
					"move": {
						Required:      []string{"target_path"},
						RequiredOneOf: [][]string{{"artifact_id", "path"}},
						Optional:      []string{"space_id", "agent_id", "bot_id", "connector_id"},
					},
				},
			},
			Products: superAgentProductContract{
				ProductTypes: []string{
					string(productEntity.AIProductTypeModel),
					string(productEntity.AIProductTypeMCPServer),
					string(productEntity.AIProductTypeStandardSkill),
					string(productEntity.AIProductTypeAgentApp),
				},
				VisibilityValues: []string{
					string(productEntity.AIProductVisibilityPrivate),
					string(productEntity.AIProductVisibilitySpace),
					string(productEntity.AIProductVisibilityGlobal),
				},
				StatusValues: []string{
					string(productEntity.AIProductStatusDraft),
					string(productEntity.AIProductStatusReviewing),
					string(productEntity.AIProductStatusPublished),
					string(productEntity.AIProductStatusDeprecated),
					string(productEntity.AIProductStatusArchived),
				},
				SourceRefTypes: []string{
					productEntity.SourceRefTypeSkill,
				},
				ListRoute:               "POST /api/super-agent/products/list",
				GetRoute:                "POST /api/super-agent/products/get",
				InstallRoute:            "POST /api/super-agent/products/install",
				UpgradeRoute:            "POST /api/super-agent/products/upgrade",
				UninstallRoute:          "POST /api/super-agent/products/uninstall",
				MarketplaceListRoute:    "POST /api/super-agent/marketplace/products/list",
				MarketplaceGetRoute:     "POST /api/super-agent/marketplace/products/get",
				MarketplaceInstallRoute: "POST /api/super-agent/marketplace/products/install",
				RequestSchemas: map[string]superAgentRequestSchema{
					"list": {
						Required: []string{"space_id"},
						Optional: []string{"type", "page", "page_size", "keyword"},
					},
					"get": {
						Required: []string{"space_id", "product_id"},
						Optional: []string{},
					},
					"install": {
						Required: []string{"space_id", "product_id"},
						Optional: []string{"version"},
					},
					"upgrade": {
						Required: []string{"space_id", "product_id"},
						Optional: []string{},
					},
					"uninstall": {
						Required: []string{"space_id", "product_id"},
						Optional: []string{},
					},
					"marketplace.list": {
						Required: []string{"space_id"},
						Optional: []string{"type", "page", "page_size", "keyword"},
					},
					"marketplace.get": {
						Required: []string{"space_id", "product_id"},
						Optional: []string{},
					},
					"marketplace.install": {
						Required: []string{"space_id", "product_id"},
						Optional: []string{"version"},
					},
				},
			},
			Skills: superAgentSkillContract{
				EntryFile: "SKILL.md",
				FileRoots: []string{
					"SKILL.md",
					"scripts/",
					"references/",
					"templates/",
					"assets/",
				},
				PublishScopes: map[string]int8{
					"private": skillEntity.SkillPublishScopePrivate,
					"space":   skillEntity.SkillPublishScopeSpace,
					"global":  skillEntity.SkillPublishScopeGlobal,
				},
				Routes: map[string]string{
					"create":              "POST /api/super-agent/skills/create",
					"validate_package":    "POST /api/super-agent/skills/validate-package",
					"import":              "POST /api/super-agent/skills/import",
					"import_runtime":      "POST /api/super-agent/skills/import-runtime",
					"export":              "POST /api/super-agent/skills/export",
					"get":                 "GET /api/super-agent/skills/get",
					"update":              "POST /api/super-agent/skills/update",
					"delete":              "POST /api/super-agent/skills/delete",
					"publish":             "POST /api/super-agent/skills/publish",
					"list":                "GET /api/super-agent/skills/list",
					"assets.list":         "GET /api/super-agent/skills/assets/list",
					"assets.get":          "GET /api/super-agent/skills/assets/get",
					"assets.upsert":       "POST /api/super-agent/skills/assets/upsert",
					"assets.delete":       "POST /api/super-agent/skills/assets/delete",
					"marketplace.list":    "GET /api/super-agent/marketplace/list",
					"marketplace.get":     "GET /api/super-agent/marketplace/get",
					"marketplace.install": "POST /api/super-agent/marketplace/install",
				},
				RequestSchemas: map[string]superAgentRequestSchema{
					"create": {
						Required: []string{"space_id", "name", "files"},
						Optional: []string{"description", "prompt", "icon_uri"},
					},
					"validate_package": {
						Required: []string{"content"},
						Optional: []string{"filename"},
					},
					"import": {
						Required: []string{"space_id", "content"},
						Optional: []string{"filename", "icon_uri"},
					},
					"import_runtime": {
						Required: []string{"agent_id", "name"},
						Optional: []string{"bot_id", "connector_id", "skill_id", "icon_uri", "publish_scope"},
					},
					"export": {
						Required: []string{"space_id", "skill_id"},
						Optional: []string{},
					},
					"get": {
						Required: []string{"space_id", "skill_id"},
						Optional: []string{},
					},
					"update": {
						Required: []string{"space_id", "skill_id"},
						Optional: []string{"name", "description", "prompt", "icon_uri", "files"},
					},
					"delete": {
						Required: []string{"space_id", "skill_id"},
						Optional: []string{},
					},
					"publish": {
						Required: []string{"space_id", "skill_id", "scope"},
						Optional: []string{},
					},
					"list": {
						Required: []string{"space_id"},
						Optional: []string{"page", "page_size", "keyword"},
					},
					"marketplace.list": {
						Required: []string{"space_id"},
						Optional: []string{"scope", "page", "page_size", "keyword"},
					},
					"marketplace.get": {
						Required: []string{"space_id", "skill_id"},
						Optional: []string{},
					},
					"marketplace.install": {
						Required: []string{"space_id", "skill_id"},
						Optional: []string{},
					},
					"assets.list": {
						Required: []string{"space_id", "skill_id"},
						Optional: []string{},
					},
					"assets.get": {
						Required: []string{"space_id", "skill_id", "path"},
						Optional: []string{},
					},
					"assets.upsert": {
						Required: []string{"space_id", "skill_id", "path", "content"},
						Optional: []string{"mime"},
					},
					"assets.delete": {
						Required: []string{"space_id", "skill_id", "path"},
						Optional: []string{},
					},
				},
				Assets: superAgentSkillAssetContract{
					Root:        "assets/",
					ListRoute:   "GET /api/super-agent/skills/assets/list",
					GetRoute:    "GET /api/super-agent/skills/assets/get",
					UpsertRoute: "POST /api/super-agent/skills/assets/upsert",
					DeleteRoute: "POST /api/super-agent/skills/assets/delete",
					AllowedMIMEs: []string{
						"image/png",
						"image/jpeg",
						"image/webp",
						"image/gif",
						"image/svg+xml",
						"application/json",
						"text/plain",
						"text/markdown",
					},
					ContentEncoding: "utf8-or-data-url",
				},
				AgentTool: superAgentSkillToolContract{
					Name: "skill_manage",
					Actions: []string{
						"create",
						"list",
						"read",
						"diff",
						"write_file",
						"edit",
						"remove_file",
						"delete",
					},
				},
			},
			SkillPublishScopes: map[string]int8{
				"private": skillEntity.SkillPublishScopePrivate,
				"space":   skillEntity.SkillPublishScopeSpace,
				"global":  skillEntity.SkillPublishScopeGlobal,
			},
			WorkspaceRoots: []superAgentWorkspaceRoot{
				{Path: "/workspace", Label: "工作区", Readonly: false},
				{Path: "/outputs", Label: "产出物", Readonly: false},
				{Path: "/uploads", Label: "上传区", Readonly: false},
				{Path: "/skills", Label: "技能", Readonly: true},
			},
			Workspace: superAgentWorkspaceContract{
				IdentifierFields: []string{"agent_id", "bot_id"},
				ReadableRoots:    []string{"/workspace", "/outputs", "/uploads", "/skills"},
				WritableRoots:    []string{"/workspace", "/outputs", "/uploads"},
				ReadMaxBytes:     superAgentWorkspaceMaxReadBytes,
				BinaryEncoding:   "base64",
				WriteRoute:       "POST /api/super-agent/workspace/write",
				MoveRoute:        "POST /api/super-agent/workspace/move",
				MkdirRoute:       "POST /api/super-agent/workspace/mkdir",
				StatRoute:        "POST /api/super-agent/workspace/stat",
				GrepRoute:        "POST /api/super-agent/workspace/grep",
				GlobRoute:        "POST /api/super-agent/workspace/glob",
				EditRoute:        "POST /api/super-agent/workspace/edit",
				PatchRoute:       "POST /api/super-agent/workspace/patch",
				RequestSchemas: map[string]superAgentRequestSchema{
					"patch": {
						Required: []string{"patch"},
						Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "workdir", "work_dir"},
						Aliases: map[string][]string{
							"workdir": {"work_dir"},
						},
					},
				},
			},
			Sandbox: superAgentSandboxContract{
				ExecRoute:      "POST /api/super-agent/sandbox/exec",
				DefaultWorkdir: "/workspace",
				MaxTimeoutSec:  superAgentSandboxMaxTimeoutSec,
				OutputMaxBytes: superAgentSandboxOutputMaxBytes,
				RequestSchemas: map[string]superAgentRequestSchema{
					"exec": {
						Required: []string{"command"},
						Optional: []string{"space_id", "agent_id", "bot_id", "connector_id", "workdir", "work_dir", "timeout_sec"},
						Aliases: map[string][]string{
							"workdir": {"work_dir"},
						},
					},
				},
			},
			Harness: superAgentHarnessContract{
				DeliverableRoot:         "/outputs",
				PlanPath:                "/workspace/.plan.json",
				SessionPlanPathTemplate: "/workspace/.agent/sessions/{conversation_id}/plan.json",
				ToolOutputRoot:          "/workspace/.agent/tooloutputs",
				StateRoute:              "POST /api/super-agent/harness/state",
				StateFields:             []string{"plan", "tool_outputs", "runtime_skills", "context"},
				PlanUpdateRoute:         "POST /api/super-agent/harness/plan",
				PlanRoute:               "POST /api/super-agent/workspace/read",
				ToolOutputsRoute:        "POST /api/super-agent/harness/tool-outputs",
				ToolOutputFields:        []string{"path", "name", "tool_call_id", "tool", "status", "arguments", "result", "error", "arguments_preview", "result_preview", "summary", "size", "mtime"},
				CleanupRoute:            "POST /api/super-agent/harness/cleanup",
				ContextClearRoute:       superAgentHarnessContextClearRouteValue,
				ContextPolicy:           buildSuperAgentHarnessContextPolicy(),
				SnapshotRoute:           "POST /api/super-agent/harness/snapshot",
				SnapshotFields:          []string{"messages", "runs", "workspace", "harness", "context", "tool_outputs", "artifacts", "trace", "approvals", "approval_decisions", "resume"},
				ResumeRoute:             "POST /api/super-agent/harness/resume",
				ResumeFields:            []string{"version", "conversation_id", "agent_id", "summary", "summary_path", "summary_exists", "plan_path", "plan_exists", "plan_status", "plan_item_count", "recent_message_count", "message_ids", "tool_output_root", "tool_output_count", "tool_output_paths", "artifact_root", "artifact_count", "artifact_paths", "trace_event_count", "approval_count", "components", "prompt"},
				SkillRuntimeRoot:        "/skills",
				SkillEntryFile:          "SKILL.md",
				SkillFileRoots:          []string{"SKILL.md", "scripts/", "references/", "templates/", "assets/"},
				Tools: []superAgentHarnessTool{
					{Name: "run_bash", Category: "sandbox", Available: "super_agent", Mutates: true},
					{Name: "read_file", Category: "sandbox", Available: "super_agent", Mutates: false},
					{Name: "write_file", Category: "sandbox", Available: "super_agent", Mutates: true},
					{Name: "edit_file", Category: "sandbox", Available: "super_agent", Mutates: true},
					{Name: "apply_patch", Category: "sandbox", Available: "super_agent", Mutates: true},
					{Name: "list_files", Category: "sandbox", Available: "super_agent", Mutates: false},
					{Name: "grep", Category: "sandbox", Available: "super_agent", Mutates: false},
					{Name: "glob", Category: "sandbox", Available: "super_agent", Mutates: false},
					{Name: "update_plan", Category: "planning", Available: "super_agent", Mutates: true},
					{Name: "deep_task", Category: "delegation", Available: "super_agent", Mutates: true},
					{Name: "read_skill", Category: "skills", Available: "when_agent_has_bound_skills", Mutates: false},
					{Name: "skill_manage", Category: "skills", Available: "super_agent", Mutates: true},
					{Name: "memory_recall", Category: "memory", Available: "super_agent", Mutates: false},
					{Name: "memory_save", Category: "memory", Available: "super_agent", Mutates: true},
					{Name: "web_search", Category: "web", Available: "super_agent", Mutates: false},
					{Name: "web_fetch", Category: "web", Available: "super_agent", Mutates: false},
				},
			},
			ExternalAPI: superAgentExternalAPIContract{
				BasePath:        "/api/super-agent",
				ProtocolVersion: superAgentProtocolVersion,
				Auth: superAgentManifestAuth{
					Type:   "bearer",
					Header: "Authorization",
					Scheme: "Bearer",
				},
				SchemaRoute:          superAgentOpenAPISchemaRoute,
				SessionAuthSupported: true,
				Transports:           []string{"json", "sse"},
				IdentifierFields:     []string{"agent_id", "bot_id"},
				Capabilities:         superAgentExternalAPICapabilities(),
				EntryRoutes:          superAgentExternalAPIEntryRoutes(),
				RequestSchemas:       superAgentExternalAPIRequestSchemas(),
				ClientMetadata: map[string]string{
					"app_server_flag":    superagentapp.AppServerFlag,
					"capabilities_param": superagentapp.CapabilitiesParam,
					"capabilities_value": superagentapp.CapabilitiesParamText,
					"transport_param":    superagentapp.TransportParam,
					"client_id_param":    superagentapp.ClientIDParam,
				},
			},
			OpenAPI: buildSuperAgentOpenAPIContract(),
			Routes: map[string]string{
				"manifest":                     "GET /api/super-agent/manifest",
				"openapi":                      superAgentOpenAPISchemaRoute,
				"runs.create":                  "POST /api/super-agent/runs/create",
				"runs.get":                     "POST /api/super-agent/runs/get",
				"runs.list":                    "POST /api/super-agent/runs/list",
				"runs.reply":                   "POST /api/super-agent/runs/reply",
				"runs.stream":                  "POST /api/super-agent/runs/stream",
				"runs.cancel":                  "POST /api/super-agent/runs/cancel",
				"sessions.create":              "POST /api/super-agent/sessions/create",
				"sessions.get":                 "POST /api/super-agent/sessions/get",
				"sessions.list":                "POST /api/super-agent/sessions/list",
				"sessions.rename":              "POST /api/super-agent/sessions/rename",
				"sessions.delete":              "POST /api/super-agent/sessions/delete",
				"runtime_config.get":           "POST /api/super-agent/runtime-config/get",
				"runtime_config.update":        "POST /api/super-agent/runtime-config/update",
				"runtime_config.delete":        "POST /api/super-agent/runtime-config/delete",
				"messages.list":                "POST /api/super-agent/messages/list",
				"approvals.list":               "POST /api/super-agent/approvals/list",
				"approvals.resolve":            "POST /api/super-agent/approvals/resolve",
				"harness.state":                "POST /api/super-agent/harness/state",
				"harness.plan":                 "POST /api/super-agent/harness/plan",
				"harness.tool_outputs":         "POST /api/super-agent/harness/tool-outputs",
				"harness.cleanup":              "POST /api/super-agent/harness/cleanup",
				"harness.resume":               "POST /api/super-agent/harness/resume",
				"harness.snapshot":             "POST /api/super-agent/harness/snapshot",
				"workspace.list":               "POST /api/super-agent/workspace/list",
				"workspace.read":               "POST /api/super-agent/workspace/read",
				"workspace.upload":             "POST /api/super-agent/workspace/upload",
				"workspace.write":              "POST /api/super-agent/workspace/write",
				"workspace.download":           "POST /api/super-agent/workspace/download",
				"workspace.delete":             "POST /api/super-agent/workspace/delete",
				"workspace.move":               "POST /api/super-agent/workspace/move",
				"workspace.mkdir":              "POST /api/super-agent/workspace/mkdir",
				"workspace.stat":               "POST /api/super-agent/workspace/stat",
				"workspace.grep":               "POST /api/super-agent/workspace/grep",
				"workspace.glob":               "POST /api/super-agent/workspace/glob",
				"workspace.edit":               "POST /api/super-agent/workspace/edit",
				"workspace.patch":              "POST /api/super-agent/workspace/patch",
				"sandbox.exec":                 "POST /api/super-agent/sandbox/exec",
				"traces.get":                   "POST /api/super-agent/traces/get",
				"artifacts.list":               "POST /api/super-agent/artifacts/list",
				"artifacts.download":           "POST /api/super-agent/artifacts/download",
				"artifacts.delete":             "POST /api/super-agent/artifacts/delete",
				"artifacts.move":               "POST /api/super-agent/artifacts/move",
				"skills.create":                "POST /api/super-agent/skills/create",
				"skills.validate_package":      "POST /api/super-agent/skills/validate-package",
				"skills.import":                "POST /api/super-agent/skills/import",
				"skills.import_runtime":        "POST /api/super-agent/skills/import-runtime",
				"skills.export":                "POST /api/super-agent/skills/export",
				"skills.get":                   "GET /api/super-agent/skills/get",
				"skills.update":                "POST /api/super-agent/skills/update",
				"skills.delete":                "POST /api/super-agent/skills/delete",
				"skills.publish":               "POST /api/super-agent/skills/publish",
				"skills.list":                  "GET /api/super-agent/skills/list",
				"skills.assets.list":           "GET /api/super-agent/skills/assets/list",
				"skills.assets.get":            "GET /api/super-agent/skills/assets/get",
				"skills.assets.upsert":         "POST /api/super-agent/skills/assets/upsert",
				"skills.assets.delete":         "POST /api/super-agent/skills/assets/delete",
				"skills.marketplace.list":      "GET /api/super-agent/marketplace/list",
				"skills.marketplace.get":       "GET /api/super-agent/marketplace/get",
				"skills.marketplace.install":   "POST /api/super-agent/marketplace/install",
				"products.list":                "POST /api/super-agent/products/list",
				"products.get":                 "POST /api/super-agent/products/get",
				"products.install":             "POST /api/super-agent/products/install",
				"products.upgrade":             "POST /api/super-agent/products/upgrade",
				"products.uninstall":           "POST /api/super-agent/products/uninstall",
				"marketplace.products.list":    "POST /api/super-agent/marketplace/products/list",
				"marketplace.products.get":     "POST /api/super-agent/marketplace/products/get",
				"marketplace.products.install": "POST /api/super-agent/marketplace/products/install",
			},
		},
	})
}

type superAgentRunRequest struct {
	SpaceID            int64                      `form:"space_id" json:"space_id,string,omitempty"`
	AgentID            int64                      `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID              int64                      `form:"bot_id" json:"bot_id,string,omitempty"`
	ConversationID     *int64                     `json:"conversation_id,string,omitempty" query:"conversation_id"`
	User               string                     `form:"user_id" json:"user_id,omitempty"`
	AdditionalMessages []*run.EnterMessage        `form:"additional_messages" json:"additional_messages,omitempty"`
	CustomVariables    map[string]string          `form:"custom_variables" json:"custom_variables,omitempty"`
	MetaData           map[string]string          `form:"meta_data" json:"meta_data,omitempty"`
	CustomConfig       *run.CustomConfig          `form:"custom_config" json:"custom_config,omitempty"`
	ExtraParams        map[string]string          `form:"extra_params" json:"extra_params,omitempty"`
	ConnectorID        *int64                     `form:"connector_id" json:"connector_id,string,omitempty"`
	ShortcutCommand    *run.ShortcutCommandDetail `form:"shortcut_command" json:"shortcut_command,omitempty"`
	ClientID           string                     `form:"client_id" json:"client_id,omitempty"`
}

type superAgentCancelRunRequest struct {
	RunID    string `form:"run_id" json:"run_id,omitempty"`
	SpaceID  int64  `form:"space_id" json:"space_id,string,omitempty"`
	AgentID  int64  `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID    int64  `form:"bot_id" json:"bot_id,string,omitempty"`
	User     string `form:"user_id" json:"user_id,omitempty"`
	ClientID string `form:"client_id" json:"client_id,omitempty"`
}

type superAgentGetRunRequest struct {
	RunID    string `form:"run_id" json:"run_id,omitempty"`
	SpaceID  int64  `form:"space_id" json:"space_id,string,omitempty"`
	AgentID  int64  `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID    int64  `form:"bot_id" json:"bot_id,string,omitempty"`
	User     string `form:"user_id" json:"user_id,omitempty"`
	ClientID string `form:"client_id" json:"client_id,omitempty"`
}

type superAgentListRunsRequest struct {
	ConversationID int64  `form:"conversation_id" json:"conversation_id,string,omitempty"`
	SpaceID        int64  `form:"space_id" json:"space_id,string,omitempty"`
	AgentID        int64  `form:"agent_id" json:"agent_id,string,omitempty"`
	BotID          int64  `form:"bot_id" json:"bot_id,string,omitempty"`
	User           string `form:"user_id" json:"user_id,omitempty"`
	ClientID       string `form:"client_id" json:"client_id,omitempty"`
	Limit          int32  `form:"limit" json:"limit,omitempty"`
	OrderBy        string `form:"order_by" json:"order_by,omitempty"`
	BeforeID       int64  `form:"before_id" json:"before_id,string,omitempty"`
	AfterID        int64  `form:"after_id" json:"after_id,string,omitempty"`
}

type superAgentCancelRunResponse struct {
	Code int                     `json:"code"`
	Msg  string                  `json:"msg"`
	Data superAgentCancelRunData `json:"data"`
}

type superAgentCancelRunData struct {
	RunID     string `json:"run_id"`
	Status    string `json:"status"`
	Cancelled bool   `json:"cancelled"`
}

type superAgentGetRunResponse struct {
	Code int                  `json:"code"`
	Msg  string               `json:"msg"`
	Data superAgentGetRunData `json:"data"`
}

type superAgentGetRunData struct {
	RunID          string                   `json:"run_id"`
	ConversationID string                   `json:"conversation_id,omitempty"`
	AgentID        string                   `json:"agent_id,omitempty"`
	Status         string                   `json:"status"`
	Active         bool                     `json:"active"`
	Error          *agentrunEntity.RunError `json:"error,omitempty"`
	CreatedAt      int64                    `json:"created_at,omitempty"`
	UpdatedAt      int64                    `json:"updated_at,omitempty"`
	CompletedAt    int64                    `json:"completed_at,omitempty"`
	FailedAt       int64                    `json:"failed_at,omitempty"`
}

type superAgentListRunsResponse struct {
	Code int                    `json:"code"`
	Msg  string                 `json:"msg"`
	Data *superAgentRunListData `json:"data"`
}

type superAgentRunListData struct {
	ConversationID string                  `json:"conversation_id"`
	Runs           []superAgentRunListItem `json:"runs"`
}

type superAgentRunListItem struct {
	RunID          string                   `json:"run_id"`
	ConversationID string                   `json:"conversation_id"`
	AgentID        string                   `json:"agent_id"`
	Status         string                   `json:"status"`
	Active         bool                     `json:"active"`
	Error          *agentrunEntity.RunError `json:"error,omitempty"`
	CreatedAt      int64                    `json:"created_at"`
	UpdatedAt      int64                    `json:"updated_at"`
	CompletedAt    int64                    `json:"completed_at,omitempty"`
	FailedAt       int64                    `json:"failed_at,omitempty"`
}

// SuperAgentCreateRun exposes a JSON App Server entrypoint for a new super-agent run.
// @router /api/super-agent/runs/create [POST]
func SuperAgentCreateRun(ctx context.Context, c *app.RequestContext) {
	superAgentRun(ctx, c, false, false)
}

// SuperAgentGetRun returns the active App Server super-agent run state.
// @router /api/super-agent/runs/get [POST]
func SuperAgentGetRun(ctx context.Context, c *app.RequestContext) {
	var req superAgentGetRunRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		invalidParamRequestResponse(c, "run_id is required")
		return
	}
	// IDOR guard: resolve the run's owning conversation and enforce the same
	// per-conversation owner gate used by runs/get-history and runs/list before
	// returning any run status/agent_id/conversation_id.
	runRecord, authErr := authorizeSuperAgentRunOwner(ctx, runID)
	if authErr != nil {
		internalServerErrorResponse(ctx, c, authErr)
		return
	}
	data := superAgentGetRunData{
		RunID:  runID,
		Status: "not_active",
		Active: false,
	}
	if state, ok := conversation.GetActiveAgentRun(runID); ok {
		data.Status = state.Status
		data.Active = true
		data.CreatedAt = state.CreatedAt
		data.UpdatedAt = state.UpdatedAt
	} else if runRecord != nil {
		data = buildSuperAgentGetRunData(runRecord)
	}
	c.JSON(http.StatusOK, &superAgentGetRunResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

func buildSuperAgentGetRunData(runRecord *agentrunEntity.RunRecordMeta) superAgentGetRunData {
	if runRecord == nil {
		return superAgentGetRunData{}
	}
	return superAgentGetRunData{
		RunID:          strconv.FormatInt(runRecord.ID, 10),
		ConversationID: strconv.FormatInt(runRecord.ConversationID, 10),
		AgentID:        strconv.FormatInt(runRecord.AgentID, 10),
		Status:         string(runRecord.Status),
		Active:         false,
		Error:          runRecord.Error,
		CreatedAt:      runRecord.CreatedAt,
		UpdatedAt:      runRecord.UpdatedAt,
		CompletedAt:    runRecord.CompletedAt,
		FailedAt:       runRecord.FailedAt,
	}
}

// authorizeSuperAgentRunOwner resolves a run id string to its run record and
// owning conversation, then enforces the per-conversation owner gate
// (checkSuperAgentTracePermission). It mirrors how runs/get and runs/list
// resolve ownership so the caller is guaranteed to be the conversation creator
// before a run can be read, cancelled, or have an approval decision applied.
//
// Returns the resolved run record (which may be nil when the run id has no
// persisted record yet, e.g. an in-memory-only active run) together with an
// error when the caller is not the owner. A nil error means access is granted.
func authorizeSuperAgentRunOwner(ctx context.Context, runID string) (*agentrunEntity.RunRecordMeta, error) {
	if conversation.ConversationSVC.AgentRunDomainSVC == nil || conversation.ConversationSVC.ConversationDomainSVC == nil {
		return nil, nil
	}
	runRecordID, parseErr := strconv.ParseInt(strings.TrimSpace(runID), 10, 64)
	if parseErr != nil || runRecordID <= 0 {
		// No resolvable run record (e.g. a synthetic/in-memory run id) — nothing
		// to authorize against, leave the decision to the caller's other guards.
		return nil, nil
	}
	runRecord, err := conversation.ConversationSVC.AgentRunDomainSVC.GetByID(ctx, runRecordID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if runRecord == nil {
		return nil, nil
	}
	currentConversation, err := conversation.ConversationSVC.ConversationDomainSVC.GetByID(ctx, runRecord.ConversationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return runRecord, errorx.New(errno.ErrConversationNotFound)
		}
		return nil, err
	}
	if currentConversation == nil {
		return runRecord, errorx.New(errno.ErrConversationNotFound)
	}
	if err := checkSuperAgentTracePermission(ctx, currentConversation.CreatorID); err != nil {
		return runRecord, err
	}
	return runRecord, nil
}

// SuperAgentListRuns returns lightweight App Server run history for a conversation.
// @router /api/super-agent/runs/list [POST]
func SuperAgentListRuns(ctx context.Context, c *app.RequestContext) {
	var req superAgentListRunsRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.ConversationID <= 0 {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}
	data, err := buildSuperAgentRunHistory(ctx, req)
	if err != nil {
		internalServerErrorResponse(ctx, c, err)
		return
	}
	c.JSON(http.StatusOK, &superAgentListRunsResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

func buildSuperAgentRunHistory(ctx context.Context, req superAgentListRunsRequest) (*superAgentRunListData, error) {
	currentConversation, err := conversation.ConversationSVC.ConversationDomainSVC.GetByID(ctx, req.ConversationID)
	if err != nil {
		return nil, err
	}
	if currentConversation == nil {
		return nil, errorx.New(errno.ErrConversationNotFound)
	}
	if err := checkSuperAgentTracePermission(ctx, currentConversation.CreatorID); err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 || limit > superAgentTraceMaxPageSize {
		limit = superAgentTraceMaxPageSize
	}
	orderBy := strings.ToLower(strings.TrimSpace(req.OrderBy))
	if orderBy != "asc" {
		orderBy = "desc"
	}
	runRecords, err := conversation.ConversationSVC.AgentRunDomainSVC.List(ctx, &agentrunEntity.ListRunRecordMeta{
		ConversationID: req.ConversationID,
		Limit:          limit,
		OrderBy:        orderBy,
		BeforeID:       req.BeforeID,
		AfterID:        req.AfterID,
	})
	if err != nil {
		return nil, err
	}
	return buildSuperAgentRunListData(req.ConversationID, runRecords), nil
}

func buildSuperAgentRunListData(conversationID int64, runRecords []*agentrunEntity.RunRecordMeta) *superAgentRunListData {
	data := &superAgentRunListData{
		ConversationID: strconv.FormatInt(conversationID, 10),
		Runs:           make([]superAgentRunListItem, 0, len(runRecords)),
	}
	for _, runRecord := range runRecords {
		if runRecord == nil {
			continue
		}
		data.Runs = append(data.Runs, buildSuperAgentRunListItem(runRecord))
	}
	return data
}

func buildSuperAgentRunListItem(runRecord *agentrunEntity.RunRecordMeta) superAgentRunListItem {
	runID := strconv.FormatInt(runRecord.ID, 10)
	status := string(runRecord.Status)
	active := false
	updatedAt := runRecord.UpdatedAt
	if state, ok := conversation.GetActiveAgentRun(runID); ok {
		status = state.Status
		active = true
		if state.UpdatedAt > updatedAt {
			updatedAt = state.UpdatedAt
		}
	}
	return superAgentRunListItem{
		RunID:          runID,
		ConversationID: strconv.FormatInt(runRecord.ConversationID, 10),
		AgentID:        strconv.FormatInt(runRecord.AgentID, 10),
		Status:         status,
		Active:         active,
		Error:          runRecord.Error,
		CreatedAt:      runRecord.CreatedAt,
		UpdatedAt:      updatedAt,
		CompletedAt:    runRecord.CompletedAt,
		FailedAt:       runRecord.FailedAt,
	}
}

// SuperAgentReplyRun exposes a JSON App Server entrypoint for continuing a run conversation.
// @router /api/super-agent/runs/reply [POST]
func SuperAgentReplyRun(ctx context.Context, c *app.RequestContext) {
	superAgentRun(ctx, c, false, true)
}

// SuperAgentStreamRun exposes an SSE App Server entrypoint for external super-agent callers.
// @router /api/super-agent/runs/stream [POST]
func SuperAgentStreamRun(ctx context.Context, c *app.RequestContext) {
	superAgentRun(ctx, c, true, false)
}

// SuperAgentCancelRun cancels an active App Server super-agent run.
// @router /api/super-agent/runs/cancel [POST]
func SuperAgentCancelRun(ctx context.Context, c *app.RequestContext) {
	var req superAgentCancelRunRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		invalidParamRequestResponse(c, "run_id is required")
		return
	}
	// IDOR guard: only the owner of the run's conversation may cancel it.
	if _, authErr := authorizeSuperAgentRunOwner(ctx, runID); authErr != nil {
		internalServerErrorResponse(ctx, c, authErr)
		return
	}
	cancelled := conversation.CancelActiveAgentRun(runID)
	status := "not_active"
	if cancelled {
		status = "cancelled"
	}
	c.JSON(http.StatusOK, &superAgentCancelRunResponse{
		Code: 0,
		Msg:  "success",
		Data: superAgentCancelRunData{
			RunID:     runID,
			Status:    status,
			Cancelled: cancelled,
		},
	})
}

func superAgentRun(ctx context.Context, c *app.RequestContext, stream bool, requireConversation bool) {
	var req superAgentRunRequest
	if err := c.BindAndValidate(&req); err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}
	if req.AgentID == 0 && req.BotID == 0 {
		invalidParamRequestResponse(c, "agent_id or bot_id is required")
		return
	}
	if requireConversation && (req.ConversationID == nil || *req.ConversationID <= 0) {
		invalidParamRequestResponse(c, "conversation_id is required")
		return
	}
	chatReq, err := prepareSuperAgentRunRequest(ctx, &req, stream)
	if err != nil {
		invalidParamRequestResponse(c, err.Error())
		return
	}

	// P1 observability: record super-agent run latency + result by transport.
	transport := "sync"
	if stream {
		transport = "stream"
	}
	runStart := time.Now()
	recordRun := func(result string) {
		observability.SuperAgentRunsTotal.WithLabelValues(transport, result).Inc()
		observability.SuperAgentRunDuration.WithLabelValues(transport, result).Observe(time.Since(runStart).Seconds())
	}

	if !stream {
		resp, err := conversation.ConversationOpenAPISVC.OpenapiAgentRunNoStream(ctx, chatReq)
		if err != nil {
			observability.StudioAgentChatTotal.WithLabelValues("error").Inc()
			recordRun("error")
			c.JSON(http.StatusInternalServerError, &run.ErrorData{
				Code: errno.ErrConversationAgentRunError,
				Msg:  err.Error(),
			})
			return
		}
		observability.StudioAgentChatTotal.WithLabelValues("success").Inc()
		recordRun("success")
		// Closed-learning-loop: after a super-agent run, asynchronously review the
		// transcript to distill per-user memory and author/patch skills. Best-effort,
		// non-blocking, super-agent only — never affects the original single-agent flow.
		triggerSuperAgentPostRunReview(&req, resp)
		c.JSON(http.StatusOK, resp)
		return
	}

	c.SetStatusCode(http.StatusOK)
	c.SetContentType("text/event-stream; charset=utf-8")
	c.Response.Header.Set("X-Accel-Buffering", "no")
	sseSender := sseImpl.NewSSESender(sse.NewStream(c))
	if err := conversation.ConversationOpenAPISVC.OpenapiAgentRun(ctx, sseSender, chatReq); err != nil {
		observability.StudioAgentChatTotal.WithLabelValues("error").Inc()
		recordRun("error")
		errData := run.ErrorData{
			Code: errno.ErrConversationAgentRunError,
			Msg:  err.Error(),
		}
		ed, _ := json.Marshal(errData)
		_ = sseSender.Send(ctx, &sse.Event{
			Event: run.RunEventError,
			Data:  ed,
		})
		return
	}
	observability.StudioAgentChatTotal.WithLabelValues("success").Inc()
	recordRun("success")
}

// superAgentReviewTimeout bounds the async post-run review fork.
const superAgentReviewTimeout = 120 * time.Second

// superAgentReviewMaxAttempts is how many times the closed-learning review is retried
// (with exponential backoff) before giving up — review failure means lost memory/skills.
const superAgentReviewMaxAttempts = 3

// triggerSuperAgentPostRunReview fires the closed-learning-loop review fork for a
// finished super-agent run. It is best-effort and fully detached from the request:
// it must never block the response or affect the original single-agent flow.
func triggerSuperAgentPostRunReview(req *superAgentRunRequest, resp *conversation.ChatV3NoStreamResponse) {
	if req == nil || resp == nil || singleagentapp.SingleAgentSVC == nil || singleagentapp.SingleAgentSVC.DomainSVC == nil {
		return
	}
	botID := req.BotID
	if botID == 0 {
		botID = req.AgentID
	}
	if botID == 0 {
		return
	}
	transcript := buildSuperAgentReviewTranscript(req.AdditionalMessages, resp.Messages)
	// Need at least one user turn and one assistant turn to be worth reviewing.
	if len(transcript) < 2 {
		return
	}
	var connectorID int64
	if req.ConnectorID != nil {
		connectorID = *req.ConnectorID
	}
	identity := &crossagent.AgentIdentity{AgentID: botID, ConnectorID: connectorID, IsDraft: true}
	userID := req.User

	go func() {
		bg, cancel := context.WithTimeout(context.Background(), superAgentReviewTimeout)
		defer cancel()
		// 闭环学习是「越用越懂你」的引擎，复盘失败=记忆/技能永久丢失，所以做有限重试
		// （指数退避 1s/2s）。仍失败才放弃，避免无限重试拖垮后台。
		var lastErr error
		for attempt := 0; attempt < superAgentReviewMaxAttempts; attempt++ {
			summary, err := singleagentapp.SingleAgentSVC.DomainSVC.PostRunReview(bg, identity, userID, transcript)
			if err == nil {
				if s := strings.TrimSpace(summary); s != "" {
					logs.CtxInfof(bg, "[super-agent] post-run review for agent %d: %s", botID, s)
				}
				return
			}
			lastErr = err
			if attempt < superAgentReviewMaxAttempts-1 {
				select {
				case <-bg.Done():
					logs.CtxWarnf(bg, "[super-agent] post-run review aborted (ctx done) after %d attempt(s): %v", attempt+1, err)
					return
				case <-time.After(time.Duration(1<<uint(attempt)) * time.Second):
				}
			}
		}
		logs.CtxWarnf(bg, "[super-agent] post-run review failed after %d attempts (best-effort): %v", superAgentReviewMaxAttempts, lastErr)
	}()
}

// buildSuperAgentReviewTranscript turns the run's user input + assistant output
// into a plain message transcript for the review fork.
func buildSuperAgentReviewTranscript(input []*run.EnterMessage, output []*run.ChatV3MessageDetail) []*schema.Message {
	out := make([]*schema.Message, 0, len(input)+len(output))
	for _, m := range input {
		if m == nil {
			continue
		}
		if content := strings.TrimSpace(m.Content); content != "" {
			out = append(out, &schema.Message{Role: schema.User, Content: content})
		}
	}
	for _, m := range output {
		if m == nil {
			continue
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		role := schema.Assistant
		if m.Role == "user" {
			role = schema.User
		}
		out = append(out, &schema.Message{Role: role, Content: content})
	}
	return out
}

func prepareSuperAgentRunRequest(ctx context.Context, req *superAgentRunRequest, stream bool) (*run.ChatV3Request, error) {
	botID := req.BotID
	if botID == 0 {
		botID = req.AgentID
	}
	if botID == 0 {
		return nil, errors.New("agent_id or bot_id is required")
	}
	chatReq, err := superagentapp.PrepareRunRequest(ctx, &superagentapp.RunInput{
		SpaceID:            req.SpaceID,
		AgentID:            req.AgentID,
		BotID:              req.BotID,
		ConversationID:     req.ConversationID,
		User:               req.User,
		AdditionalMessages: req.AdditionalMessages,
		CustomVariables:    req.CustomVariables,
		MetaData:           req.MetaData,
		CustomConfig:       req.CustomConfig,
		ExtraParams:        req.ExtraParams,
		ConnectorID:        req.ConnectorID,
		ShortcutCommand:    req.ShortcutCommand,
		ClientID:           req.ClientID,
	}, stream)
	if err != nil {
		return nil, err
	}
	if err := checkParamsV3(ctx, chatReq); err != nil {
		return nil, err
	}
	return chatReq, nil
}
