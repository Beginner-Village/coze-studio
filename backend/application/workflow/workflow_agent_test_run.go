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

package workflow

import (
	"context"
	"errors"
	"strings"
	"time"

	modelworkflow "github.com/ynet-dev/ynet-studio/backend/api/model/workflow"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
)

const (
	defaultAgentWorkflowTestRunTimeout  = 60 * time.Second
	defaultAgentWorkflowTestRunInterval = time.Second
	maxAgentWorkflowTestRunTimeout      = 120 * time.Second
	minAgentWorkflowTestRunTimeout      = time.Second
	minAgentWorkflowTestRunInterval     = 200 * time.Millisecond
	maxAgentWorkflowTestRunInterval     = 5 * time.Second
	agentWorkflowTestRunTextLimit       = 4000
)

type AgentWorkflowTestRunRequest struct {
	WorkflowID string            `json:"workflow_id"`
	SpaceID    string            `json:"space_id"`
	Input      map[string]string `json:"input,omitempty"`
	BotID      string            `json:"bot_id,omitempty"`
	ProjectID  string            `json:"project_id,omitempty"`
	CommitID   string            `json:"commit_id,omitempty"`
	TimeoutMs  int               `json:"timeout_ms,omitempty"`
	IntervalMs int               `json:"interval_ms,omitempty"`
}

type AgentWorkflowTestRunResponse struct {
	Code int64                     `json:"code"`
	Msg  string                    `json:"msg"`
	Data *AgentWorkflowTestRunData `json:"data"`
}

type AgentWorkflowTestRunData struct {
	WorkflowID        string                          `json:"workflow_id"`
	ExecuteID         string                          `json:"execute_id"`
	Status            string                          `json:"status"`
	ExecuteStatus     modelworkflow.WorkflowExeStatus `json:"execute_status"`
	ExecuteStatusText string                          `json:"execute_status_text"`
	Reason            string                          `json:"reason,omitempty"`
	LastNodeID        string                          `json:"last_node_id,omitempty"`
	LogID             string                          `json:"log_id,omitempty"`
	Rate              string                          `json:"rate,omitempty"`
	WorkflowExeCost   string                          `json:"workflow_exe_cost,omitempty"`
	Nodes             []*AgentWorkflowTestRunNode     `json:"nodes,omitempty"`
	FailedNodes       []*AgentWorkflowTestRunNode     `json:"failed_nodes,omitempty"`
	OutputNodes       []*AgentWorkflowTestRunNode     `json:"output_nodes,omitempty"`
}

type AgentWorkflowTestRunNode struct {
	NodeID      string                      `json:"node_id"`
	NodeType    string                      `json:"node_type"`
	NodeName    string                      `json:"node_name"`
	NodeStatus  modelworkflow.NodeExeStatus `json:"node_status"`
	StatusText  string                      `json:"status_text"`
	ErrorInfo   string                      `json:"error_info,omitempty"`
	ErrorLevel  string                      `json:"error_level,omitempty"`
	Input       string                      `json:"input,omitempty"`
	Output      string                      `json:"output,omitempty"`
	RawOutput   string                      `json:"raw_output,omitempty"`
	NodeExeCost string                      `json:"node_exe_cost,omitempty"`
	ExecuteID   string                      `json:"execute_id,omitempty"`
}

type agentWorkflowTestRunService interface {
	TestRun(context.Context, *modelworkflow.WorkFlowTestRunRequest) (*modelworkflow.WorkFlowTestRunResponse, error)
	GetProcess(context.Context, *modelworkflow.GetWorkflowProcessRequest) (*modelworkflow.GetWorkflowProcessResponse, error)
}

type agentWorkflowTestRunConfig struct {
	timeout  time.Duration
	interval time.Duration
}

type agentWorkflowTestRunOption func(*agentWorkflowTestRunConfig)

func agentWorkflowTestRunTimeout(timeout time.Duration) agentWorkflowTestRunOption {
	return func(cfg *agentWorkflowTestRunConfig) {
		cfg.timeout = timeout
	}
}

func agentWorkflowTestRunInterval(interval time.Duration) agentWorkflowTestRunOption {
	return func(cfg *agentWorkflowTestRunConfig) {
		cfg.interval = interval
	}
}

func (w *ApplicationService) AgentTestRun(ctx context.Context, req *AgentWorkflowTestRunRequest) (*AgentWorkflowTestRunResponse, error) {
	return runAgentWorkflowTestAndWait(ctx, w, req, agentWorkflowTestRunOptionsFromRequest(req)...)
}

func runAgentWorkflowTestAndWait(ctx context.Context, svc agentWorkflowTestRunService, req *AgentWorkflowTestRunRequest, opts ...agentWorkflowTestRunOption) (*AgentWorkflowTestRunResponse, error) {
	if svc == nil {
		return nil, errors.New("workflow test service is nil")
	}
	if req == nil {
		return nil, errors.New("request is nil")
	}
	workflowID := strings.TrimSpace(req.WorkflowID)
	spaceID := strings.TrimSpace(req.SpaceID)
	if workflowID == "" {
		return nil, errors.New("workflow_id is required")
	}
	if spaceID == "" {
		return nil, errors.New("space_id is required")
	}

	testReq := &modelworkflow.WorkFlowTestRunRequest{
		WorkflowID: workflowID,
		Input:      req.Input,
		SpaceID:    ptr.Of(spaceID),
	}
	if v := strings.TrimSpace(req.BotID); v != "" {
		testReq.BotID = ptr.Of(v)
	}
	if v := strings.TrimSpace(req.ProjectID); v != "" {
		testReq.ProjectID = ptr.Of(v)
	}
	if v := strings.TrimSpace(req.CommitID); v != "" {
		testReq.CommitID = ptr.Of(v)
	}

	testResp, err := svc.TestRun(ctx, testReq)
	if err != nil {
		return nil, err
	}
	if testResp == nil || testResp.GetData() == nil || strings.TrimSpace(testResp.GetData().GetExecuteID()) == "" {
		return nil, errors.New("workflow test_run did not return execute_id")
	}

	cfg := agentWorkflowTestRunConfig{
		timeout:  defaultAgentWorkflowTestRunTimeout,
		interval: defaultAgentWorkflowTestRunInterval,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.timeout <= 0 {
		cfg.timeout = defaultAgentWorkflowTestRunTimeout
	}
	if cfg.interval <= 0 {
		cfg.interval = defaultAgentWorkflowTestRunInterval
	}

	executeID := strings.TrimSpace(testResp.GetData().GetExecuteID())
	processReq := &modelworkflow.GetWorkflowProcessRequest{
		WorkflowID: workflowID,
		SpaceID:    spaceID,
		ExecuteID:  ptr.Of(executeID),
	}
	timeout := time.NewTimer(cfg.timeout)
	defer timeout.Stop()
	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	var last *AgentWorkflowTestRunResponse
	for {
		processResp, err := svc.GetProcess(ctx, processReq)
		if err != nil {
			return nil, err
		}
		var processData *modelworkflow.GetWorkFlowProcessData
		if processResp != nil {
			processData = processResp.GetData()
		}
		last = buildAgentWorkflowTestRunResponse(workflowID, executeID, processData)
		if last.Data.ExecuteStatus != modelworkflow.WorkflowExeStatus_Running {
			return last, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout.C:
			last.Data.Status = "timeout"
			return last, nil
		case <-ticker.C:
		}
	}
}

func agentWorkflowTestRunOptionsFromRequest(req *AgentWorkflowTestRunRequest) []agentWorkflowTestRunOption {
	if req == nil {
		return nil
	}
	var opts []agentWorkflowTestRunOption
	if req.TimeoutMs > 0 {
		opts = append(opts, agentWorkflowTestRunTimeout(clampAgentWorkflowTestRunDuration(time.Duration(req.TimeoutMs)*time.Millisecond, minAgentWorkflowTestRunTimeout, maxAgentWorkflowTestRunTimeout)))
	}
	if req.IntervalMs > 0 {
		opts = append(opts, agentWorkflowTestRunInterval(clampAgentWorkflowTestRunDuration(time.Duration(req.IntervalMs)*time.Millisecond, minAgentWorkflowTestRunInterval, maxAgentWorkflowTestRunInterval)))
	}
	return opts
}

func clampAgentWorkflowTestRunDuration(value, minValue, maxValue time.Duration) time.Duration {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func buildAgentWorkflowTestRunResponse(workflowID, executeID string, process *modelworkflow.GetWorkFlowProcessData) *AgentWorkflowTestRunResponse {
	data := &AgentWorkflowTestRunData{
		WorkflowID:    workflowID,
		ExecuteID:     executeID,
		Status:        "unknown",
		ExecuteStatus: modelworkflow.WorkflowExeStatus_Running,
	}
	if process != nil {
		if v := strings.TrimSpace(process.GetWorkFlowId()); v != "" {
			data.WorkflowID = v
		}
		if v := strings.TrimSpace(process.GetExecuteId()); v != "" {
			data.ExecuteID = v
		}
		data.ExecuteStatus = process.GetExecuteStatus()
		data.Reason = truncateAgentWorkflowTestRunText(process.GetReason())
		data.LastNodeID = process.GetLastNodeID()
		data.LogID = process.GetLogID()
		data.Rate = process.GetRate()
		data.WorkflowExeCost = process.GetWorkflowExeCost()
		data.Nodes = buildAgentWorkflowTestRunNodes(process.GetNodeResults())
		for _, node := range data.Nodes {
			if node.NodeStatus == modelworkflow.NodeExeStatus_Fail || strings.TrimSpace(node.ErrorInfo) != "" {
				data.FailedNodes = append(data.FailedNodes, node)
			}
			if strings.TrimSpace(node.Output) != "" || strings.TrimSpace(node.RawOutput) != "" {
				data.OutputNodes = append(data.OutputNodes, node)
			}
		}
	}
	data.ExecuteStatusText = data.ExecuteStatus.String()
	data.Status = agentWorkflowTestRunStatus(data.ExecuteStatus)

	return &AgentWorkflowTestRunResponse{
		Code: 0,
		Msg:  "",
		Data: data,
	}
}

func buildAgentWorkflowTestRunNodes(results []*modelworkflow.NodeResult) []*AgentWorkflowTestRunNode {
	out := make([]*AgentWorkflowTestRunNode, 0, len(results))
	for _, result := range results {
		if result == nil {
			continue
		}
		out = append(out, &AgentWorkflowTestRunNode{
			NodeID:      result.GetNodeId(),
			NodeType:    result.GetNodeType(),
			NodeName:    result.GetNodeName(),
			NodeStatus:  result.GetNodeStatus(),
			StatusText:  result.GetNodeStatus().String(),
			ErrorInfo:   truncateAgentWorkflowTestRunText(result.GetErrorInfo()),
			ErrorLevel:  result.GetErrorLevel(),
			Input:       truncateAgentWorkflowTestRunText(result.GetInput()),
			Output:      truncateAgentWorkflowTestRunText(result.GetOutput()),
			RawOutput:   truncateAgentWorkflowTestRunText(result.GetRawOutput()),
			NodeExeCost: result.GetNodeExeCost(),
			ExecuteID:   result.GetExecuteId(),
		})
	}
	return out
}

func agentWorkflowTestRunStatus(status modelworkflow.WorkflowExeStatus) string {
	switch status {
	case modelworkflow.WorkflowExeStatus_Running:
		return "running"
	case modelworkflow.WorkflowExeStatus_Success:
		return "success"
	case modelworkflow.WorkflowExeStatus_Fail:
		return "failed"
	case modelworkflow.WorkflowExeStatus_Cancel:
		return "canceled"
	default:
		return "unknown"
	}
}

func truncateAgentWorkflowTestRunText(value string) string {
	if len(value) <= agentWorkflowTestRunTextLimit {
		return value
	}
	return value[:agentWorkflowTestRunTextLimit] + "...[truncated]"
}
