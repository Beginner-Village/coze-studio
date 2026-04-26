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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/domain/knowledge/entity"
	contract "github.com/ynet-dev/ynet-studio/backend/infra/contract/document/parser"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

func ParseJSON(config *contract.Config) ParseFn {
	return func(ctx context.Context, reader io.Reader, opts ...parser.Option) (docs []*schema.Document, err error) {
		counted := &countingReader{r: io.LimitReader(reader, entity.MaxOtherFileSize+1), max: entity.MaxOtherFileSize}

		dec := json.NewDecoder(counted)
		tok, err := dec.Token()
		if err != nil {
			if errors.Is(err, errFileTooLarge) {
				return nil, errorx.New(errno.ErrKnowledgeFileTooLargeCode,
					errorx.KVf("msg", "JSON file size exceeds %d bytes", entity.MaxOtherFileSize))
			}
			return nil, err
		}
		if d, ok := tok.(json.Delim); !ok || d != '[' {
			return nil, fmt.Errorf("[ParseJSON] expected JSON array")
		}

		var header []string
		if config.ParsingStrategy.IsAppend {
			for _, col := range config.ParsingStrategy.Columns {
				header = append(header, col.Name)
			}
		}

		iter := &streamingJSONIterator{
			dec:    dec,
			header: header,
		}

		return parseByRowIterator(iter, config, opts...)
	}
}

// streamingJSONIterator 通过 json.Decoder 流式逐元素读取 JSON 数组。
// 避免在 parse 阶段把整个数组一次性加载到内存。
type streamingJSONIterator struct {
	dec        *json.Decoder
	header     []string
	headerSent bool
	emitted    bool
	firstRow   map[string]string
}

func (s *streamingJSONIterator) NextRow() (row []string, end bool, err error) {
	if !s.headerSent {
		if len(s.header) == 0 {
			// 尚未拿到 header：peek 第一条 record 推断 keys。
			if !s.dec.More() {
				return nil, false, errors.New("[ParseJSON] json data is empty")
			}
			var first map[string]string
			if err := s.dec.Decode(&first); err != nil {
				return nil, false, mapStreamError(err)
			}
			for k := range first {
				s.header = append(s.header, k)
			}
			s.headerSent = true
			s.emitted = true
			s.firstRow = first
			return append([]string{}, s.header...), false, nil
		}
		s.headerSent = true
		return append([]string{}, s.header...), false, nil
	}

	if s.firstRow != nil {
		raw := s.firstRow
		s.firstRow = nil
		return s.rowFromMap(raw), false, nil
	}

	if !s.dec.More() {
		return nil, true, nil
	}
	var item map[string]string
	if err := s.dec.Decode(&item); err != nil {
		return nil, false, mapStreamError(err)
	}
	return s.rowFromMap(item), false, nil
}

func mapStreamError(err error) error {
	if errors.Is(err, errFileTooLarge) {
		return errorx.New(errno.ErrKnowledgeFileTooLargeCode,
			errorx.KVf("msg", "JSON file size exceeds %d bytes", entity.MaxOtherFileSize))
	}
	return err
}

func (s *streamingJSONIterator) rowFromMap(item map[string]string) []string {
	row := make([]string, 0, len(s.header))
	for _, h := range s.header {
		row = append(row, item[h])
	}
	return row
}

var errFileTooLarge = errors.New("file too large")

type countingReader struct {
	r   io.Reader
	max int64
	cnt int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.cnt += int64(n)
	if c.cnt > c.max {
		return n, errFileTooLarge
	}
	return n, err
}
