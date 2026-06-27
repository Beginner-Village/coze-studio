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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	modelworkflow "github.com/ynet-dev/ynet-studio/backend/api/model/workflow"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
)

func TestRunAgentWorkflowTestAndWaitReturnsTerminalFailureDetails(t *testing.T) {
	fake := &fakeAgentWorkflowTestService{
		testRun: &modelworkflow.WorkFlowTestRunResponse{
			Data: &modelworkflow.WorkFlowTestRunData{
				WorkflowID: "wf_1",
				ExecuteID:  "exe_1",
			},
		},
		processes: []*modelworkflow.GetWorkflowProcessResponse{
			{
				Data: &modelworkflow.GetWorkFlowProcessData{
					ExecuteStatus: modelworkflow.WorkflowExeStatus_Running,
				},
			},
			{
				Data: &modelworkflow.GetWorkFlowProcessData{
					WorkFlowId:       "wf_1",
					ExecuteId:        "exe_1",
					ExecuteStatus:    modelworkflow.WorkflowExeStatus_Fail,
					Reason:           ptr.Of("workflow failed"),
					LogID:            "log_1",
					WorkflowExeCost:  "0.123s",
					Rate:             "0.50",
					LastNodeID:       ptr.Of("node_bad"),
					ExeHistoryStatus: modelworkflow.WorkflowExeHistoryStatus_HasHistory,
					NodeResults: []*modelworkflow.NodeResult{
						{
							NodeId:      "node_ok",
							NodeType:    "Start",
							NodeName:    "开始",
							NodeStatus:  modelworkflow.NodeExeStatus_Success,
							Output:      `{"input":"hello"}`,
							NodeExeCost: "0.001s",
						},
						{
							NodeId:      "node_bad",
							NodeType:    "Output",
							NodeName:    "输出业务名称",
							NodeStatus:  modelworkflow.NodeExeStatus_Fail,
							ErrorInfo:   "引用变量不存在",
							Input:       `{"content":"{{category}}"}`,
							NodeExeCost: "0.002s",
						},
					},
				},
			},
		},
	}

	resp, err := runAgentWorkflowTestAndWait(context.Background(), fake, &AgentWorkflowTestRunRequest{
		WorkflowID: "wf_1",
		SpaceID:    "space_1",
		Input:      map[string]string{"input": "hello"},
	}, agentWorkflowTestRunTimeout(200*time.Millisecond), agentWorkflowTestRunInterval(time.Millisecond))

	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	assert.Equal(t, "failed", resp.Data.Status)
	assert.Equal(t, modelworkflow.WorkflowExeStatus_Fail, resp.Data.ExecuteStatus)
	assert.Equal(t, "workflow failed", resp.Data.Reason)
	assert.Equal(t, "node_bad", resp.Data.LastNodeID)
	require.Len(t, resp.Data.FailedNodes, 1)
	assert.Equal(t, "node_bad", resp.Data.FailedNodes[0].NodeID)
	assert.Equal(t, "输出业务名称", resp.Data.FailedNodes[0].NodeName)
	assert.Equal(t, "引用变量不存在", resp.Data.FailedNodes[0].ErrorInfo)
	require.Len(t, resp.Data.OutputNodes, 1)
	assert.Equal(t, "node_ok", resp.Data.OutputNodes[0].NodeID)
	assert.Equal(t, "exe_1", fake.processRequests[0].GetExecuteID())
	assert.Equal(t, "space_1", fake.testRunRequest.GetSpaceID())
	assert.Equal(t, map[string]string{"input": "hello"}, fake.testRunRequest.Input)
}

func TestRunAgentWorkflowTestAndWaitReturnsTimeoutSnapshot(t *testing.T) {
	fake := &fakeAgentWorkflowTestService{
		testRun: &modelworkflow.WorkFlowTestRunResponse{
			Data: &modelworkflow.WorkFlowTestRunData{
				WorkflowID: "wf_1",
				ExecuteID:  "exe_1",
			},
		},
		processes: []*modelworkflow.GetWorkflowProcessResponse{
			{
				Data: &modelworkflow.GetWorkFlowProcessData{
					WorkFlowId:       "wf_1",
					ExecuteId:        "exe_1",
					ExecuteStatus:    modelworkflow.WorkflowExeStatus_Running,
					WorkflowExeCost:  "0.050s",
					Rate:             "0.10",
					ExeHistoryStatus: modelworkflow.WorkflowExeHistoryStatus_HasHistory,
				},
			},
		},
	}

	resp, err := runAgentWorkflowTestAndWait(context.Background(), fake, &AgentWorkflowTestRunRequest{
		WorkflowID: "wf_1",
		SpaceID:    "space_1",
	}, agentWorkflowTestRunTimeout(3*time.Millisecond), agentWorkflowTestRunInterval(time.Millisecond))

	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	assert.Equal(t, "timeout", resp.Data.Status)
	assert.Equal(t, modelworkflow.WorkflowExeStatus_Running, resp.Data.ExecuteStatus)
	assert.Equal(t, "0.10", resp.Data.Rate)
}

type fakeAgentWorkflowTestService struct {
	testRun         *modelworkflow.WorkFlowTestRunResponse
	testRunErr      error
	testRunRequest  *modelworkflow.WorkFlowTestRunRequest
	processes       []*modelworkflow.GetWorkflowProcessResponse
	processErr      error
	processRequests []*modelworkflow.GetWorkflowProcessRequest
}

func (f *fakeAgentWorkflowTestService) TestRun(ctx context.Context, req *modelworkflow.WorkFlowTestRunRequest) (*modelworkflow.WorkFlowTestRunResponse, error) {
	f.testRunRequest = req
	return f.testRun, f.testRunErr
}

func (f *fakeAgentWorkflowTestService) GetProcess(ctx context.Context, req *modelworkflow.GetWorkflowProcessRequest) (*modelworkflow.GetWorkflowProcessResponse, error) {
	f.processRequests = append(f.processRequests, req)
	if f.processErr != nil {
		return nil, f.processErr
	}
	if len(f.processes) == 0 {
		return &modelworkflow.GetWorkflowProcessResponse{Data: &modelworkflow.GetWorkFlowProcessData{}}, nil
	}
	resp := f.processes[0]
	if len(f.processes) > 1 {
		f.processes = f.processes[1:]
	}
	return resp, nil
}
