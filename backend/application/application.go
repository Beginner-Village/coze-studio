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

package application

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ynet-dev/ynet-studio/backend/api/handler/coze"
	embeddingHandler "github.com/ynet-dev/ynet-studio/backend/api/handler/embedding"
	rerankHandler "github.com/ynet-dev/ynet-studio/backend/api/handler/rerank"
	embeddingApp "github.com/ynet-dev/ynet-studio/backend/application/embedding"
	rerankApp "github.com/ynet-dev/ynet-studio/backend/application/rerank"
	"github.com/ynet-dev/ynet-studio/backend/application/openauth"
	"github.com/ynet-dev/ynet-studio/backend/application/template"
	crosssearch "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/search"
	agentrepository "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/repository"
	apprepository "github.com/ynet-dev/ynet-studio/backend/domain/app/repository"
	knowledgerepository "github.com/ynet-dev/ynet-studio/backend/domain/knowledge/repository"
	knowledgesvc "github.com/ynet-dev/ynet-studio/backend/domain/knowledge/service"
	oplogsvc "github.com/ynet-dev/ynet-studio/backend/domain/operationlog/service"
	searchService "github.com/ynet-dev/ynet-studio/backend/domain/search/service"
	modelrepository "github.com/ynet-dev/ynet-studio/backend/domain/model/repository"
	modelservice "github.com/ynet-dev/ynet-studio/backend/domain/model/service"

	"github.com/ynet-dev/ynet-studio/backend/application/app"
	"github.com/ynet-dev/ynet-studio/backend/application/base/appinfra"
	"github.com/ynet-dev/ynet-studio/backend/application/connector"
	"github.com/ynet-dev/ynet-studio/backend/application/conversation"
	"github.com/ynet-dev/ynet-studio/backend/application/external_knowledge"
	"github.com/ynet-dev/ynet-studio/backend/application/knowledge"
	"github.com/ynet-dev/ynet-studio/backend/application/memory"
	"github.com/ynet-dev/ynet-studio/backend/application/admin"
	"github.com/ynet-dev/ynet-studio/backend/application/modelmgr"
	"github.com/ynet-dev/ynet-studio/backend/application/operationlog"
	"github.com/ynet-dev/ynet-studio/backend/application/plugin"
	"github.com/ynet-dev/ynet-studio/backend/application/prompt"
	"github.com/ynet-dev/ynet-studio/backend/application/search"
	"github.com/ynet-dev/ynet-studio/backend/application/shortcutcmd"
	"github.com/ynet-dev/ynet-studio/backend/application/singleagent"
	skillApp "github.com/ynet-dev/ynet-studio/backend/application/skill"
	spaceapp "github.com/ynet-dev/ynet-studio/backend/application/space"
	"github.com/ynet-dev/ynet-studio/backend/application/statistics"
	"github.com/ynet-dev/ynet-studio/backend/application/upload"
	"github.com/ynet-dev/ynet-studio/backend/application/user"
	"github.com/ynet-dev/ynet-studio/backend/application/workflow"
	ynet_agent_app "github.com/ynet-dev/ynet-studio/backend/application/ynet_agent"
	crossagent "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/agent"
	crossagentrun "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/agentrun"
	crossconnector "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/connector"
	crossconversation "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/conversation"
	crossdatabase "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/database"
	crossdatacopy "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/datacopy"
	crossknowledge "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/knowledge"
	crossmessage "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/message"
	"github.com/ynet-dev/ynet-studio/backend/domain/ynet_agent"
	ynet_agent_repo "github.com/ynet-dev/ynet-studio/backend/infra/repository/ynet_agent"
	crossmodelmgr "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/modelmgr"
	crossplugin "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/plugin"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	crossskill "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/skill"
	crossuser "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/user"
	crossvariables "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/variables"
	crossworkflow "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/workflow"
	agentrunImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/agentrun"
	connectorImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/connector"
	conversationImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/conversation"
	crossuserImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/crossuser"
	databaseImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/database"
	dataCopyImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/datacopy"
	knowledgeImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/knowledge"
	messageImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/message"
	modelmgrImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/modelmgr"
	pluginImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/plugin"
	skillImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/skill"
	searchImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/search"
	singleagentImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/singleagent"
	variablesImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/variables"
	workflowImpl "github.com/ynet-dev/ynet-studio/backend/crossdomain/impl/workflow"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/eventbus"
	"github.com/ynet-dev/ynet-studio/backend/infra/impl/checkpoint"
	implEventbus "github.com/ynet-dev/ynet-studio/backend/infra/impl/eventbus"
	adminrepo "github.com/ynet-dev/ynet-studio/backend/domain/admin/repository"
)

type eventbusImpl struct {
	resourceEventBus search.ResourceEventBus
	projectEventBus  search.ProjectEventBus
}

type basicServices struct {
	infra         *appinfra.AppDependencies
	eventbus      *eventbusImpl
	modelMgrSVC   *modelmgr.ModelmgrApplicationService
	connectorSVC  *connector.ConnectorApplicationService
	userSVC       *user.UserApplicationService
	promptSVC     *prompt.PromptApplicationService
	templateSVC   *template.ApplicationService
	openAuthSVC   *openauth.OpenAuthApplicationService
	statisticsApp *statistics.StatisticsApp
}

type primaryServices struct {
	basicServices *basicServices
	infra         *appinfra.AppDependencies

	pluginSVC    *plugin.PluginApplicationService
	memorySVC    *memory.MemoryApplicationServices
	knowledgeSVC *knowledge.KnowledgeApplicationService
	workflowSVC  *workflow.ApplicationService
	shortcutSVC  *shortcutcmd.ShortcutCmdApplicationService
}

type complexServices struct {
	primaryServices *primaryServices
	singleAgentSVC  *singleagent.SingleAgentApplicationService
	appSVC          *app.APPApplicationService
	searchSVC       *search.SearchApplicationService
	conversationSVC *conversation.ConversationApplicationService
}

func Init(ctx context.Context) (err error) {
	infra, err := appinfra.Init(ctx)
	if err != nil {
		return err
	}

	eventbus := initEventBus(infra)

	basicServices, err := initBasicServices(ctx, infra, eventbus)
	if err != nil {
		return fmt.Errorf("Init - initBasicServices failed, err: %v", err)
	}

	// 设置全局统计服务（线程安全）
	setGlobalStatisticsApp(basicServices.statisticsApp)

	primaryServices, err := initPrimaryServices(ctx, basicServices)
	if err != nil {
		return fmt.Errorf("Init - initPrimaryServices failed, err: %v", err)
	}

	complexServices, err := initComplexServices(ctx, primaryServices)
	if err != nil {
		return fmt.Errorf("Init - initVitalServices failed, err: %v", err)
	}

	crossconnector.SetDefaultSVC(connectorImpl.InitDomainService(basicServices.connectorSVC.DomainSVC))
	crossdatabase.SetDefaultSVC(databaseImpl.InitDomainService(primaryServices.memorySVC.DatabaseDomainSVC))
	crossknowledge.SetDefaultSVC(knowledgeImpl.InitDomainService(primaryServices.knowledgeSVC.DomainSVC))
	crossplugin.SetDefaultSVC(pluginImpl.InitDomainService(primaryServices.pluginSVC.DomainSVC, infra.TOSClient))
	crossvariables.SetDefaultSVC(variablesImpl.InitDomainService(primaryServices.memorySVC.VariablesDomainSVC))
	crossworkflow.SetDefaultSVC(workflowImpl.InitDomainService(primaryServices.workflowSVC.DomainSVC))
	crossconversation.SetDefaultSVC(conversationImpl.InitDomainService(complexServices.conversationSVC.ConversationDomainSVC))
	crossmessage.SetDefaultSVC(messageImpl.InitDomainService(complexServices.conversationSVC.MessageDomainSVC))
	crossagentrun.SetDefaultSVC(agentrunImpl.InitDomainService(complexServices.conversationSVC.AgentRunDomainSVC))
	crossagent.SetDefaultSVC(singleagentImpl.InitDomainService(complexServices.singleAgentSVC.DomainSVC))
	crossuser.SetDefaultSVC(crossuserImpl.InitDomainService(basicServices.userSVC.DomainSVC))
	crossdatacopy.SetDefaultSVC(dataCopyImpl.InitDomainService(basicServices.infra))
	crosssearch.SetDefaultSVC(searchImpl.InitDomainService(complexServices.searchSVC.DomainSVC))
	crossmodelmgr.SetDefaultSVC(modelmgrImpl.InitDomainService(infra.ModelMgr, nil))

	// Initialize Model Service
	modelService := initModelService(infra)
	if modelService != nil {
		coze.InitModelService(modelService)
	}

	// Initialize Space Embedding Service
	spaceEmbeddingApp := embeddingApp.NewSpaceEmbeddingApp(infra.DB)
	embeddingHandler.InitSpaceEmbeddingApp(spaceEmbeddingApp)

	// Initialize Space Rerank Service
	spaceRerankApp := rerankApp.NewSpaceRerankApp(infra.DB)
	rerankHandler.InitSpaceRerankApp(spaceRerankApp)

	// Wire per-space ES resync.
	//
	// Has to happen after both searchSVC and knowledgeSVC exist (search needs
	// agent/app/kb listers to replay writes; kb adapter has to live in the
	// knowledge domain because *model.Knowledge is internal-import-only).
	agentResyncRepo := agentrepository.NewSingleAgentRepo(infra.DB, infra.IDGenSVC, infra.CacheCli)
	appResyncRepo := apprepository.NewAPPRepo(&apprepository.APPRepoComponents{
		IDGen:    infra.IDGenSVC,
		DB:       infra.DB,
		CacheCli: infra.CacheCli,
	})
	kbResyncRepo := knowledgerepository.NewKnowledgeDAO(infra.DB)
	complexServices.searchSVC.DomainSVC.SetResyncDeps(
		agentResyncRepo,
		appResyncRepo,
		knowledgesvc.NewKbInfoLister(kbResyncRepo),
		// 4 raw-gorm-backed listers for the other coze_resource res_types.
		// They live in the search/service package because the underlying
		// PO model packages are internal/dal/model/ (Go internal-import
		// rule blocks them from being reached via the domain repos here).
		searchService.NewWorkflowDBLister(infra.DB),
		searchService.NewPluginDBLister(infra.DB),
		searchService.NewPromptDBLister(infra.DB),
		searchService.NewDatabaseDBLister(infra.DB),
	)
	spaceapp.InitResyncService(
		basicServices.userSVC.DomainSVC,
		complexServices.searchSVC.DomainSVC,
		primaryServices.knowledgeSVC.DomainSVC,
	)
	spaceapp.InitConfigureModelsService(
		basicServices.userSVC.DomainSVC,
		infra.DB,
		infra.CacheCli,
		primaryServices.knowledgeSVC.RerankCacheInvalidator,
		primaryServices.knowledgeSVC.EmbeddingCacheInvalidator,
	)
	spaceapp.InitDiagnoseService(
		basicServices.userSVC.DomainSVC,
		infra.DB,
		infra.ESClient,
		infra.SearchStoreManagers,
	)

	// Wire space-level operation audit log: build the domain service, start its
	// background batch-writer + retention-cleanup loops, and populate the global
	// singleton used by OperationLogMW. The OPERATION_LOG_ENABLED=false escape
	// hatch skips Init entirely; the middleware's Collect is a no-op until then.
	if !strings.EqualFold(os.Getenv("OPERATION_LOG_ENABLED"), "false") {
		operationlog.Init(
			ctx,
			infra.DB,
			infra.IDGenSVC,
			basicServices.userSVC.DomainSVC,
			oplogsvc.Config{
				BufferSize:    envInt("OPERATION_LOG_BUFFER_SIZE", 4096),
				RetentionDays: envInt("OPERATION_LOG_RETENTION_DAYS", 90),
			},
		)
	}

	return nil
}

// envInt reads an integer environment variable, falling back to def when the
// variable is unset or not a valid integer.
func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func initEventBus(infra *appinfra.AppDependencies) *eventbusImpl {
	e := &eventbusImpl{}
	eventbus.SetDefaultSVC(implEventbus.NewConsumerService())
	e.resourceEventBus = search.NewResourceEventBus(infra.ResourceEventProducer)
	e.projectEventBus = search.NewProjectEventBus(infra.AppEventProducer)

	return e
}

// initBasicServices init basic services that only depends on infra.
func initBasicServices(ctx context.Context, infra *appinfra.AppDependencies, e *eventbusImpl) (*basicServices, error) {
	upload.InitService(&upload.UploadComponents{
		Cache:  infra.CacheCli,
		Oss:    infra.TOSClient,
		DB:     infra.DB,
		Idgen:  infra.IDGenSVC,
		ImageX: infra.ImageXClient,
	})
	openAuthSVC := openauth.InitService(infra.DB, infra.IDGenSVC)
	promptSVC := prompt.InitService(infra.DB, infra.IDGenSVC, e.resourceEventBus)
	// Initialize model repository and service
	modelRepo := modelrepository.NewModelRepository(infra.DB)
	modelTemplateRepo := modelrepository.NewModelTemplateRepository(infra.DB)
	modelService := modelservice.NewModelService(modelRepo, infra.TOSClient)
	modelMgrSVC := modelmgr.InitService(infra.ModelMgr, infra.TOSClient, modelService, modelRepo, modelTemplateRepo)
	// wire admin application service dependencies
	adminRepo := adminrepo.NewAdminRepository(infra.DB)
	admin.AdminApplicationSVC = &admin.AdminApplicationService{
		AdminRepo:     adminRepo,
		ModelRepo:     modelRepo,
		StorageClient: infra.TOSClient,
	}
	connectorSVC := connector.InitService(infra.TOSClient)
	userSVC := user.InitService(ctx, infra.DB, infra.TOSClient, infra.IDGenSVC)
	spaceapp.InitSpaceExportImportService(infra.DB, infra.TOSClient, infra.IDGenSVC, e.resourceEventBus, e.projectEventBus)
	spaceapp.InitSyncService(infra.DB, infra.TOSClient, infra.IDGenSVC, e.resourceEventBus, e.projectEventBus)
	spaceapp.InitReleaseService(infra.DB, infra.TOSClient)
	templateSVC := template.InitService(ctx, &template.ServiceComponents{
		DB:      infra.DB,
		IDGen:   infra.IDGenSVC,
		Storage: infra.TOSClient,
	})
	statisticsApp := statistics.NewStatisticsApp(infra.DB, infra.TOSClient)

	// Initialize external knowledge service
	external_knowledge.InitExternalKnowledgeService(infra.DB)

	// Initialize Skill service
	skillSVC := skillApp.InitService(&skillApp.ServiceComponents{
		IDGen: infra.IDGenSVC,
		DB:    infra.DB,
	})
	crossskill.SetDefaultSVC(skillImpl.InitDomainService(skillSVC.DomainSVC))

	// Wire sandbox manager for agent runtime tools (run_bash/read_file/write_file/list_files)
	if infra.SandboxManager != nil {
		crosssandbox.SetDefaultSVC(infra.SandboxManager)
	}

	// Initialize HiAgent repository
	hiAgentRepo := ynet_agent_repo.NewHiAgentRepository(infra.DB)
	ynet_agent.SetHiAgentRepository(hiAgentRepo)
	ynet_agent_app.InitService(infra.DB, infra.IDGenSVC, hiAgentRepo)

	return &basicServices{
		infra:         infra,
		eventbus:      e,
		modelMgrSVC:   modelMgrSVC,
		connectorSVC:  connectorSVC,
		userSVC:       userSVC,
		promptSVC:     promptSVC,
		templateSVC:   templateSVC,
		openAuthSVC:   openAuthSVC,
		statisticsApp: statisticsApp,
	}, nil
}

// initPrimaryServices init primary services that depends on basic services.
func initPrimaryServices(ctx context.Context, basicServices *basicServices) (*primaryServices, error) {
	pluginSVC, err := plugin.InitService(ctx, basicServices.toPluginServiceComponents())
	if err != nil {
		return nil, err
	}

	memorySVC := memory.InitService(basicServices.toMemoryServiceComponents())

	knowledgeSVC, err := knowledge.InitService(basicServices.toKnowledgeServiceComponents(memorySVC))
	if err != nil {
		return nil, err
	}

	workflowDomainSVC, err := workflow.InitService(ctx,
		basicServices.toWorkflowServiceComponents(pluginSVC, memorySVC, knowledgeSVC))
	if err != nil {
		return nil, err
	}

	shortcutSVC := shortcutcmd.InitService(basicServices.infra.DB, basicServices.infra.IDGenSVC)

	return &primaryServices{
		basicServices: basicServices,
		pluginSVC:     pluginSVC,
		memorySVC:     memorySVC,
		knowledgeSVC:  knowledgeSVC,
		workflowSVC:   workflowDomainSVC,
		shortcutSVC:   shortcutSVC,
		infra:         basicServices.infra,
	}, nil
}

// initComplexServices init complex services that depends on primary services.
func initComplexServices(ctx context.Context, p *primaryServices) (*complexServices, error) {
	singleAgentSVC, err := singleagent.InitService(p.toSingleAgentServiceComponents())
	if err != nil {
		return nil, err
	}

	appSVC, err := app.InitService(p.toAPPServiceComponents())
	if err != nil {
		return nil, err
	}

	searchSVC, err := search.InitService(ctx, p.toSearchServiceComponents(singleAgentSVC, appSVC))
	if err != nil {
		return nil, err
	}

	conversationSVC := conversation.InitService(p.toConversationComponents(singleAgentSVC))

	return &complexServices{
		primaryServices: p,
		singleAgentSVC:  singleAgentSVC,
		appSVC:          appSVC,
		searchSVC:       searchSVC,
		conversationSVC: conversationSVC,
	}, nil
}

func (b *basicServices) toPluginServiceComponents() *plugin.ServiceComponents {
	return &plugin.ServiceComponents{
		IDGen:    b.infra.IDGenSVC,
		DB:       b.infra.DB,
		EventBus: b.eventbus.resourceEventBus,
		OSS:      b.infra.TOSClient,
		UserSVC:  b.userSVC.DomainSVC,
	}
}

func (b *basicServices) toKnowledgeServiceComponents(memoryService *memory.MemoryApplicationServices) *knowledge.ServiceComponents {
	return &knowledge.ServiceComponents{
		DB:                  b.infra.DB,
		IDGenSVC:            b.infra.IDGenSVC,
		Storage:             b.infra.TOSClient,
		RDB:                 memoryService.RDBDomainSVC,
		SearchStoreManagers: b.infra.SearchStoreManagers,
		EventBus:            b.eventbus.resourceEventBus,
		CacheCli:            b.infra.CacheCli,
		OCR:                 b.infra.OCR,
		ParserManager:       b.infra.ParserManager,
	}
}

func (b *basicServices) toMemoryServiceComponents() *memory.ServiceComponents {
	return &memory.ServiceComponents{
		IDGen:                  b.infra.IDGenSVC,
		DB:                     b.infra.DB,
		EventBus:               b.eventbus.resourceEventBus,
		TosClient:              b.infra.TOSClient,
		ResourceDomainNotifier: b.eventbus.resourceEventBus,
		CacheCli:               b.infra.CacheCli,
	}
}

func (b *basicServices) toWorkflowServiceComponents(pluginSVC *plugin.PluginApplicationService, memorySVC *memory.MemoryApplicationServices, knowledgeSVC *knowledge.KnowledgeApplicationService) *workflow.ServiceComponents {
	return &workflow.ServiceComponents{
		IDGen:              b.infra.IDGenSVC,
		DB:                 b.infra.DB,
		Cache:              b.infra.CacheCli,
		Tos:                b.infra.TOSClient,
		ImageX:             b.infra.ImageXClient,
		DatabaseDomainSVC:  memorySVC.DatabaseDomainSVC,
		VariablesDomainSVC: memorySVC.VariablesDomainSVC,
		PluginDomainSVC:    pluginSVC.DomainSVC,
		KnowledgeDomainSVC: knowledgeSVC.DomainSVC,
		DomainNotifier:     b.eventbus.resourceEventBus,
		CPStore:            checkpoint.NewRedisStore(b.infra.CacheCli),
		CodeRunner:         b.infra.CodeRunner,
	}
}

func (p *primaryServices) toSingleAgentServiceComponents() *singleagent.ServiceComponents {
	return &singleagent.ServiceComponents{
		IDGen:                p.basicServices.infra.IDGenSVC,
		DB:                   p.basicServices.infra.DB,
		Cache:                p.basicServices.infra.CacheCli,
		TosClient:            p.basicServices.infra.TOSClient,
		ImageX:               p.basicServices.infra.ImageXClient,
		ModelMgr:             p.infra.ModelMgr,
		UserDomainSVC:        p.basicServices.userSVC.DomainSVC,
		EventBus:             p.basicServices.eventbus.projectEventBus,
		DatabaseDomainSVC:    p.memorySVC.DatabaseDomainSVC,
		ConnectorDomainSVC:   p.basicServices.connectorSVC.DomainSVC,
		KnowledgeDomainSVC:   p.knowledgeSVC.DomainSVC,
		PluginDomainSVC:      p.pluginSVC.DomainSVC,
		WorkflowDomainSVC:    p.workflowSVC.DomainSVC,
		VariablesDomainSVC:   p.memorySVC.VariablesDomainSVC,
		ShortcutCMDDomainSVC: p.shortcutSVC.ShortCutDomainSVC,
		CPStore:              checkpoint.NewRedisStore(p.infra.CacheCli),
		Embedder:             p.basicServices.infra.Embedder,
	}
}

func (p *primaryServices) toSearchServiceComponents(singleAgentSVC *singleagent.SingleAgentApplicationService, appSVC *app.APPApplicationService) *search.ServiceComponents {
	infra := p.basicServices.infra

	return &search.ServiceComponents{
		DB:                   infra.DB,
		Cache:                infra.CacheCli,
		TOS:                  infra.TOSClient,
		ESClient:             infra.ESClient,
		ProjectEventBus:      p.basicServices.eventbus.projectEventBus,
		SingleAgentDomainSVC: singleAgentSVC.DomainSVC,
		APPDomainSVC:         appSVC.DomainSVC,
		KnowledgeDomainSVC:   p.knowledgeSVC.DomainSVC,
		PluginDomainSVC:      p.pluginSVC.DomainSVC,
		WorkflowDomainSVC:    p.workflowSVC.DomainSVC,
		UserDomainSVC:        p.basicServices.userSVC.DomainSVC,
		ConnectorDomainSVC:   p.basicServices.connectorSVC.DomainSVC,
		PromptDomainSVC:      p.basicServices.promptSVC.DomainSVC,
		DatabaseDomainSVC:    p.memorySVC.DatabaseDomainSVC,
	}
}

func (p *primaryServices) toAPPServiceComponents() *app.ServiceComponents {
	infra := p.basicServices.infra
	basic := p.basicServices
	return &app.ServiceComponents{
		IDGen:           infra.IDGenSVC,
		DB:              infra.DB,
		OSS:             infra.TOSClient,
		CacheCli:        infra.CacheCli,
		ModelMgr:        infra.ModelMgr,
		ProjectEventBus: basic.eventbus.projectEventBus,
		UserSVC:         basic.userSVC.DomainSVC,
		ConnectorSVC:    basic.connectorSVC.DomainSVC,
		VariablesSVC:    p.memorySVC.VariablesDomainSVC,
	}
}

func (p *primaryServices) toConversationComponents(singleAgentSVC *singleagent.SingleAgentApplicationService) *conversation.ServiceComponents {
	infra := p.basicServices.infra

	return &conversation.ServiceComponents{
		DB:                   infra.DB,
		IDGen:                infra.IDGenSVC,
		TosClient:            infra.TOSClient,
		ImageX:               infra.ImageXClient,
		ModelMgr:             p.basicServices.modelMgrSVC.Mgr,
		Cache:                infra.CacheCli,
		SingleAgentDomainSVC: singleAgentSVC.DomainSVC,
	}
}

func initModelService(infra *appinfra.AppDependencies) modelservice.ModelService {
	repo := modelrepository.NewModelRepository(infra.DB)
	return modelservice.NewModelService(repo, infra.TOSClient)
}

var (
	globalStatisticsApp      *statistics.StatisticsApp
	statisticsAppOnce        sync.Once
	statisticsAppInitialized bool
)

// GetStatisticsApp 获取统计应用实例（线程安全）
// 注意：此函数应在 Init() 调用后使用
func GetStatisticsApp() *statistics.StatisticsApp {
	if !statisticsAppInitialized {
		return nil
	}
	return globalStatisticsApp
}

// setGlobalStatisticsApp 设置全局统计应用实例（仅供Init调用）
func setGlobalStatisticsApp(app *statistics.StatisticsApp) {
	statisticsAppOnce.Do(func() {
		globalStatisticsApp = app
		statisticsAppInitialized = true
	})
}
