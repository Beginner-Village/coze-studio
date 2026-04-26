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
	"fmt"
	"io"

	"github.com/cloudwego/eino/components/document/parser"
	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/domain/knowledge/entity"
	contract "github.com/ynet-dev/ynet-studio/backend/infra/contract/document/parser"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

func ParseText(config *contract.Config) ParseFn {
	return func(ctx context.Context, reader io.Reader, opts ...parser.Option) (docs []*schema.Document, err error) {
		limited := io.LimitReader(reader, entity.MaxOtherFileSize+1)
		content, err := io.ReadAll(limited)
		if err != nil {
			return nil, err
		}
		if int64(len(content)) > entity.MaxOtherFileSize {
			return nil, errorx.New(errno.ErrKnowledgeFileTooLargeCode,
				errorx.KVf("msg", "file size exceeds %d bytes", entity.MaxOtherFileSize))
		}

		switch config.ChunkingStrategy.ChunkType {
		case contract.ChunkTypeCustom, contract.ChunkTypeDefault:
			docs, err = ChunkCustom(ctx, string(content), config, opts...)
		default:
			return nil, fmt.Errorf("[ParseText] chunk type not support, type=%d", config.ChunkingStrategy.ChunkType)
		}
		if err != nil {
			return nil, err
		}

		return docs, nil
	}
}
