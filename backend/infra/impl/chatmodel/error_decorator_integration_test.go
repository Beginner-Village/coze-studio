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
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/chatmodel"
)

// TestFactoryWrapsRealEinoModelError exercises the full factory path against the
// real eino-ext openai client: a bad base_url makes Generate fail with a transport
// error, and the factory decorator must surface it as a *chatmodel.ModelCallError.
// This closes the integration gap that the fake-model unit test can't cover
// (real eino client error -> decorator -> ModelCallError chain).
func TestFactoryWrapsRealEinoModelError(t *testing.T) {
	f := NewDefaultFactory()
	m, err := f.CreateChatModel(context.Background(), chatmodel.ProtocolOpenAI, &chatmodel.Config{
		BaseURL: "http://127.0.0.1:1/v1", // unreachable -> instant connection refused
		APIKey:  "test-key",
		Model:   "gpt-4o",
		Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("CreateChatModel should succeed (lazy client), got: %v", err)
	}

	_, genErr := m.Generate(context.Background(), []*schema.Message{
		schema.UserMessage("hi"),
	})
	if genErr == nil {
		t.Fatal("expected Generate to fail against unreachable base_url, got nil")
	}

	mce, ok := chatmodel.AsModelCallError(genErr)
	if !ok {
		t.Fatalf("expected error chain to contain *ModelCallError, got %T: %v", genErr, genErr)
	}
	// Transport failure carries no HTTP status, but the provider/transport message
	// must be preserved (non-empty) so debug mode shows a concrete cause.
	if mce.ProviderMessage == "" && mce.Raw == nil {
		t.Fatalf("ModelCallError carries no detail: %+v", mce)
	}
	t.Logf("enriched error: %s", mce.Error())
}
