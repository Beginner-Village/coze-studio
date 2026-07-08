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

package agentflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	"github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox"
)

// sandboxKeyFor 由 connector/agent/user_id 组合出稳定且容器名安全的沙箱 key。
// 委托到 agentsandbox.SandboxKeyFor,与沙箱空间管理 API 共用同一算法。
func sandboxKeyFor(connectorID, agentID int64, userID string) string {
	return agentsandbox.SandboxKeyFor(connectorID, agentID, userID)
}

// defaultMaxToolOutputBytes 是回灌给模型的工具输出上限（约几千 token）。
const defaultMaxToolOutputBytes = 16000

// maxToolOutputBytes 返回工具输出上限，可经环境变量 AGENT_TOOL_OUTPUT_MAX_BYTES 覆盖。
func maxToolOutputBytes() int {
	if v := os.Getenv("AGENT_TOOL_OUTPUT_MAX_BYTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultMaxToolOutputBytes
}

// truncateForModel 仅截断回灌给模型的字符串，不影响沙箱里真实写入/执行的内容。
// 超长时保留头部 70%、尾部 30%，中间插入 truncated 标记。
func truncateForModel(s string) string {
	max := maxToolOutputBytes()
	if len(s) <= max {
		// 清洗非法 UTF-8,避免入库 message 表报 MySQL 1366。
		return strings.ToValidUTF8(s, "")
	}
	// 按字节切会把多字节字符切成两半 → 非法 UTF-8,ToValidUTF8 去掉边界碎片。
	head := strings.ToValidUTF8(s[:max*7/10], "")
	tail := strings.ToValidUTF8(s[len(s)-max*3/10:], "")
	return head + fmt.Sprintf("\n\n...[truncated %d bytes]...\n\n", len(s)-len(head)-len(tail)) + tail
}

// resolvePath 把相对路径归一到 /workspace 下；绝对路径原样保留。
func resolvePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/workspace"
	}
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/workspace/" + p
}

// pathIsUnderSkills reports whether p refers to a path inside the /skills tree.
// It normalises Windows-style backslashes and leading "./" before checking,
// then resolves ".." segments with path.Clean to prevent traversal bypasses
// such as "/workspace/../skills/evil.py".
func pathIsUnderSkills(p string) bool {
	cleaned := strings.TrimPrefix(strings.ReplaceAll(p, "\\", "/"), "./")
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	cleaned = path.Clean(cleaned)
	return cleaned == "/skills" || strings.HasPrefix(cleaned, "/skills/")
}

// guardSkillWrite returns an error when readonlySkills is true and target falls
// under /skills. Non-instance callers (readonlySkills=false) are always allowed.
func guardSkillWrite(readonlySkills bool, target string) error {
	if readonlySkills && pathIsUnderSkills(target) {
		return fmt.Errorf("permission denied: /skills is read-only for a virtual employee instance")
	}
	return nil
}

// bashCommandWritesSkills is a best-effort heuristic that returns true when a
// bash command appears to write into the /skills tree. It checks for shell
// redirects (> /skills/…, >> /skills/…) and common write commands (tee, cp,
// mv). It cannot catch every possible construct; the read-only mount (Task 8)
// is the hard enforcement layer.
func bashCommandWritesSkills(cmd string) bool {
	// Patterns: redirect targets and explicit write commands
	writePatterns := []string{
		"> /skills/",
		">> /skills/",
		"tee /skills/",
		"tee -a /skills/",
		"cp ", // checked below with target
		"mv ", // checked below with target
	}
	for _, pat := range writePatterns {
		if strings.Contains(cmd, pat) {
			// For cp/mv we need the destination to be /skills
			if (pat == "cp " || pat == "mv ") && !strings.Contains(cmd, "/skills/") {
				continue
			}
			return true
		}
	}
	return false
}

// ---- run_bash ----

type runBashTool struct {
	key            string
	readonlySkills bool
}

type runBashRequest struct {
	Command    string `json:"command" jsonschema:"description=The bash command to run inside the sandbox /workspace directory"`
	TimeoutSec int    `json:"timeout_sec,omitempty" jsonschema:"description=Optional timeout in seconds (default 60)"`
}

func (t *runBashTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "run_bash",
		Desc: "Run a bash command inside the user's persistent sandbox (working directory /workspace). Use this to execute skill scripts (e.g. `python /skills/<name>/scripts/run.py`) or arbitrary shell commands. Returns stdout, stderr and exit code.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command":     {Type: schema.String, Desc: "The bash command to run", Required: true},
			"timeout_sec": {Type: schema.Integer, Desc: "Optional timeout in seconds (default 60)", Required: false},
		}),
	}, nil
}

func (t *runBashTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req runBashRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	if strings.TrimSpace(req.Command) == "" {
		return "Error: command is required", nil
	}
	// Best-effort guard: detect common shell write patterns targeting /skills
	// (redirect >, tee, cp, mv). Cannot catch every shell construct; the
	// read-only mount in Task 8 is the hard enforcement layer.
	if t.readonlySkills && bashCommandWritesSkills(req.Command) {
		return "Error: permission denied: /skills is read-only for a virtual employee instance", nil
	}
	if isMutatingCommand(req.Command) {
		if ok, reason := checkMutationAllowed("running a command that writes or deletes files"); !ok {
			return reason, nil
		}
	}
	res, err := svc.Exec(ctx, t.key, req.Command, req.TimeoutSec)
	if err != nil {
		return fmt.Sprintf("Error running command: %v", err), nil
	}
	stdout := offloadToolResultOrTruncate(ctx, svc, t.key, toolOutputOffloadMeta{
		Tool:      "run_bash",
		Arguments: req,
	}, res.Stdout)
	return fmt.Sprintf("exit_code: %d\nstdout:\n%s\nstderr:\n%s", res.ExitCode, stdout, truncateForModel(res.Stderr)), nil
}

// ---- read_file ----

type readFileTool struct{ key string }

type readFileRequest struct {
	Path string `json:"path" jsonschema:"description=File path; relative paths resolve under /workspace"`
}

func (t *readFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_file",
		Desc: "Read the contents of a file in the sandbox. Relative paths resolve under /workspace. Use for files under /skills or /workspace.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {Type: schema.String, Desc: "File path to read", Required: true},
		}),
	}, nil
}

func (t *readFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req readFileRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	b, err := svc.ReadFile(ctx, t.key, resolvePath(req.Path))
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err), nil
	}
	return offloadToolResultOrTruncate(ctx, svc, t.key, toolOutputOffloadMeta{
		Tool:      "read_file",
		Arguments: req,
	}, string(b)), nil
}

// ---- write_file ----

type writeFileTool struct {
	key            string
	readonlySkills bool
}

type writeFileRequest struct {
	Path    string `json:"path" jsonschema:"description=File path; relative paths resolve under /workspace"`
	Content string `json:"content" jsonschema:"description=Content to write"`
}

func (t *writeFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "write_file",
		Desc: "Write (create or overwrite) a file in the sandbox. Relative paths resolve under /workspace. Parent directories are created automatically. Files under /workspace persist across sessions for the same user.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path":    {Type: schema.String, Desc: "File path to write", Required: true},
			"content": {Type: schema.String, Desc: "Content to write", Required: true},
		}),
	}, nil
}

func (t *writeFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req writeFileRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	if strings.TrimSpace(req.Path) == "" {
		return "Error: path is required", nil
	}
	if err := guardSkillWrite(t.readonlySkills, resolvePath(req.Path)); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if ok, reason := checkMutationAllowed("write_file"); !ok {
		return reason, nil
	}
	wp := resolvePath(req.Path)
	unlock := lockSandboxFile(t.key, wp)
	err := svc.WriteFile(ctx, t.key, wp, []byte(req.Content))
	unlock()
	if err != nil {
		return fmt.Sprintf("Error writing file: %v", err), nil
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(req.Content), resolvePath(req.Path)), nil
}

// ---- list_files ----

type listFilesTool struct{ key string }

type listFilesRequest struct {
	Path string `json:"path,omitempty" jsonschema:"description=Directory path; relative paths resolve under /workspace; default /workspace"`
}

func (t *listFilesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_files",
		Desc: "List the entries of a directory in the sandbox. Relative paths resolve under /workspace; default is /workspace.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {Type: schema.String, Desc: "Directory path to list", Required: false},
		}),
	}, nil
}

func (t *listFilesTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req listFilesRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	files, err := svc.ListFiles(ctx, t.key, resolvePath(req.Path))
	if err != nil {
		return fmt.Sprintf("Error listing files: %v", err), nil
	}
	return strings.Join(files, "\n"), nil
}

// defaultAgentMaxStep 是普通单 Agent 的 ReAct 默认最大步数（约 15 轮工具往返）。
const defaultAgentMaxStep = 30

// superAgentMaxStep:超级体(harness 智能体)不设实际可达的步数上限。
// 超级体对标 Claude Code / Codex —— 长程任务可能需要任意多轮工具往返,绝不能在中途
// 因为「步数用完」被打断。eino 的 cyclic graph 不支持真正的「无限」(maxRunSteps==0
// 会被改成更小的默认值,步数检查也是硬性的),且 react 会按 MaxStep+1 预分配 slice,
// 因此这里取一个实际永远到不了的高值(10 万步 ≈ 5 万轮工具往返,预分配仅 ~800KB)。
// 真正的运行兜底交给 context(用户主动停止 / 请求超时),而不是固定步数。
const superAgentMaxStep = 100000

// agentMaxStep 返回 ReAct 最大步数。
//   - 普通单 Agent 默认 30,可经 AGENT_MAX_STEP 覆盖。
//   - 超级体取 superAgentMaxStep(实际不限制),可经 SUPER_AGENT_MAX_STEP 覆盖。
//
// super 仅影响超级体,绝不改变原生单 Agent 行为。
func agentMaxStep(super bool) int {
	if super {
		if v := os.Getenv("SUPER_AGENT_MAX_STEP"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				return n
			}
		}
		return superAgentMaxStep
	}
	if v := os.Getenv("AGENT_MAX_STEP"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultAgentMaxStep
}

// sandboxToolsEnabled 决定是否给该 agent 挂载沙箱工具。
// 规则：环境变量 SANDBOX_TOOLS_ENABLED=true 强制开启；否则当 agent 绑定了技能时开启。
func sandboxToolsEnabled(skillCount int) bool {
	if strings.EqualFold(os.Getenv("SANDBOX_TOOLS_ENABLED"), "true") {
		return true
	}
	return skillCount > 0
}

// skillExecutionDisabled 报告是否通过环境变量 SKILL_EXECUTION_DISABLED=true 全局关闭
// 「普通(非超级)智能体」的技能脚本执行。开启后普通体把技能当成只读文档：read_skill 仍
// 返回 SKILL.md 正文，但不再向沙箱注入脚本、也不挂载 run_bash/write_file 等沙箱工具——
// 技能变成「只能读说明、不能跑脚本」。
func skillExecutionDisabled() bool {
	return strings.EqualFold(os.Getenv("SKILL_EXECUTION_DISABLED"), "true")
}

// skillReadOnlyMode 报告「这个 agent 是否应把技能当只读」。它只可能对普通体为 true：
// 超级体(harness)永远不受影响，以保持「超级体/普通体互不改变对方行为」的铁律。
// 普通体只读的判定 = per-agent 开关关闭(perAgentSkillExecEnabled=false) 或 全局 env 强制关闭。
// 即：每个智能体各自决定是否允许执行技能脚本，另留一个全局 env 兜底(运维一键全关)。
func skillReadOnlyMode(isSuper bool, perAgentSkillExecEnabled bool) bool {
	if isSuper {
		return false
	}
	return !perAgentSkillExecEnabled || skillExecutionDisabled()
}

// shouldMountSandbox 集中决定是否给某个 agent 挂载沙箱工具集
// (run_bash/read_file/write_file/…)：
//   - workflow-canvas 模式：从不挂(画布有自己的工具集)。
//   - 超级体：除非沙箱总开关关闭(纯 MCP 模式)，否则挂。
//   - 普通体：仅当技能触发(sandboxToolsEnabled)且未处于只读技能模式时才挂。
func shouldMountSandbox(isSuper, workflowCanvasMode, sandboxOff, skillReadOnly bool, skillCount int) bool {
	if workflowCanvasMode {
		return false
	}
	if isSuper {
		return !sandboxOff
	}
	return !skillReadOnly && sandboxToolsEnabled(skillCount)
}

// resolveSandboxOff 报告「沙箱对该 agent 是否关闭」。它在两种情况下为 true：
//   - 全局沙箱服务不可用（现场未配置沙箱，SANDBOX_ENABLED=false → DefaultSVC()==nil）；
//   - 超级体的 per-agent 沙箱总开关关闭（纯 MCP 模式）。
//
// 普通体没有 per-agent 沙箱总开关，只受「服务是否可用」这一半影响。sandboxOff=true 时，
// 沙箱工具/web_search/web_fetch/skill_manage/deep_task 都会从工具集里剔除，技能脚本不注入。
func resolveSandboxOff(sandboxAvailable, isSuper, superSandboxEnabled bool) bool {
	if !sandboxAvailable {
		return true
	}
	return isSuper && !superSandboxEnabled
}

// ---- update_plan (DeepAgents 风格的显式规划/进度追踪) ----

type updatePlanTool struct{ key string }

type planStep struct {
	Content string `json:"content" jsonschema:"description=The step description"`
	Status  string `json:"status" jsonschema:"description=One of: pending, in_progress, completed (done is accepted as a compatibility alias)"`
}

type updatePlanRequest struct {
	Plan []planStep `json:"plan" jsonschema:"description=The full ordered list of plan steps with their current status"`
}

const planFilePath = "/workspace/.plan.json"

func (t *updatePlanTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "update_plan",
		Desc: "Maintain an explicit, ordered plan (todo list) for a complex task and track progress. Call it first to decompose the task into steps, then call it again to update step statuses as you complete them. The plan is persisted across the conversation. Each step has a status: pending, in_progress, or completed (done is accepted as a compatibility alias).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"plan": {
				Type:     schema.Array,
				Desc:     "The full ordered list of plan steps (replaces the previous plan)",
				Required: true,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"content": {Type: schema.String, Desc: "The step description", Required: true},
						"status":  {Type: schema.String, Desc: "One of: pending, in_progress, completed (done is accepted as a compatibility alias)", Required: true},
					},
				},
			},
		}),
	}, nil
}

func (t *updatePlanTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req updatePlanRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	if len(req.Plan) == 0 {
		return "Error: plan must contain at least one step", nil
	}
	blob, _ := json.MarshalIndent(req.Plan, "", "  ")
	unlock := lockSandboxFile(t.key, planFilePath)
	err := svc.WriteFile(ctx, t.key, planFilePath, blob)
	unlock()
	if err != nil {
		return fmt.Sprintf("Error persisting plan: %v", err), nil
	}
	return renderPlan(req.Plan), nil
}

func renderPlan(steps []planStep) string {
	var b strings.Builder
	done := 0
	for _, s := range steps {
		mark := "[ ]"
		switch s.Status {
		case "done", "completed":
			mark = "[x]"
			done++
		case "in_progress":
			mark = "[~]"
		}
		b.WriteString(mark + " " + s.Content + "\n")
	}
	return fmt.Sprintf("Plan updated (%d/%d done):\n%s", done, len(steps), b.String())
}

// ---- edit_file ----

type editFileTool struct {
	key            string
	readonlySkills bool
}

type editFileRequest struct {
	Path       string `json:"path" jsonschema:"description=File path; relative paths resolve under /workspace"`
	OldString  string `json:"old_string" jsonschema:"description=The exact text to replace (must match verbatim, including indentation)"`
	NewString  string `json:"new_string" jsonschema:"description=The replacement text"`
	ReplaceAll bool   `json:"replace_all,omitempty" jsonschema:"description=Replace every occurrence; default false (old_string must be unique)"`
}

func (t *editFileTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "edit_file",
		Desc: "Edit a file in the sandbox by exact string replacement (search-replace). Read the file first. old_string must match verbatim and be unique unless replace_all=true. Prefer this over write_file for modifying existing files.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path":        {Type: schema.String, Desc: "File path to edit", Required: true},
			"old_string":  {Type: schema.String, Desc: "Exact text to replace", Required: true},
			"new_string":  {Type: schema.String, Desc: "Replacement text", Required: true},
			"replace_all": {Type: schema.Boolean, Desc: "Replace all occurrences (default false)", Required: false},
		}),
	}, nil
}

func (t *editFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req editFileRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	if strings.TrimSpace(req.Path) == "" {
		return "Error: path is required", nil
	}
	if err := guardSkillWrite(t.readonlySkills, resolvePath(req.Path)); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if ok, reason := checkMutationAllowed("edit_file"); !ok {
		return reason, nil
	}
	ep := resolvePath(req.Path)
	unlock := lockSandboxFile(t.key, ep)
	n, err := svc.EditFile(ctx, t.key, ep, req.OldString, req.NewString, req.ReplaceAll)
	unlock()
	if err != nil {
		return fmt.Sprintf("Error editing file: %v", err), nil
	}
	return fmt.Sprintf("Edited %s (%d replacement(s))", resolvePath(req.Path), n), nil
}

// ---- grep ----

type grepTool struct{ key string }

type grepRequest struct {
	Pattern string `json:"pattern" jsonschema:"description=Regex pattern to search for"`
	Path    string `json:"path,omitempty" jsonschema:"description=Directory or file to search; default /workspace"`
}

func (t *grepTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "grep",
		Desc: "Search file contents in the sandbox by regex (ripgrep, falls back to grep). Returns matching lines with file:line. Relative paths resolve under /workspace.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern": {Type: schema.String, Desc: "Regex to search for", Required: true},
			"path":    {Type: schema.String, Desc: "Directory or file to search (default /workspace)", Required: false},
		}),
	}, nil
}

func (t *grepTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req grepRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	path := ""
	if strings.TrimSpace(req.Path) != "" {
		path = resolvePath(req.Path)
	}
	out, err := svc.Grep(ctx, t.key, req.Pattern, path)
	if err != nil {
		return fmt.Sprintf("Error running grep: %v", err), nil
	}
	return offloadToolResultOrTruncate(ctx, svc, t.key, toolOutputOffloadMeta{
		Tool:      "grep",
		Arguments: req,
	}, out), nil
}

// ---- glob ----

type globTool struct{ key string }

type globRequest struct {
	Pattern string `json:"pattern" jsonschema:"description=Filename glob pattern, e.g. *.go"`
}

func (t *globTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "glob",
		Desc: "Find files in the sandbox /workspace by filename pattern (e.g. *.go). Returns matching file paths.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern": {Type: schema.String, Desc: "Filename glob pattern", Required: true},
		}),
	}, nil
}

func (t *globTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	svc := crosssandbox.DefaultSVC()
	if svc == nil {
		return "Error: sandbox is not available", nil
	}
	var req globRequest
	if err := json.Unmarshal([]byte(argumentsInJSON), &req); err != nil {
		return argParseErrMsg(err), nil
	}
	out, err := svc.Glob(ctx, t.key, req.Pattern)
	if err != nil {
		return fmt.Sprintf("Error running glob: %v", err), nil
	}
	return offloadToolResultOrTruncate(ctx, svc, t.key, toolOutputOffloadMeta{
		Tool:      "glob",
		Arguments: req,
	}, out), nil
}

// newSandboxTools 构造沙箱工具。沙箱服务未初始化时返回 nil。
// readonlySkills=true 时，write_file / edit_file / run_bash 工具会拒绝写入 /skills 树
// （虚拟员工实例使用此标志，普通 agent 传 false）。
func newSandboxTools(key string, readonlySkills bool) []tool.InvokableTool {
	if crosssandbox.DefaultSVC() == nil {
		return nil
	}
	return []tool.InvokableTool{
		&runBashTool{key: key, readonlySkills: readonlySkills},
		&readFileTool{key: key},
		&writeFileTool{key: key, readonlySkills: readonlySkills},
		&editFileTool{key: key, readonlySkills: readonlySkills},
		&listFilesTool{key: key},
		&grepTool{key: key},
		&globTool{key: key},
		&updatePlanTool{key: key},
	}
}
