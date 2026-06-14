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

package chatmodel

import (
	"context"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

// errEnrichModel 包装内层模型，把 Generate/Stream 的错误富化为 ModelCallError。
type errEnrichModel struct {
	inner chatmodel.ToolCallingChatModel
}

func withErrorEnrichment(m chatmodel.ToolCallingChatModel) chatmodel.ToolCallingChatModel {
	if m == nil {
		return nil
	}
	return &errEnrichModel{inner: m}
}

func (m *errEnrichModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	out, err := m.inner.Generate(ctx, input, opts...)
	if err != nil {
		return nil, wrapModelError(err)
	}
	return out, nil
}

func (m *errEnrichModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	out, err := m.inner.Stream(ctx, input, opts...)
	if err != nil {
		return nil, wrapModelError(err)
	}
	return out, nil
}

func (m *errEnrichModel) WithTools(tools []*schema.ToolInfo) (chatmodel.ToolCallingChatModel, error) {
	inner, err := m.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &errEnrichModel{inner: inner}, nil
}
