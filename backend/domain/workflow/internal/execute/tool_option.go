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

package execute

import (
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	workflowModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/workflow"
	"github.com/ynet-dev/ynet-studio/backend/domain/workflow/entity"
)

type workflowToolOption struct {
	resumeReq            *entity.ResumeRequest
	sw                   *schema.StreamWriter[*entity.Message]
	exeCfg               workflowModel.ExecuteConfig
	allInterruptEvents   map[string]*entity.ToolInterruptEvent
	parentTokenCollector *TokenCollector
}

func WithResume(req *entity.ResumeRequest, all map[string]*entity.ToolInterruptEvent) tool.Option {
	return tool.WrapImplSpecificOptFn(func(opts *workflowToolOption) {
		opts.resumeReq = req
		opts.allInterruptEvents = all
	})
}

func WithIntermediateStreamWriter(sw *schema.StreamWriter[*entity.Message]) tool.Option {
	return tool.WrapImplSpecificOptFn(func(opts *workflowToolOption) {
		opts.sw = sw
	})
}

func WithExecuteConfig(cfg workflowModel.ExecuteConfig) tool.Option {
	return tool.WrapImplSpecificOptFn(func(opts *workflowToolOption) {
		opts.exeCfg = cfg
	})
}

func GetResumeRequest(opts ...tool.Option) (*entity.ResumeRequest, map[string]*entity.ToolInterruptEvent) {
	opt := tool.GetImplSpecificOptions(&workflowToolOption{}, opts...)
	return opt.resumeReq, opt.allInterruptEvents
}

func GetIntermediateStreamWriter(opts ...tool.Option) *schema.StreamWriter[*entity.Message] {
	opt := tool.GetImplSpecificOptions(&workflowToolOption{}, opts...)
	return opt.sw
}

func GetExecuteConfig(opts ...tool.Option) workflowModel.ExecuteConfig {
	opt := tool.GetImplSpecificOptions(&workflowToolOption{}, opts...)
	return opt.exeCfg
}

// WithMessagePipe returns an Option which is meant to be passed to the tool workflow,
// a StreamReader to read the messages from the tool workflow, and a StreamWriter that
// the caller should close when all tool invocations are done.
// This Option will apply to ALL workflow tools to be executed by eino's ToolsNode.
// The workflow tools will emit messages to this stream.
// IMPORTANT: The caller is responsible for closing the returned StreamWriter when done,
// otherwise the StreamReader will block forever waiting for EOF.
func WithMessagePipe() (compose.Option, *schema.StreamReader[*entity.Message], *schema.StreamWriter[*entity.Message]) {
	sr, sw := schema.Pipe[*entity.Message](10)
	opt := compose.WithToolsNodeOption(compose.WithToolOption(WithIntermediateStreamWriter(sw)))
	return opt, sr, sw
}
