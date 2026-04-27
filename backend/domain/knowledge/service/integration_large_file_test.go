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
	"runtime"
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

// TestIntegration_LargeTextStreaming 验证 50MB 文本解析时的内存增量在合理范围内。
// streamingTextChunker 直接走 bufio.Scanner，理论上常驻内存约几个 chunk。
// 注意 ParseText 当前仍走 ChunkCustom（保留原行为），所以这里直接验证 streamingTextChunker。
func TestIntegration_LargeTextStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// 50MB 文本
	bigText := strings.Repeat("hello world\n", 50*1024*1024/12)

	chunker := builtin.NewStreamingTextChunker(4096, 0)
	docs, err := chunker.Chunk(context.Background(), strings.NewReader(bigText))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(docs), 100)

	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	deltaMB := int64(memAfter.Alloc-memBefore.Alloc) / 1024 / 1024
	t.Logf("memory delta: %d MB, chunks: %d", deltaMB, len(docs))
	// 严格意义的内存上界包括 docs 切片本身，每条 chunk 是 string copy。50MB 文本切成 4KB chunk = ~13k chunks * 4KB = 50MB string memory.
	// 这里阈值放宽到 200MB，主要验证不发生 OOM 数量级膨胀。
	assert.LessOrEqual(t, deltaMB, int64(200), "memory delta should be under 200MB, got %d MB", deltaMB)
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
