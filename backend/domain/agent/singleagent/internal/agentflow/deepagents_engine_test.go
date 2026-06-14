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
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepAgentsEnabled(t *testing.T) {
	t.Run("unset defaults to false (ReAct)", func(t *testing.T) {
		t.Setenv("AGENT_ENGINE", "")
		assert.False(t, deepAgentsEnabled())
	})

	t.Run("other value is false (ReAct)", func(t *testing.T) {
		t.Setenv("AGENT_ENGINE", "react")
		assert.False(t, deepAgentsEnabled())
	})

	t.Run("deepagents enables", func(t *testing.T) {
		t.Setenv("AGENT_ENGINE", "deepagents")
		assert.True(t, deepAgentsEnabled())
	})

	t.Run("case-insensitive with surrounding spaces", func(t *testing.T) {
		t.Setenv("AGENT_ENGINE", "  DeepAgents  ")
		assert.True(t, deepAgentsEnabled())
	})
}

// fakeToolCallingModel is a no-op chat model satisfying chatmodel.ToolCallingChatModel,
// just enough to let deep.New construct an agent without a real LLM.
type fakeToolCallingModel struct{}

func (f *fakeToolCallingModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage("ok", nil), nil
}

func (f *fakeToolCallingModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](1)
	sw.Send(schema.AssistantMessage("ok", nil), nil)
	sw.Close()
	return sr, nil
}

func (f *fakeToolCallingModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return f, nil
}

func TestBuildDeepAgent_DoesNotPanic(t *testing.T) {
	ctx := context.Background()

	echoTool, err := utils.InferTool("echo", "echo back the input", func(_ context.Context, in struct {
		Text string `json:"text"`
	}) (string, error) {
		return in.Text, nil
	})
	require.NoError(t, err)

	agentTools := []tool.BaseTool{echoTool}

	agent, err := buildDeepAgent(ctx, &Config{}, &fakeToolCallingModel{}, agentTools)
	require.NoError(t, err)
	require.NotNil(t, agent)
	assert.NotEmpty(t, agent.Name(ctx))
}
