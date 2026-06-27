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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
	modelworkflow "github.com/ynet-dev/ynet-studio/backend/api/model/workflow"
	crossworkflow "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/workflow"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

const (
	workflowCanvasBackendTestRunTimeout  = 60 * time.Second
	workflowCanvasBackendTestRunInterval = time.Second
	workflowCanvasBackendTextLimit       = 4000
)

type workflowCanvasBackendTestRunFunc func(context.Context, workflowCanvasBackendTestRunRequest) (*workflowCanvasBackendTestRunResponse, error)

type workflowCanvasBackendTestRunRequest struct {
	WorkflowID string            `json:"workflow_id"`
	SpaceID    string            `json:"space_id"`
	Input      map[string]string `json:"input,omitempty"`
}

type workflowCanvasBackendTestRunResponse struct {
	Data *workflowCanvasBackendTestRunData `json:"data"`
}

type workflowCanvasBackendTestRunData struct {
	WorkflowID        string                              `json:"workflow_id"`
	ExecuteID         string                              `json:"execute_id"`
	Status            string                              `json:"status"`
	ExecuteStatus     modelworkflow.WorkflowExeStatus     `json:"execute_status"`
	ExecuteStatusText string                              `json:"execute_status_text"`
	Reason            string                              `json:"reason,omitempty"`
	LastNodeID        string                              `json:"last_node_id,omitempty"`
	LogID             string                              `json:"log_id,omitempty"`
	Rate              string                              `json:"rate,omitempty"`
	WorkflowExeCost   string                              `json:"workflow_exe_cost,omitempty"`
	Nodes             []*workflowCanvasBackendTestRunNode `json:"nodes,omitempty"`
	FailedNodes       []*workflowCanvasBackendTestRunNode `json:"failed_nodes,omitempty"`
	OutputNodes       []*workflowCanvasBackendTestRunNode `json:"output_nodes,omitempty"`
}

type workflowCanvasBackendTestRunNode struct {
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

func newWorkflowCanvasBackendTestRunFunc(deps superAgentToolDeps) workflowCanvasBackendTestRunFunc {
	return func(ctx context.Context, req workflowCanvasBackendTestRunRequest) (*workflowCanvasBackendTestRunResponse, error) {
		return runWorkflowCanvasBackendTest(ctx, deps, req)
	}
}

func runWorkflowCanvasBackendTest(ctx context.Context, deps superAgentToolDeps, req workflowCanvasBackendTestRunRequest) (*workflowCanvasBackendTestRunResponse, error) {
	svc := crossworkflow.DefaultSVC()
	if svc == nil {
		return nil, errors.New("workflow service is not available")
	}
	if deps.UserID == 0 {
		return nil, errors.New("user_id is required")
	}
	workflowID, err := strconv.ParseInt(strings.TrimSpace(req.WorkflowID), 10, 64)
	if err != nil || workflowID <= 0 {
		return nil, fmt.Errorf("invalid workflow_id: %s", req.WorkflowID)
	}
	spaceID, err := strconv.ParseInt(strings.TrimSpace(req.SpaceID), 10, 64)
	if err != nil || spaceID <= 0 {
		return nil, fmt.Errorf("invalid space_id: %s", req.SpaceID)
	}
	if deps.SpaceID > 0 && deps.SpaceID != spaceID {
		return nil, fmt.Errorf("workflow space mismatch: tool space=%d request space=%d", deps.SpaceID, spaceID)
	}

	cfg := workflowModel.ExecuteConfig{
		ID:           workflowID,
		From:         workflowModel.FromDraft,
		Operator:     deps.UserID,
		Mode:         workflowModel.ExecuteModeDebug,
		ConnectorID:  consts.CozeConnectorID,
		ConnectorUID: strconv.FormatInt(deps.UserID, 10),
		TaskType:     workflowModel.TaskTypeForeground,
		SyncPattern:  workflowModel.SyncPatternAsync,
		BizType:      workflowModel.BizTypeWorkflow,
		Cancellable:  true,
	}
	if deps.AgentID > 0 {
		cfg.AgentID = ptr.Of(deps.AgentID)
	}

	exeID, err := svc.AsyncExecute(ctx, cfg, workflowCanvasInputToAny(req.Input))
	if err != nil {
		return nil, err
	}

	timeout := time.NewTimer(workflowCanvasBackendTestRunTimeout)
	defer timeout.Stop()
	ticker := time.NewTicker(workflowCanvasBackendTestRunInterval)
	defer ticker.Stop()

	var last *workflowCanvasBackendTestRunResponse
	for {
		execution, err := svc.GetExecution(ctx, &entity.WorkflowExecution{
			ID:         exeID,
			WorkflowID: workflowID,
		}, true)
		if err != nil {
			return nil, err
		}
		last = buildWorkflowCanvasBackendTestRunResponse(workflowID, exeID, execution)
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

func workflowCanvasInputToAny(input map[string]string) map[string]any {
	out := make(map[string]any, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func buildWorkflowCanvasBackendTestRunResponse(workflowID, executeID int64, execution *entity.WorkflowExecution) *workflowCanvasBackendTestRunResponse {
	data := &workflowCanvasBackendTestRunData{
		WorkflowID:    strconv.FormatInt(workflowID, 10),
		ExecuteID:     strconv.FormatInt(executeID, 10),
		Status:        "running",
		ExecuteStatus: modelworkflow.WorkflowExeStatus_Running,
	}
	if execution != nil {
		data.WorkflowID = strconv.FormatInt(execution.WorkflowID, 10)
		data.ExecuteID = strconv.FormatInt(execution.ID, 10)
		status := execution.Status
		if status == entity.WorkflowInterrupted {
			status = entity.WorkflowRunning
		}
		data.ExecuteStatus = modelworkflow.WorkflowExeStatus(status)
		data.ExecuteStatusText = data.ExecuteStatus.String()
		data.Status = workflowCanvasBackendStatus(data.ExecuteStatus)
		data.WorkflowExeCost = fmt.Sprintf("%.3fs", execution.Duration.Seconds())
		data.LogID = execution.LogID
		if execution.FailReason != nil {
			data.Reason = truncateWorkflowCanvasBackendText(*execution.FailReason)
		}
		data.Nodes = buildWorkflowCanvasBackendNodes(execution)
		for _, node := range data.Nodes {
			if node.NodeStatus == modelworkflow.NodeExeStatus_Fail || strings.TrimSpace(node.ErrorInfo) != "" {
				data.FailedNodes = append(data.FailedNodes, node)
				data.LastNodeID = node.NodeID
			}
			if strings.TrimSpace(node.Output) != "" || strings.TrimSpace(node.RawOutput) != "" {
				data.OutputNodes = append(data.OutputNodes, node)
			}
		}
		if execution.NodeCount > 0 {
			successNum := 0
			for _, node := range data.Nodes {
				if node.NodeStatus == modelworkflow.NodeExeStatus_Success {
					successNum++
				}
			}
			data.Rate = fmt.Sprintf("%.2f", float64(successNum)/float64(execution.NodeCount))
		}
	}
	if data.ExecuteStatusText == "" {
		data.ExecuteStatusText = data.ExecuteStatus.String()
	}
	return &workflowCanvasBackendTestRunResponse{Data: data}
}

func buildWorkflowCanvasBackendNodes(execution *entity.WorkflowExecution) []*workflowCanvasBackendTestRunNode {
	if execution == nil {
		return nil
	}
	out := make([]*workflowCanvasBackendTestRunNode, 0, len(execution.NodeExecutions))
	for _, node := range execution.NodeExecutions {
		if node == nil {
			continue
		}
		result := &workflowCanvasBackendTestRunNode{
			NodeID:      node.NodeID,
			NodeName:    node.NodeName,
			NodeType:    string(node.NodeType),
			NodeStatus:  modelworkflow.NodeExeStatus(node.Status),
			StatusText:  modelworkflow.NodeExeStatus(node.Status).String(),
			NodeExeCost: fmt.Sprintf("%.3fs", node.Duration.Seconds()),
			ExecuteID:   strconv.FormatInt(node.ExecuteID, 10),
		}
		if node.Input != nil {
			result.Input = truncateWorkflowCanvasBackendText(*node.Input)
		}
		if node.Output != nil {
			result.Output = truncateWorkflowCanvasBackendText(*node.Output)
		}
		if node.RawOutput != nil {
			result.RawOutput = truncateWorkflowCanvasBackendText(*node.RawOutput)
		}
		if node.ErrorInfo != nil {
			result.ErrorInfo = truncateWorkflowCanvasBackendText(*node.ErrorInfo)
		}
		if node.ErrorLevel != nil {
			result.ErrorLevel = *node.ErrorLevel
		}
		out = append(out, result)
	}
	return out
}

func workflowCanvasBackendStatus(status modelworkflow.WorkflowExeStatus) string {
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

func truncateWorkflowCanvasBackendText(value string) string {
	if len(value) <= workflowCanvasBackendTextLimit {
		return value
	}
	return value[:workflowCanvasBackendTextLimit] + "...[truncated]"
}
