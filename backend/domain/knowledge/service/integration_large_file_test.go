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

//go:build integration

package service

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ynet-dev/ynet-studio/backend/domain/knowledge/entity"
	contract "github.com/ynet-dev/ynet-studio/backend/infra/contract/document/parser"
	"github.com/ynet-dev/ynet-studio/backend/infra/impl/document/parser/builtin"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// TestIntegration_RejectsOversizedUpload 验证 parser 层拒绝超大文件。
// HTTP 入口的拒绝由 Task 2 的 unit-level 检查覆盖。
func TestIntegration_RejectsOversizedUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	bigPayload := bytes.Repeat([]byte("x"), int(entity.MaxOtherFileSize)+10)
	cfg := &contract.Config{
		ChunkingStrategy: &contract.ChunkingStrategy{ChunkType: contract.ChunkTypeDefault, ChunkSize: 1024},
	}
	pfn := builtin.ParseText(cfg)
	docs, err := pfn(context.Background(), bytes.NewReader(bigPayload), parser.WithExtraMeta(nil))

	require.Error(t, err)
	require.Nil(t, docs)
	assert.Contains(t, err.Error(), "exceeds")
	// errorx code propagates
	assert.Contains(t, err.Error(), strings.TrimPrefix(""+itoa(errno.ErrKnowledgeFileTooLargeCode), ""))
}

// itoa 辅助：避免引入额外包。
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte(i%10) + '0'}, digits...)
		i /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
