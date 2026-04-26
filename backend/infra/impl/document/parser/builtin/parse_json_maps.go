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
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document"
	contract "github.com/ynet-dev/ynet-studio/backend/infra/contract/document/parser"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

func ParseJSONMaps(config *contract.Config) ParseFn {
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
			return nil, fmt.Errorf("[ParseJSONMaps] expected JSON array")
		}

		if config.ParsingStrategy == nil {
			config.ParsingStrategy = &contract.ParsingStrategy{
				HeaderLine:    0,
				DataStartLine: 1,
				RowsCount:     0,
			}
		}

		iter := &customContentContainer{
			dec:        dec,
			curColumns: config.ParsingStrategy.Columns,
		}

		newConfig := &contract.Config{
			FileExtension: config.FileExtension,
			ParsingStrategy: &contract.ParsingStrategy{
				SheetID:       config.ParsingStrategy.SheetID,
				HeaderLine:    0,
				DataStartLine: 1,
				RowsCount:     0,
				IsAppend:      config.ParsingStrategy.IsAppend,
				Columns:       config.ParsingStrategy.Columns,
			},
			ChunkingStrategy: config.ChunkingStrategy,
		}

		return parseByRowIterator(iter, newConfig, opts...)
	}
}

type customContentContainer struct {
	dec        *json.Decoder
	colIdx     map[string]int
	curColumns []*document.Column
	pending    map[string]string
}

func (c *customContentContainer) NextRow() (row []string, end bool, err error) {
	if c.colIdx == nil {
		if !c.dec.More() {
			return nil, false, fmt.Errorf("[customContentContainer] data is nil")
		}
		var headerRow map[string]string
		if err := c.dec.Decode(&headerRow); err != nil {
			return nil, false, mapStreamError(err)
		}

		founded := make(map[string]struct{})
		colIdx := make(map[string]int, len(headerRow))

		for _, col := range c.curColumns {
			name := col.Name
			if _, found := headerRow[name]; found {
				founded[name] = struct{}{}
				colIdx[name] = len(colIdx)
				row = append(row, name)
			}
		}
		for name := range headerRow {
			if _, found := founded[name]; !found {
				colIdx[name] = len(colIdx)
				row = append(row, name)
			}
		}

		c.colIdx = colIdx
		c.pending = headerRow
		return row, false, nil
	}

	var content map[string]string
	if c.pending != nil {
		content = c.pending
		c.pending = nil
	} else {
		if !c.dec.More() {
			return nil, true, nil
		}
		if err := c.dec.Decode(&content); err != nil {
			return nil, false, mapStreamError(err)
		}
	}

	row = make([]string, len(content))
	for k, v := range content {
		idx, found := c.colIdx[k]
		if !found {
			return nil, false, fmt.Errorf("[customContentContainer] column not found, name=%s", k)
		}

		row[idx] = v
	}

	return row, false, nil
}
