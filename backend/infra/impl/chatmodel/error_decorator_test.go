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
	"errors"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

type fakeModel struct{ err error }

func (f *fakeModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return nil, f.err
}
func (f *fakeModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, f.err
}
func (f *fakeModel) WithTools(tools []*schema.ToolInfo) (chatmodel.ToolCallingChatModel, error) {
	return f, nil
}

func TestDecoratorEnrichesGenerateError(t *testing.T) {
	raw := errors.New("error, status code: 400, message: invalid request")
	m := withErrorEnrichment(&fakeModel{err: raw})
	_, err := m.Generate(context.Background(), nil)
	mce, ok := chatmodel.AsModelCallError(err)
	if !ok {
		t.Fatalf("expected ModelCallError, got %T: %v", err, err)
	}
	if mce.HTTPStatus != 400 {
		t.Fatalf("want status 400, got %d", mce.HTTPStatus)
	}
}
