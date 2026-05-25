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

package agentflow

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/getkin/kin-openapi/openapi3"

	"github.com/ynet-dev/ynet-studio/backend/api/model/app/bot_common"
	crossknowledge "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/knowledge"
	knowledgeEntity "github.com/ynet-dev/ynet-studio/backend/domain/knowledge/entity"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/modelmgr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

const (
	knowledgeToolName = "recallKnowledge"
	knowledgeDesc     = `Provides multiple retrieval methods to search for content fragments stored in the knowledge base, helping the large model obtain more accurate and reliable context information`
)

type knowledgeConfig struct {
	knowledgeInfos  []*knowledgeEntity.Knowledge
	knowledgeConfig *bot_common.Knowledge
	Input           *schema.Message
	GetHistory      func() []*schema.Message
	modelInfo       *modelmgr.Model // Model info for query rewrite and NL2SQL
}

func newKnowledgeTool(ctx context.Context, conf *knowledgeConfig) (tool.InvokableTool, error) {
	kl := &knowledgeTool{
		knowledgeConfig: conf.knowledgeConfig,
		Input:           conf.Input,
		GetHistory:      conf.GetHistory,
		modelInfo:       conf.modelInfo,
	}

	customTagsFn := func(name string, t reflect.Type, tag reflect.StructTag,
		schema *openapi3.Schema,
	) error {
		// Process KnowledgeIDs field only
		if name != "KnowledgeIDs" {
			return nil
		}

		// Build knowledge base description
		desc := "Available Knowledge Base List as format knowledge_id: knowledge_name - knowledge_description: \n"
		for _, k := range conf.knowledgeInfos {
			desc += fmt.Sprintf("- %d: %s - %s\n", k.ID, k.Name, k.Description)
		}

		schema.Type = openapi3.TypeArray
		schema.Items = &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Type: openapi3.TypeInteger,
			},
		}
		// Set field descriptions and enumeration values
		schema.Description = desc
		schema.Enum = make([]interface{}, 0, len(conf.knowledgeInfos))
		for _, k := range conf.knowledgeInfos {
			schema.Enum = append(schema.Enum, strconv.FormatInt(k.ID, 10))
		}

		return nil
	}

	return utils.InferTool(knowledgeToolName, knowledgeDesc, kl.Retrieve, utils.WithSchemaCustomizer(customTagsFn))
}

type RetrieveRequest struct {
	KnowledgeIDs []int64 `json:"knowledge_ids" jsonschema:"description="`
}

type knowledgeTool struct {
	knowledgeConfig *bot_common.Knowledge
	Input           *schema.Message
	GetHistory      func() []*schema.Message
	modelInfo       *modelmgr.Model // Model info for query rewrite and NL2SQL
}

func (k *knowledgeTool) Retrieve(ctx context.Context, req *RetrieveRequest) ([]*schema.Document, error) {
	rr, err := genKnowledgeRequest(ctx, req.KnowledgeIDs, k.knowledgeConfig, k.Input.Content, k.GetHistory(), k.modelInfo)
	if err != nil {
		return nil, err
	}

	resp, err := crossknowledge.DefaultSVC().Retrieve(ctx, rr)
	if err != nil {
		return nil, err
	}

	if len(resp.RetrieveSlices) == 0 {
		logs.CtxInfof(ctx, "[retrieve] empty result: query=%q knowledge_ids=%v", k.Input.Content, req.KnowledgeIDs)
	}

	docs, err := convertDocument(ctx, resp.RetrieveSlices)
	if err != nil {
		return nil, err
	}

	return docs, nil
}
