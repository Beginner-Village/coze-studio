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

package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	ollamaEmb "github.com/cloudwego/eino-ext/components/embedding/ollama"
	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"github.com/volcengine/volc-sdk-golang/service/vikingdb"
	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/application/internal"
	"github.com/ynet-dev/ynet-studio/backend/application/search"
	embeddingService "github.com/ynet-dev/ynet-studio/backend/domain/embedding/service"
	knowledgeImpl "github.com/ynet-dev/ynet-studio/backend/domain/knowledge/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/cache"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/nl2sql"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/ocr"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/parser"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/searchstore"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/embedding"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/messages2query"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/rdb"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/storage"
	chatmodelImpl "github.com/ynet-dev/ynet-studio/backend/infra/impl/chatmodel"
	builtinNL2SQL "github.com/ynet-dev/ynet-studio/backend/infra/impl/document/nl2sql/builtin"
	"github.com/ynet-dev/ynet-studio/backend/infra/impl/document/rerank/rrf"
	rerankProvider "github.com/ynet-dev/ynet-studio/backend/infra/impl/rerank/provider"
	rerankRepo "github.com/ynet-dev/ynet-studio/backend/infra/impl/rerank/repository"
	rerankService "github.com/ynet-dev/ynet-studio/backend/domain/rerank/service"
	ssmilvus "github.com/ynet-dev/ynet-studio/backend/infra/impl/document/searchstore/milvus"
	ssob "github.com/ynet-dev/ynet-studio/backend/infra/impl/document/searchstore/oceanbase"
	ssvikingdb "github.com/ynet-dev/ynet-studio/backend/infra/impl/document/searchstore/vikingdb"
	arkemb "github.com/ynet-dev/ynet-studio/backend/infra/impl/embedding/ark"
	"github.com/ynet-dev/ynet-studio/backend/infra/impl/embedding/http"
	embeddingProvider "github.com/ynet-dev/ynet-studio/backend/infra/impl/embedding/provider"
	embeddingRepo "github.com/ynet-dev/ynet-studio/backend/infra/impl/embedding/repository"
	"github.com/ynet-dev/ynet-studio/backend/infra/impl/embedding/wrap"
	"github.com/ynet-dev/ynet-studio/backend/infra/impl/eventbus"
	builtinM2Q "github.com/ynet-dev/ynet-studio/backend/infra/impl/messages2query/builtin"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/conv"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

type ServiceComponents struct {
	DB                  *gorm.DB
	IDGenSVC            idgen.IDGenerator
	Storage             storage.Storage
	RDB                 rdb.RDB
	EventBus            search.ResourceEventBus
	CacheCli            cache.Cmdable
	OCR                 ocr.OCR
	ParserManager       parser.Manager
	SearchStoreManagers []searchstore.Manager
}

func InitService(c *ServiceComponents) (*KnowledgeApplicationService, error) {
	ctx := context.Background()

	nameServer := os.Getenv(consts.MQServer)

	knowledgeProducer, err := eventbus.NewProducer(nameServer, consts.RMQTopicKnowledge, consts.RMQConsumeGroupKnowledge, 2)
	if err != nil {
		return nil, fmt.Errorf("init knowledge producer failed, err=%w", err)
	}

	var sManagers []searchstore.Manager

	// Use the provided search store managers
	sManagers = append(sManagers, c.SearchStoreManagers...)

	// Create ManagerFactory for space-level embedding support (preferred)
	managerFactory, err := createManagerFactory(ctx, c.DB)
	if err != nil {
		logs.CtxWarnf(ctx, "[InitService] failed to create manager factory: %v, will try legacy managers", err)
	}

	// Legacy vector search (fallback when ManagerFactory is not available)
	mgr, err := getVectorStore(ctx, c.DB)
	if err != nil {
		if managerFactory == nil {
			return nil, fmt.Errorf("init vector store failed and no manager factory available, err=%w", err)
		}
		// ManagerFactory is available — space-level embedding will handle it
		logs.CtxInfof(ctx, "[InitService] legacy vector store init failed (%v), using space-level ManagerFactory only", err)
	} else {
		sManagers = append(sManagers, mgr)
	}

	// Use the provided OCR implementation from ServiceComponents

	root, err := os.Getwd()
	if err != nil {
		logs.Warnf("[InitConfig] Failed to get current working directory: %v", err)
		root = os.Getenv("PWD")
	}

	// Query rewriter is optional - if not configured, knowledge retrieval will use original query
	var rewriter messages2query.MessagesToQuery
	if rewriterChatModel, configured, err := internal.GetBuiltinChatModel(ctx, "M2Q_"); err != nil {
		logs.CtxWarnf(ctx, "[InitService] failed to get M2Q chat model: %v, query rewrite will be disabled", err)
	} else if configured && rewriterChatModel != nil {
		filePath := filepath.Join(root, "resources/conf/prompt/messages_to_query_template_jinja2.json")
		rewriterTemplate, err := readJinja2PromptTemplate(filePath)
		if err != nil {
			logs.CtxWarnf(ctx, "[InitService] failed to read M2Q template: %v, query rewrite will be disabled", err)
		} else {
			rewriter, err = builtinM2Q.NewMessagesToQuery(ctx, rewriterChatModel, rewriterTemplate)
			if err != nil {
				logs.CtxWarnf(ctx, "[InitService] failed to create M2Q rewriter: %v, query rewrite will be disabled", err)
			} else {
				logs.CtxInfof(ctx, "[InitService] query rewriter initialized successfully")
			}
		}
	} else {
		logs.CtxInfof(ctx, "[InitService] M2Q chat model not configured, query rewrite will be disabled")
	}

	// NL2SQL is optional - if not configured, table knowledge NL2SQL queries will be disabled
	var n2s nl2sql.NL2SQL
	if n2sChatModel, configured, err := internal.GetBuiltinChatModel(ctx, "NL2SQL_"); err != nil {
		logs.CtxWarnf(ctx, "[InitService] failed to get NL2SQL chat model: %v, NL2SQL will be disabled", err)
	} else if configured && n2sChatModel != nil {
		filePath := filepath.Join(root, "resources/conf/prompt/nl2sql_template_jinja2.json")
		n2sTemplate, err := readJinja2PromptTemplate(filePath)
		if err != nil {
			logs.CtxWarnf(ctx, "[InitService] failed to read NL2SQL template: %v, NL2SQL will be disabled", err)
		} else {
			n2s, err = builtinNL2SQL.NewNL2SQL(ctx, n2sChatModel, n2sTemplate)
			if err != nil {
				logs.CtxWarnf(ctx, "[InitService] failed to create NL2SQL: %v, NL2SQL will be disabled", err)
			} else {
				logs.CtxInfof(ctx, "[InitService] NL2SQL initialized successfully")
			}
		}
	} else {
		logs.CtxInfof(ctx, "[InitService] NL2SQL chat model not configured, NL2SQL will be disabled")
	}

	// Create space-level rerank provider
	rrRepo := rerankRepo.NewSpaceRerankRepository(c.DB)
	rrSvc := rerankService.NewSpaceRerankService(rrRepo)
	rrProvider := rerankProvider.NewSpaceRerankProvider(rrSvc)

	knowledgeDomainSVC, knowledgeEventHandler := knowledgeImpl.NewKnowledgeSVC(&knowledgeImpl.KnowledgeSVCConfig{
		DB:                  c.DB,
		IDGen:               c.IDGenSVC,
		RDB:                 c.RDB,
		Producer:            knowledgeProducer,
		SearchStoreManagers: sManagers,
		ManagerFactory:      managerFactory, // New: space-level embedding support
		ParseManager:        c.ParserManager,
		Storage:             c.Storage,
		Rewriter:            rewriter,
		Reranker:            rrf.NewRRFReranker(0), // default rrf
		RerankProvider:      rrProvider,             // space-level rerank model support
		NL2Sql:              n2s,
		OCR:                 c.OCR,
		CacheCli:            c.CacheCli,
		ModelFactory:        chatmodelImpl.NewDefaultFactory(),
	})

	if err = eventbus.DefaultSVC().RegisterConsumer(nameServer, consts.RMQTopicKnowledge, consts.RMQConsumeGroupKnowledge, knowledgeEventHandler); err != nil {
		return nil, fmt.Errorf("register knowledge consumer failed, err=%w", err)
	}

	KnowledgeSVC.DomainSVC = knowledgeDomainSVC
	KnowledgeSVC.eventBus = c.EventBus
	KnowledgeSVC.storage = c.Storage
	return KnowledgeSVC, nil
}

// createManagerFactory creates a ManagerFactory for space-level embedding support
// Supports Milvus and OceanBase vector stores
// Global embedding from env is optional — space-level embedding configurations from DB take priority.
func createManagerFactory(ctx context.Context, db *gorm.DB) (searchstore.ManagerFactory, error) {
	vsType := os.Getenv("VECTOR_STORE_TYPE")

	// Get global embedding as fallback (optional — space-level configs from DB take priority)
	globalEmb, err := getEmbedding(ctx)
	if err != nil {
		logs.CtxInfof(ctx, "[createManagerFactory] no global embedding from env: %v, space-level configs from DB will be used", err)
		// globalEmb is nil — this is fine, spaces must have their own embedding config
	}

	// Create SpaceEmbeddingService
	repo := embeddingRepo.NewSpaceEmbeddingRepository(db)
	embSvc := embeddingService.NewSpaceEmbeddingService(repo)

	// Create SpaceEmbeddingProvider
	provider := embeddingProvider.NewSpaceEmbeddingProvider(embSvc, globalEmb)

	switch vsType {
	case "milvus":
		// Create Milvus client
		cctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		milvusAddr := os.Getenv("MILVUS_ADDR")
		mc, err := milvusclient.New(cctx, &milvusclient.ClientConfig{Address: milvusAddr})
		if err != nil {
			return nil, fmt.Errorf("failed to create milvus client: %w", err)
		}

		// Create ManagerFactory
		enableHybrid := false
		if globalEmb != nil {
			enableHybrid = globalEmb.SupportStatus() == embedding.SupportDenseAndSparse
		}
		factory, err := ssmilvus.NewManagerFactory(&ssmilvus.ManagerFactoryConfig{
			Client:            mc,
			EmbeddingProvider: provider,
			GlobalEmbedding:   globalEmb,
			EnableHybrid:      ptr.Of(enableHybrid),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create manager factory: %w", err)
		}

		logs.CtxInfof(ctx, "[createManagerFactory] successfully created Milvus ManagerFactory with space-level embedding support")
		return factory, nil

	case "oceanbase":
		factory, err := ssob.NewManagerFactory(&ssob.ManagerFactoryConfig{
			DB:                db,
			EmbeddingProvider: provider,
			GlobalEmbedding:   globalEmb,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create oceanbase manager factory: %w", err)
		}

		logs.CtxInfof(ctx, "[createManagerFactory] successfully created OceanBase ManagerFactory with space-level embedding support")
		return factory, nil

	default:
		logs.CtxInfof(ctx, "[createManagerFactory] vector store type %s does not support ManagerFactory yet", vsType)
		return nil, nil
	}
}

func getVectorStore(ctx context.Context, db *gorm.DB) (searchstore.Manager, error) {
	vsType := os.Getenv("VECTOR_STORE_TYPE")

	switch vsType {
	case "oceanbase":
		emb, err := getEmbedding(ctx)
		if err != nil {
			return nil, fmt.Errorf("init oceanbase embedding failed, err=%w", err)
		}

		mgr, err := ssob.NewManager(&ssob.ManagerConfig{
			DB:        db,
			Embedding: emb,
		})
		if err != nil {
			return nil, fmt.Errorf("init oceanbase vector store failed, err=%w", err)
		}

		return mgr, nil

	case "milvus":
		cctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()

		milvusAddr := os.Getenv("MILVUS_ADDR")
		mc, err := milvusclient.New(cctx, &milvusclient.ClientConfig{Address: milvusAddr})
		if err != nil {
			return nil, fmt.Errorf("init milvus client failed, err=%w", err)
		}

		emb, err := getEmbedding(ctx)
		if err != nil {
			return nil, fmt.Errorf("init milvus embedding failed, err=%w", err)
		}

		mgr, err := ssmilvus.NewManager(&ssmilvus.ManagerConfig{
			Client:       mc,
			Embedding:    emb,
			EnableHybrid: ptr.Of(true),
		})
		if err != nil {
			return nil, fmt.Errorf("init milvus vector store failed, err=%w", err)
		}

		return mgr, nil
	case "vikingdb":
		var (
			host      = os.Getenv("VIKING_DB_HOST")
			region    = os.Getenv("VIKING_DB_REGION")
			ak        = os.Getenv("VIKING_DB_AK")
			sk        = os.Getenv("VIKING_DB_SK")
			scheme    = os.Getenv("VIKING_DB_SCHEME")
			modelName = os.Getenv("VIKING_DB_MODEL_NAME")
		)
		if ak == "" || sk == "" {
			return nil, fmt.Errorf("invalid vikingdb ak / sk")
		}
		if host == "" {
			host = "api-vikingdb.volces.com"
		}
		if region == "" {
			region = "cn-beijing"
		}
		if scheme == "" {
			scheme = "https"
		}

		var embConfig *ssvikingdb.VikingEmbeddingConfig
		if modelName != "" {
			embName := ssvikingdb.VikingEmbeddingModelName(modelName)
			if embName.Dimensions() == 0 {
				return nil, fmt.Errorf("embedding model not support, model_name=%s", modelName)
			}
			embConfig = &ssvikingdb.VikingEmbeddingConfig{
				UseVikingEmbedding: true,
				EnableHybrid:       embName.SupportStatus() == embedding.SupportDenseAndSparse,
				ModelName:          embName,
				ModelVersion:       embName.ModelVersion(),
				DenseWeight:        ptr.Of(0.2),
				BuiltinEmbedding:   nil,
			}
		} else {
			builtinEmbedding, err := getEmbedding(ctx)
			if err != nil {
				return nil, fmt.Errorf("builtint embedding init failed, err=%w", err)
			}

			embConfig = &ssvikingdb.VikingEmbeddingConfig{
				UseVikingEmbedding: false,
				EnableHybrid:       false,
				BuiltinEmbedding:   builtinEmbedding,
			}
		}
		svc := vikingdb.NewVikingDBService(host, region, ak, sk, scheme)
		mgr, err := ssvikingdb.NewManager(&ssvikingdb.ManagerConfig{
			Service:         svc,
			IndexingConfig:  nil, // use default config
			EmbeddingConfig: embConfig,
		})
		if err != nil {
			return nil, fmt.Errorf("init vikingdb manager failed, err=%w", err)
		}

		return mgr, nil

	default:
		return nil, fmt.Errorf("unexpected vector store type, type=%s", vsType)
	}
}

func getEmbedding(ctx context.Context) (embedding.Embedder, error) {
	var batchSize int
	if bs, err := strconv.ParseInt(os.Getenv("EMBEDDING_MAX_BATCH_SIZE"), 10, 64); err != nil {
		logs.CtxWarnf(ctx, "EMBEDDING_MAX_BATCH_SIZE not set / invalid, using default batchSize=100")
		batchSize = 100
	} else {
		batchSize = int(bs)
	}

	var emb embedding.Embedder

	switch os.Getenv("EMBEDDING_TYPE") {
	case "openai":
		var (
			openAIEmbeddingBaseURL     = os.Getenv("OPENAI_EMBEDDING_BASE_URL")
			openAIEmbeddingModel       = os.Getenv("OPENAI_EMBEDDING_MODEL")
			openAIEmbeddingApiKey      = os.Getenv("OPENAI_EMBEDDING_API_KEY")
			openAIEmbeddingByAzure     = os.Getenv("OPENAI_EMBEDDING_BY_AZURE")
			openAIEmbeddingApiVersion  = os.Getenv("OPENAI_EMBEDDING_API_VERSION")
			openAIEmbeddingDims        = os.Getenv("OPENAI_EMBEDDING_DIMS")
			openAIRequestEmbeddingDims = os.Getenv("OPENAI_EMBEDDING_REQUEST_DIMS")
		)

		byAzure, err := strconv.ParseBool(openAIEmbeddingByAzure)
		if err != nil {
			return nil, fmt.Errorf("init openai embedding by_azure failed, err=%w", err)
		}

		dims, err := strconv.ParseInt(openAIEmbeddingDims, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("init openai embedding dims failed, err=%w", err)
		}

		openAICfg := &openai.EmbeddingConfig{
			APIKey:     openAIEmbeddingApiKey,
			ByAzure:    byAzure,
			BaseURL:    openAIEmbeddingBaseURL,
			APIVersion: openAIEmbeddingApiVersion,
			Model:      openAIEmbeddingModel,
			// Dimensions: ptr.Of(int(dims)),
		}
		reqDims := conv.StrToInt64D(openAIRequestEmbeddingDims, 0)
		if reqDims > 0 {
			// some openai model not support request dims
			openAICfg.Dimensions = ptr.Of(int(reqDims))
		}

		emb, err = wrap.NewOpenAIEmbedder(ctx, openAICfg, dims, batchSize)
		if err != nil {
			return nil, fmt.Errorf("init openai embedding failed, err=%w", err)
		}

	case "ark":
		var (
			arkEmbeddingBaseURL = os.Getenv("ARK_EMBEDDING_BASE_URL")
			arkEmbeddingModel   = os.Getenv("ARK_EMBEDDING_MODEL")
			arkEmbeddingApiKey  = os.Getenv("ARK_EMBEDDING_API_KEY")
			// deprecated: use ARK_EMBEDDING_API_KEY instead
			// ARK_EMBEDDING_AK will be removed in the future
			arkEmbeddingAK      = os.Getenv("ARK_EMBEDDING_AK")
			arkEmbeddingDims    = os.Getenv("ARK_EMBEDDING_DIMS")
			arkEmbeddingAPIType = os.Getenv("ARK_EMBEDDING_API_TYPE")
		)

		dims, err := strconv.ParseInt(arkEmbeddingDims, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("init ark embedding dims failed, err=%w", err)
		}

		apiType := ark.APITypeText
		if arkEmbeddingAPIType != "" {
			if t := ark.APIType(arkEmbeddingAPIType); t != ark.APITypeText && t != ark.APITypeMultiModal {
				return nil, fmt.Errorf("init ark embedding api_type failed, invalid api_type=%s", t)
			} else {
				apiType = t
			}
		}

		// ARK API has a strict limit of 10 for batch size (applies to all ARK embeddings)
		arkBatchSize := batchSize
		if arkBatchSize > 10 {
			arkBatchSize = 10
		}

		emb, err = arkemb.NewArkEmbedder(ctx, &ark.EmbeddingConfig{
			APIKey: func() string {
				if arkEmbeddingApiKey != "" {
					return arkEmbeddingApiKey
				}
				return arkEmbeddingAK
			}(),
			Model:   arkEmbeddingModel,
			BaseURL: arkEmbeddingBaseURL,
			APIType: &apiType,
		}, dims, arkBatchSize)
		if err != nil {
			return nil, fmt.Errorf("init ark embedding client failed, err=%w", err)
		}

	case "ollama":
		var (
			ollamaEmbeddingBaseURL = os.Getenv("OLLAMA_EMBEDDING_BASE_URL")
			ollamaEmbeddingModel   = os.Getenv("OLLAMA_EMBEDDING_MODEL")
			ollamaEmbeddingDims    = os.Getenv("OLLAMA_EMBEDDING_DIMS")
		)

		dims, err := strconv.ParseInt(ollamaEmbeddingDims, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("init ollama embedding dims failed, err=%w", err)
		}

		emb, err = wrap.NewOllamaEmbedder(ctx, &ollamaEmb.EmbeddingConfig{
			BaseURL: ollamaEmbeddingBaseURL,
			Model:   ollamaEmbeddingModel,
		}, dims, batchSize)
		if err != nil {
			return nil, fmt.Errorf("init ollama embedding failed, err=%w", err)
		}

	case "http":
		var (
			httpEmbeddingBaseURL = os.Getenv("HTTP_EMBEDDING_ADDR")
			httpEmbeddingDims    = os.Getenv("HTTP_EMBEDDING_DIMS")
		)
		dims, err := strconv.ParseInt(httpEmbeddingDims, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("init http embedding dims failed, err=%w", err)
		}
		emb, err = http.NewEmbedding(httpEmbeddingBaseURL, dims, batchSize)
		if err != nil {
			return nil, fmt.Errorf("init http embedding failed, err=%w", err)
		}

	default:
		return nil, fmt.Errorf("init knowledge embedding failed, type not configured")
	}

	return emb, nil
}

func readJinja2PromptTemplate(jsonFilePath string) (prompt.ChatTemplate, error) {
	b, err := os.ReadFile(jsonFilePath)
	if err != nil {
		return nil, err
	}
	var m2qMessages []*schema.Message
	if err = json.Unmarshal(b, &m2qMessages); err != nil {
		return nil, err
	}
	tpl := make([]schema.MessagesTemplate, len(m2qMessages))
	for i := range m2qMessages {
		tpl[i] = m2qMessages[i]
	}
	return prompt.FromMessages(schema.Jinja2, tpl...), nil
}
