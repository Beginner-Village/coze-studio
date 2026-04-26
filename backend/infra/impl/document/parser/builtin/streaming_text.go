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

package builtin

import (
	"bufio"
	"context"
	"io"

	"github.com/cloudwego/eino/schema"
)

// streamingTextChunker 用 bufio.Scanner 按定长 chunk 流式读取文本，避免一次性加载整文件到内存。
type streamingTextChunker struct {
	chunkSize int
	overlap   int
}

func (c *streamingTextChunker) Chunk(ctx context.Context, reader io.Reader) ([]*schema.Document, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	scanner.Split(splitByChunkSize(c.chunkSize))

	var docs []*schema.Document
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			continue
		}
		docs = append(docs, &schema.Document{Content: text})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return docs, nil
}

// splitByChunkSize 返回一个按 chunkSize 字节切分的 SplitFunc，并避免切到 UTF-8 字符中间。
func splitByChunkSize(chunkSize int) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if len(data) >= chunkSize {
			end := chunkSize
			// 如果切到的位置正好是 UTF-8 多字节字符中间，回退到字符起点。
			for end > 0 && end < len(data) && (data[end]&0xC0) == 0x80 {
				end--
			}
			return end, data[:end], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
}
