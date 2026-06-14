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

// deepagents_bridge.go is the adk<->project bridge for the experimental
// DeepAgents engine (step 1: "happy path", no interrupt/resume yet).
//
// It drives the deep agent via an adk.Runner and TRANSLATES adk's AgentEvent
// stream into the project's own entity.AgentEvent stream — the exact same type
// the ReAct path produces — so the entire downstream (agent_run_impl.go
// pull/push, persistence, SSE) consumes it unchanged.
//
// NOT YET HANDLED (step 2): interrupt/resume (plugin OAuth, workflow-as-tool),
// checkpoint store adaptation, returnDirectly semantics. Interrupt events are
// logged and skipped for now.

import (
	"context"
	"errors"
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/pkg/safego"
)

// adkAgent is the engine handle stored on AgentRunner. ResumableAgent embeds
// adk.Agent, so it works both for Run (step 1) and Resume (step 2).
type adkAgent = adk.ResumableAgent

// streamExecuteDeep runs the DeepAgents engine and adapts its event stream to
// the project's entity.AgentEvent stream.
func (r *AgentRunner) streamExecuteDeep(ctx context.Context, req *AgentRequest) (
	*schema.StreamReader[*entity.AgentEvent], error,
) {
	sr, sw := schema.Pipe[*entity.AgentEvent](10)

	// Build the adk input messages: history + current input.
	msgs := make([]*schema.Message, 0, len(req.History)+1)
	msgs = append(msgs, req.History...)
	if req.Input != nil {
		msgs = append(msgs, req.Input)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           r.deepAgent,
		EnableStreaming: true,
		CheckPointStore: r.cpStore, // 复用现有 Redis checkpoint，支持中断/恢复
	})

	// checkpoint id：恢复时沿用中断时的 id；新会话生成一个新 id。
	checkpointID := uuid.New().String()
	resuming := false
	if req.ResumeInfo != nil && req.ResumeInfo.InterruptID != "" {
		checkpointID = req.ResumeInfo.InterruptID
		resuming = true
	}

	safego.Go(ctx, func() {
		defer sw.Close()
		var iter *adk.AsyncIterator[*adk.AgentEvent]
		if resuming {
			it, err := runner.Resume(ctx, checkpointID)
			if err != nil {
				logs.CtxErrorf(ctx, "[deepagents] resume failed: %v", err)
				sw.Send(nil, err)
				return
			}
			iter = it
		} else {
			iter = runner.Run(ctx, msgs, adk.WithCheckPointID(checkpointID))
		}
		for {
			ev, ok := iter.Next()
			if !ok {
				return
			}
			if ev == nil {
				continue
			}
			if ev.Err != nil {
				if errors.Is(ev.Err, io.EOF) {
					return
				}
				logs.CtxErrorf(ctx, "[deepagents] event error: %v", ev.Err)
				sw.Send(nil, ev.Err)
				return
			}
			// 中断：翻译成项目 InterruptInfo，带上 checkpointID 供前端恢复时回传。
			if ev.Action != nil && ev.Action.Interrupted != nil {
				sw.Send(&entity.AgentEvent{
					EventType: singleagent.EventTypeOfInterrupt,
					Interrupt: translateDeepInterrupt(ev.Action.Interrupted, checkpointID),
				}, nil)
				return
			}
			translateDeepEvent(ev, sw)
		}
	})

	return sr, nil
}

// translateDeepInterrupt 把 adk 的中断信息翻译成项目的 InterruptInfo。
// InterruptID = checkpointID：前端恢复时回传它，streamExecuteDeep 据此 runner.Resume。
func translateDeepInterrupt(info *adk.InterruptInfo, checkpointID string) *singleagent.InterruptInfo {
	out := &singleagent.InterruptInfo{
		InterruptID:   checkpointID,
		InterruptType: singleagent.InterruptEventType_Question, // 通用"需用户介入后恢复"语义
	}
	if info != nil && len(info.InterruptContexts) > 0 && info.InterruptContexts[0] != nil {
		out.ToolCallID = info.InterruptContexts[0].ID
	}
	return out
}

// translateDeepEvent maps a single adk AgentEvent to the project's AgentEvent(s).
func translateDeepEvent(ev *adk.AgentEvent, sw *schema.StreamWriter[*entity.AgentEvent]) {
	if ev.Output == nil || ev.Output.MessageOutput == nil {
		return
	}
	mo := ev.Output.MessageOutput

	switch mo.Role {
	case schema.Tool:
		// Tool execution result -> ToolsMessage.
		msg := mo.Message
		if msg == nil && mo.MessageStream != nil {
			msg = drainToMessage(mo.MessageStream)
		}
		if msg != nil {
			sw.Send(&entity.AgentEvent{
				EventType:    singleagent.EventTypeOfToolsMessage,
				ToolsMessage: []*schema.Message{msg},
			}, nil)
		}
	default:
		// Assistant output: streaming text -> ChatModelAnswer (the downstream
		// answer handler also detects tool_calls within the stream); a non-stream
		// message with tool_calls -> FuncCall; otherwise wrap as a single-frame
		// answer stream.
		if mo.MessageStream != nil {
			sw.Send(&entity.AgentEvent{
				EventType:       singleagent.EventTypeOfChatModelAnswer,
				ChatModelAnswer: mo.MessageStream,
			}, nil)
			return
		}
		if mo.Message == nil {
			return
		}
		if len(mo.Message.ToolCalls) > 0 {
			sw.Send(&entity.AgentEvent{
				EventType: singleagent.EventTypeOfFuncCall,
				FuncCall:  mo.Message,
			}, nil)
			return
		}
		sw.Send(&entity.AgentEvent{
			EventType:       singleagent.EventTypeOfChatModelAnswer,
			ChatModelAnswer: schema.StreamReaderFromArray([]*schema.Message{mo.Message}),
		}, nil)
	}
}

// drainToMessage concatenates a message stream into a single message
// (content concatenated; role/tool metadata taken from the first frame).
func drainToMessage(s *schema.StreamReader[*schema.Message]) *schema.Message {
	defer s.Close()
	var out *schema.Message
	for {
		frame, err := s.Recv()
		if err != nil {
			break
		}
		if frame == nil {
			continue
		}
		if out == nil {
			cp := *frame
			out = &cp
			continue
		}
		out.Content += frame.Content
	}
	return out
}
