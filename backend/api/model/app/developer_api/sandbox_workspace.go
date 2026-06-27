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

package developer_api

// 超级智能体「沙箱空间管理」接口的请求/响应模型(手写，Hertz 按 json tag 绑定)。

type SandboxFileInfo struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
	// Mtime 为文件最后修改时间(Unix 秒);前端用于显示创建/修改时间并按新→旧排序。
	Mtime int64 `json:"mtime"`
}

// ---- 列目录 ----

type ListSandboxFilesRequest struct {
	SpaceID     int64   `json:"space_id,string" query:"space_id"`
	AgentID     int64   `json:"agent_id,string" query:"agent_id"`
	BotID       int64   `json:"bot_id,string" query:"bot_id"`
	Path        string  `json:"path" query:"path"`
	Recursive   bool    `json:"recursive,omitempty" query:"recursive"`
	ConnectorID *string `json:"connector_id,omitempty" query:"connector_id"`
}

type ListSandboxFilesData struct {
	Path  string             `json:"path"`
	Files []*SandboxFileInfo `json:"files"`
}

type ListSandboxFilesResponse struct {
	Code int64                 `json:"code"`
	Msg  string                `json:"msg"`
	Data *ListSandboxFilesData `json:"data"`
}

// ---- 读文件 ----

type ReadSandboxFileRequest struct {
	SpaceID     int64   `json:"space_id,string" query:"space_id"`
	AgentID     int64   `json:"agent_id,string" query:"agent_id"`
	BotID       int64   `json:"bot_id,string" query:"bot_id"`
	Path        string  `json:"path" query:"path"`
	ConnectorID *string `json:"connector_id,omitempty" query:"connector_id"`
}

type ReadSandboxFileData struct {
	Path string `json:"path"`
	// Content 为 UTF-8 文本;二进制内容时 IsBinary=true 且 Content 为 base64。
	Content  string `json:"content"`
	IsBinary bool   `json:"is_binary"`
	Size     int64  `json:"size"`
	// TotalSize 为文件真实大小;IsTruncated=true 表示文件超过读取上限,
	// 二进制超限时不回传 Content(截断会损坏文件),前端据此提示下载。
	TotalSize   int64 `json:"total_size"`
	IsTruncated bool  `json:"is_truncated"`
}

type ReadSandboxFileResponse struct {
	Code int64                `json:"code"`
	Msg  string               `json:"msg"`
	Data *ReadSandboxFileData `json:"data"`
}

// ---- 下载文件 ----

type DownloadSandboxFileRequest = ReadSandboxFileRequest

type DownloadSandboxFileData struct {
	Path    string `json:"path"`
	Content []byte `json:"-"`
	Size    int64  `json:"size"`
}

// ---- 写/上传文件 ----

type UploadSandboxFileRequest struct {
	SpaceID int64  `json:"space_id,string"`
	AgentID int64  `json:"agent_id,string"`
	BotID   int64  `json:"bot_id,string"`
	Path    string `json:"path"`
	// Content 为文本内容;二进制请传 base64 并置 IsBase64=true。
	Content     string  `json:"content"`
	IsBase64    bool    `json:"is_base64"`
	Encoding    string  `json:"encoding,omitempty"`
	ConnectorID *string `json:"connector_id,omitempty"`
}

type UploadSandboxFileData struct {
	Path string `json:"path"`
}

type UploadSandboxFileResponse struct {
	Code int64                  `json:"code"`
	Msg  string                 `json:"msg"`
	Data *UploadSandboxFileData `json:"data"`
}

// ---- 删除文件 ----

type DeleteSandboxFileRequest struct {
	SpaceID     int64   `json:"space_id,string"`
	AgentID     int64   `json:"agent_id,string"`
	BotID       int64   `json:"bot_id,string"`
	Path        string  `json:"path"`
	ConnectorID *string `json:"connector_id,omitempty"`
}

type DeleteSandboxFileResponse struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// ---- 移动/重命名文件 ----

type MoveSandboxFileRequest struct {
	SpaceID     int64   `json:"space_id,string"`
	AgentID     int64   `json:"agent_id,string"`
	BotID       int64   `json:"bot_id,string"`
	Path        string  `json:"path"`
	FromPath    string  `json:"from_path,omitempty"`
	TargetPath  string  `json:"target_path"`
	ToPath      string  `json:"to_path,omitempty"`
	ConnectorID *string `json:"connector_id,omitempty"`
}

type MoveSandboxFileData struct {
	Path     string `json:"path"`
	FromPath string `json:"from_path"`
}

type MoveSandboxFileResponse struct {
	Code int64                `json:"code"`
	Msg  string               `json:"msg"`
	Data *MoveSandboxFileData `json:"data"`
}

// ---- 创建目录 ----

type CreateSandboxDirectoryRequest struct {
	SpaceID     int64   `json:"space_id,string"`
	AgentID     int64   `json:"agent_id,string"`
	BotID       int64   `json:"bot_id,string"`
	Path        string  `json:"path"`
	ConnectorID *string `json:"connector_id,omitempty"`
}

type CreateSandboxDirectoryData struct {
	Path string `json:"path"`
}

type CreateSandboxDirectoryResponse struct {
	Code int64                       `json:"code"`
	Msg  string                      `json:"msg"`
	Data *CreateSandboxDirectoryData `json:"data"`
}

// ---- 查询文件元信息 ----

type StatSandboxFileRequest struct {
	SpaceID     int64   `json:"space_id,string"`
	AgentID     int64   `json:"agent_id,string"`
	BotID       int64   `json:"bot_id,string"`
	Path        string  `json:"path"`
	ConnectorID *string `json:"connector_id,omitempty"`
}

type StatSandboxFileData struct {
	Path   string           `json:"path"`
	Exists bool             `json:"exists"`
	File   *SandboxFileInfo `json:"file,omitempty"`
}

type StatSandboxFileResponse struct {
	Code int64                `json:"code"`
	Msg  string               `json:"msg"`
	Data *StatSandboxFileData `json:"data"`
}

// ---- 搜索文件内容 / 匹配文件名 ----

type GrepSandboxFilesRequest struct {
	SpaceID       int64    `json:"space_id,string"`
	AgentID       int64    `json:"agent_id,string"`
	BotID         int64    `json:"bot_id,string"`
	Path          string   `json:"path"`
	Pattern       string   `json:"pattern"`
	CaseSensitive *bool    `json:"case_sensitive,omitempty"`
	Include       []string `json:"include,omitempty"`
	Exclude       []string `json:"exclude,omitempty"`
	ConnectorID   *string  `json:"connector_id,omitempty"`
}

type GrepSandboxFilesData struct {
	Path        string `json:"path"`
	Pattern     string `json:"pattern"`
	Output      string `json:"output"`
	IsTruncated bool   `json:"is_truncated"`
}

type GrepSandboxFilesResponse struct {
	Code int64                 `json:"code"`
	Msg  string                `json:"msg"`
	Data *GrepSandboxFilesData `json:"data"`
}

type GlobSandboxFilesRequest struct {
	SpaceID     int64   `json:"space_id,string"`
	AgentID     int64   `json:"agent_id,string"`
	BotID       int64   `json:"bot_id,string"`
	Path        string  `json:"path"`
	Pattern     string  `json:"pattern"`
	Limit       int32   `json:"limit,omitempty"`
	ConnectorID *string `json:"connector_id,omitempty"`
}

type GlobSandboxFilesData struct {
	Path        string   `json:"path"`
	Pattern     string   `json:"pattern"`
	Matches     []string `json:"matches"`
	IsTruncated bool     `json:"is_truncated"`
}

type GlobSandboxFilesResponse struct {
	Code int64                 `json:"code"`
	Msg  string                `json:"msg"`
	Data *GlobSandboxFilesData `json:"data"`
}

// ---- 精确编辑文件 ----

type EditSandboxFileRequest struct {
	SpaceID     int64   `json:"space_id,string"`
	AgentID     int64   `json:"agent_id,string"`
	BotID       int64   `json:"bot_id,string"`
	Path        string  `json:"path"`
	OldString   string  `json:"old_string"`
	NewString   string  `json:"new_string"`
	ReplaceAll  bool    `json:"replace_all,omitempty"`
	ConnectorID *string `json:"connector_id,omitempty"`
}

type EditSandboxFileData struct {
	Path         string `json:"path"`
	Replacements int32  `json:"replacements"`
}

type EditSandboxFileResponse struct {
	Code int64                `json:"code"`
	Msg  string               `json:"msg"`
	Data *EditSandboxFileData `json:"data"`
}

// ---- Codex-style patch ----

type ApplySandboxPatchRequest struct {
	SpaceID      int64   `json:"space_id,string"`
	AgentID      int64   `json:"agent_id,string"`
	BotID        int64   `json:"bot_id,string"`
	WorkDir      string  `json:"workdir,omitempty"`
	WorkDirAlias string  `json:"work_dir,omitempty"`
	Patch        string  `json:"patch"`
	ConnectorID  *string `json:"connector_id,omitempty"`
}

type ApplySandboxPatchData struct {
	ChangedFiles int32    `json:"changed_files"`
	Paths        []string `json:"paths"`
}

type ApplySandboxPatchResponse struct {
	Code int64                  `json:"code"`
	Msg  string                 `json:"msg"`
	Data *ApplySandboxPatchData `json:"data"`
}

// ---- 执行沙箱命令 ----

type ExecSandboxCommandRequest struct {
	SpaceID      int64   `json:"space_id,string"`
	AgentID      int64   `json:"agent_id,string"`
	BotID        int64   `json:"bot_id,string"`
	Command      string  `json:"command"`
	WorkDir      string  `json:"workdir,omitempty"`
	WorkDirAlias string  `json:"work_dir,omitempty"`
	TimeoutSec   int     `json:"timeout_sec,omitempty"`
	ConnectorID  *string `json:"connector_id,omitempty"`
}

type ExecSandboxCommandData struct {
	ExitCode    int32  `json:"exit_code"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	IsTruncated bool   `json:"is_truncated"`
}

type ExecSandboxCommandResponse struct {
	Code int64                   `json:"code"`
	Msg  string                  `json:"msg"`
	Data *ExecSandboxCommandData `json:"data"`
}

// ---- harness 状态 ----

type SuperAgentHarnessStateRequest struct {
	SpaceID        int64   `json:"space_id,string" query:"space_id"`
	AgentID        int64   `json:"agent_id,string" query:"agent_id"`
	BotID          int64   `json:"bot_id,string" query:"bot_id"`
	ConversationID int64   `json:"conversation_id,string,omitempty" query:"conversation_id"`
	ConnectorID    *string `json:"connector_id,omitempty" query:"connector_id"`
}

type SuperAgentHarnessPlanStep struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

type SuperAgentHarnessPlanUpdateRequest struct {
	SpaceID        int64                        `json:"space_id,string"`
	AgentID        int64                        `json:"agent_id,string"`
	BotID          int64                        `json:"bot_id,string"`
	ConversationID int64                        `json:"conversation_id,string,omitempty"`
	Plan           []*SuperAgentHarnessPlanStep `json:"plan"`
	Items          []*SuperAgentHarnessPlanStep `json:"items"`
	ConnectorID    *string                      `json:"connector_id,omitempty"`
}

type SuperAgentHarnessPlanUpdateData struct {
	Path    string `json:"path"`
	Steps   int32  `json:"steps"`
	Content string `json:"content"`
}

type SuperAgentHarnessPlanUpdateResponse struct {
	Code int64                            `json:"code"`
	Msg  string                           `json:"msg"`
	Data *SuperAgentHarnessPlanUpdateData `json:"data"`
}

type SuperAgentHarnessPlanState struct {
	Path        string `json:"path"`
	Exists      bool   `json:"exists"`
	Content     string `json:"content"`
	IsBinary    bool   `json:"is_binary"`
	Size        int64  `json:"size"`
	TotalSize   int64  `json:"total_size"`
	IsTruncated bool   `json:"is_truncated"`
	Mtime       int64  `json:"mtime"`
}

type SuperAgentHarnessToolOutputsState struct {
	Root  string             `json:"root"`
	Path  string             `json:"path"`
	Files []*SandboxFileInfo `json:"files"`
}

type SuperAgentHarnessRuntimeSkillSummary struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	EntryPath   string   `json:"entry_path"`
	Standard    bool     `json:"standard"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Category    string   `json:"category"`
	FilePaths   []string `json:"file_paths"`
	AssetPaths  []string `json:"asset_paths"`
	ImagePaths  []string `json:"image_paths"`
}

type SuperAgentHarnessRuntimeSkillsState struct {
	Root   string                                  `json:"root"`
	Skills []*SuperAgentHarnessRuntimeSkillSummary `json:"skills"`
}

type SuperAgentHarnessContextSummary struct {
	Version           string   `json:"version,omitempty"`
	Summary           string   `json:"summary,omitempty"`
	UpdatedAt         int64    `json:"updated_at,omitempty"`
	MessageID         string   `json:"message_id,omitempty"`
	RunID             string   `json:"run_id,omitempty"`
	Trigger           string   `json:"trigger,omitempty"`
	OriginalMessages  int      `json:"original_messages,omitempty"`
	CompactedMessages int      `json:"compacted_messages,omitempty"`
	RetainedMessages  int      `json:"retained_messages,omitempty"`
	OriginalBytes     int      `json:"original_bytes,omitempty"`
	MaxBytes          int      `json:"max_bytes,omitempty"`
	SummaryPath       string   `json:"summary_path,omitempty"`
	KeyFiles          []string `json:"key_files,omitempty"`
	Artifacts         []string `json:"artifacts,omitempty"`
	NextActions       []string `json:"next_actions,omitempty"`
}

type SuperAgentHarnessContextPolicy struct {
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

type SuperAgentHarnessContextState struct {
	Strategy             string                           `json:"strategy"`
	SummaryPath          string                           `json:"summary_path"`
	SummaryExists        bool                             `json:"summary_exists"`
	Summary              *SuperAgentHarnessContextSummary `json:"summary,omitempty"`
	Policy               *SuperAgentHarnessContextPolicy  `json:"policy,omitempty"`
	ToolOutputRoot       string                           `json:"tool_output_root"`
	ToolOutputOffload    bool                             `json:"tool_output_offload"`
	RecentMessagesPolicy string                           `json:"recent_messages_policy"`
	Components           []string                         `json:"components"`
}

type SuperAgentHarnessStateData struct {
	Plan          *SuperAgentHarnessPlanState          `json:"plan"`
	ToolOutputs   *SuperAgentHarnessToolOutputsState   `json:"tool_outputs"`
	RuntimeSkills *SuperAgentHarnessRuntimeSkillsState `json:"runtime_skills"`
	Context       *SuperAgentHarnessContextState       `json:"context,omitempty"`
}

type SuperAgentHarnessStateResponse struct {
	Code int64                       `json:"code"`
	Msg  string                      `json:"msg"`
	Data *SuperAgentHarnessStateData `json:"data"`
}

type SuperAgentHarnessContextClearRequest struct {
	SpaceID        int64   `json:"space_id,string,omitempty" query:"space_id"`
	AgentID        int64   `json:"agent_id,string,omitempty" query:"agent_id"`
	BotID          int64   `json:"bot_id,string,omitempty" query:"bot_id"`
	ConversationID int64   `json:"conversation_id,string,omitempty" query:"conversation_id"`
	ConnectorID    *string `json:"connector_id,omitempty" query:"connector_id"`
	DryRun         bool    `json:"dry_run,omitempty" query:"dry_run"`
}

type SuperAgentHarnessContextClearData struct {
	ConversationID      string `json:"conversation_id,omitempty"`
	SummaryPath         string `json:"summary_path"`
	SummaryExistsBefore bool   `json:"summary_exists_before"`
	Cleared             bool   `json:"cleared"`
	DryRun              bool   `json:"dry_run"`
}

type SuperAgentHarnessContextClearResponse struct {
	Code int64                              `json:"code"`
	Msg  string                             `json:"msg"`
	Data *SuperAgentHarnessContextClearData `json:"data"`
}
