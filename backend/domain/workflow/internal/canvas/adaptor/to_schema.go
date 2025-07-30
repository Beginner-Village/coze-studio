/*
 * Copyright 2025 coze-dev Authors
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

package adaptor

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	einoCompose "github.com/cloudwego/eino/compose"
	"github.com/spf13/cast"

	"github.com/coze-dev/coze-studio/backend/domain/workflow"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/crossdomain/model"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/entity/vo"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/canvas/convert"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/compose"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/batch"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/code"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/database"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/emitter"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/entry"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/exit"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/httprequester"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/intentdetector"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/json"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/knowledge"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/llm"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/loop"
	_break "github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/loop/break"
	_continue "github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/loop/continue"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/plugin"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/qa"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/receiver"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/selector"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/subworkflow"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/textprocessor"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/variableaggregator"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/nodes/variableassigner"
	"github.com/coze-dev/coze-studio/backend/domain/workflow/internal/schema"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/slices"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ternary"
	"github.com/coze-dev/coze-studio/backend/pkg/safego"
	"github.com/coze-dev/coze-studio/backend/pkg/sonic"
)

func CanvasToWorkflowSchema(ctx context.Context, s *vo.Canvas) (sc *schema.WorkflowSchema, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = safego.NewPanicErr(panicErr, debug.Stack())
		}
	}()

	connectedNodes, connectedEdges := PruneIsolatedNodes(s.Nodes, s.Edges, nil)
	s = &vo.Canvas{
		Nodes: connectedNodes,
		Edges: connectedEdges,
	}

	sc = &schema.WorkflowSchema{}

	nodeMap := make(map[string]*vo.Node)

	for i, node := range s.Nodes {
		nodeMap[node.ID] = s.Nodes[i]
		for j, subNode := range node.Blocks {
			nodeMap[subNode.ID] = node.Blocks[j]
			subNode.SetParent(node)
			if len(subNode.Blocks) > 0 {
				return nil, fmt.Errorf("nested inner-workflow is not supported")
			}

			if len(subNode.Edges) > 0 {
				return nil, fmt.Errorf("nodes in inner-workflow should not have edges info")
			}

			if subNode.Type == entity.NodeTypeBreak.IDStr() || subNode.Type == entity.NodeTypeContinue.IDStr() {
				sc.Connections = append(sc.Connections, &schema.Connection{
					FromNode: vo.NodeKey(subNode.ID),
					ToNode:   vo.NodeKey(subNode.Parent().ID),
				})
			}
		}

		newNode, enableBatch, err := parseBatchMode(node)
		if err != nil {
			return nil, err
		}

		if enableBatch {
			node = newNode
			sc.GeneratedNodes = append(sc.GeneratedNodes, vo.NodeKey(node.Blocks[0].ID))
		}

		nsList, hierarchy, err := NodeToNodeSchema(ctx, node, s)
		if err != nil {
			return nil, err
		}

		sc.Nodes = append(sc.Nodes, nsList...)
		if len(hierarchy) > 0 {
			if sc.Hierarchy == nil {
				sc.Hierarchy = make(map[vo.NodeKey]vo.NodeKey)
			}

			for k, v := range hierarchy {
				sc.Hierarchy[k] = v
			}
		}

		for _, edge := range node.Edges {
			sc.Connections = append(sc.Connections, EdgeToConnection(edge))
		}
	}

	for _, edge := range s.Edges {
		sc.Connections = append(sc.Connections, EdgeToConnection(edge))
	}

	newConnections, err := normalizePorts(sc.Connections, nodeMap)
	if err != nil {
		return nil, err
	}
	sc.Connections = newConnections

	branches, err := schema.BuildBranches(newConnections)
	if err != nil {
		return nil, err
	}

	sc.Branches = branches

	sc.Init()

	return sc, nil
}

func normalizePorts(connections []*schema.Connection, nodeMap map[string]*vo.Node) (normalized []*schema.Connection, err error) {
	for i := range connections {
		conn := connections[i]
		if conn.FromPort == nil {
			normalized = append(normalized, conn)
			continue
		}

		if len(*conn.FromPort) == 0 {
			conn.FromPort = nil
			normalized = append(normalized, conn)
			continue
		}

		if *conn.FromPort == "loop-function-inline-output" || *conn.FromPort == "loop-output" ||
			*conn.FromPort == "batch-function-inline-output" || *conn.FromPort == "batch-output" { // ignore this, we don't need this for inner workflow to work
			conn.FromPort = nil
			normalized = append(normalized, conn)
			continue
		}

		node, ok := nodeMap[string(conn.FromNode)]
		if !ok {
			return nil, fmt.Errorf("node %s not found in node map", conn.FromNode)
		}

		var newPort string
		switch node.Type {
		case entity.NodeTypeSelector.IDStr():
			if *conn.FromPort == "true" {
				newPort = fmt.Sprintf(schema.PortBranchFormat, 0)
			} else if *conn.FromPort == "false" {
				newPort = schema.PortDefault
			} else if strings.HasPrefix(*conn.FromPort, "true_") {
				portN := strings.TrimPrefix(*conn.FromPort, "true_")
				n, err := strconv.Atoi(portN)
				if err != nil {
					return nil, fmt.Errorf("invalid port name: %s", *conn.FromPort)
				}
				newPort = fmt.Sprintf(schema.PortBranchFormat, n)
			}
		default:
			newPort = *conn.FromPort
		}

		normalized = append(normalized, &schema.Connection{
			FromNode: conn.FromNode,
			ToNode:   conn.ToNode,
			FromPort: &newPort,
		})
	}

	return normalized, nil
}

var blockTypeToSkip = map[entity.NodeType]bool{
	entity.NodeTypeComment: true,
}

func NodeToNodeSchema(ctx context.Context, n *vo.Node, c *vo.Canvas) ([]*schema.NodeSchema, map[vo.NodeKey]vo.NodeKey, error) {
	et := entity.IDStrToNodeType(n.Type)

	if et == entity.NodeTypeSubWorkflow {
		ns, err := toSubWorkflowNodeSchema(ctx, n)
		if err != nil {
			return nil, nil, err
		}
		if ns.ExceptionConfigs, err = toExceptionConfig(n, ns.Type); err != nil {
			return nil, nil, err
		}
		return []*schema.NodeSchema{ns}, nil, nil
	}

	na, ok := nodes.GetNodeAdaptor(et)
	if ok {
		ns, err := na.Adapt(ctx, n, nodes.WithCanvas(c))
		if err != nil {
			return nil, nil, err
		}

		if ns.ExceptionConfigs, err = toExceptionConfig(n, ns.Type); err != nil {
			return nil, nil, err
		}

		if len(n.Blocks) > 0 {
			var (
				allNS     []*schema.NodeSchema
				hierarchy = make(map[vo.NodeKey]vo.NodeKey)
			)

			for _, childN := range n.Blocks {
				childN.SetParent(n)
				childNS, _, err := NodeToNodeSchema(ctx, childN, c)
				if err != nil {
					return nil, nil, err
				}

				allNS = append(allNS, childNS...)
				hierarchy[vo.NodeKey(childN.ID)] = vo.NodeKey(n.ID)
			}

			allNS = append(allNS, ns)
			return allNS, hierarchy, nil
		}

		return []*schema.NodeSchema{ns}, nil, nil
	}

	_, ok = blockTypeToSkip[et]
	if ok {
		return nil, nil, nil
	}

	return nil, nil, fmt.Errorf("unsupported block type: %v", n.Type)
}

func EdgeToConnection(e *vo.Edge) *schema.Connection {
	toNode := vo.NodeKey(e.TargetNodeID)
	if len(e.SourcePortID) > 0 && (e.TargetPortID == "loop-function-inline-input" || e.TargetPortID == "batch-function-inline-input") {
		toNode = einoCompose.END
	}

	conn := &schema.Connection{
		FromNode: vo.NodeKey(e.SourceNodeID),
		ToNode:   toNode,
	}

	if len(e.SourceNodeID) > 0 {
		conn.FromPort = &e.SourcePortID
	}

	return conn
}

func toExceptionConfig(n *vo.Node, nType entity.NodeType) (*schema.ExceptionConfig, error) {
	nodeMeta := entity.NodeMetaByNodeType(nType)

	var settingOnErr *vo.SettingOnError

	if n.Data.Inputs != nil {
		settingOnErr = n.Data.Inputs.SettingOnError
	}

	// settingOnErr.Switch seems to be useless, because if set to false, the timeout still takes effect
	if settingOnErr == nil && nodeMeta.DefaultTimeoutMS == 0 {
		return nil, nil
	}

	metaConf := &schema.ExceptionConfig{
		TimeoutMS: nodeMeta.DefaultTimeoutMS,
	}

	if settingOnErr != nil {
		metaConf = &schema.ExceptionConfig{
			TimeoutMS:   settingOnErr.TimeoutMs,
			MaxRetry:    settingOnErr.RetryTimes,
			DataOnErr:   settingOnErr.DataOnErr,
			ProcessType: settingOnErr.ProcessType,
		}

		if metaConf.ProcessType != nil && *metaConf.ProcessType == vo.ErrorProcessTypeReturnDefaultData {
			if len(metaConf.DataOnErr) == 0 {
				return nil, errors.New("error process type is returning default value, but dataOnError is not specified")
			}
		}

		if metaConf.ProcessType == nil && len(metaConf.DataOnErr) > 0 && settingOnErr.Switch {
			metaConf.ProcessType = ptr.Of(vo.ErrorProcessTypeReturnDefaultData)
		}
	}

	return metaConf, nil
}

func toSubWorkflowNodeSchema(ctx context.Context, n *vo.Node) (*schema.NodeSchema, error) {
	idStr := n.Data.Inputs.WorkflowID
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow id: %w", err)
	}

	version := n.Data.Inputs.WorkflowVersion

	subWF, err := workflow.GetRepository().GetEntity(ctx, &vo.GetPolicy{
		ID:      id,
		QType:   ternary.IFElse(len(version) == 0, vo.FromDraft, vo.FromSpecificVersion),
		Version: version,
	})
	if err != nil {
		return nil, err
	}

	var subCanvas vo.Canvas
	if err = sonic.UnmarshalString(subWF.Canvas, &subCanvas); err != nil {
		return nil, err
	}

	subWorkflowSC, err := CanvasToWorkflowSchema(ctx, &subCanvas)
	if err != nil {
		return nil, err
	}

	cfg := &subworkflow.Config{}

	ns := &schema.NodeSchema{
		Key:               vo.NodeKey(n.ID),
		Type:              entity.NodeTypeSubWorkflow,
		Name:              n.Data.Meta.Title,
		SubWorkflowBasic:  subWF.GetBasic(),
		SubWorkflowSchema: subWorkflowSC,
		Configs:           cfg,
	}

	workflowIDStr := n.Data.Inputs.WorkflowID
	if workflowIDStr == "" {
		return nil, fmt.Errorf("sub workflow node's workflowID is empty")
	}
	workflowID, err := strconv.ParseInt(workflowIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("sub workflow node's workflowID is not a number: %s", workflowIDStr)
	}
	cfg.WorkflowID = workflowID
	cfg.WorkflowVersion = n.Data.Inputs.WorkflowVersion

	if err := convert.SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}
	if err := convert.SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}
	return ns, nil
}

func toIntentDetectorSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeIntentDetector,
		Name: n.Data.Meta.Title,
	}

	param := n.Data.Inputs.LLMParam
	if param == nil {
		return nil, fmt.Errorf("intent detector node's llmParam is nil")
	}

	llmParam, ok := param.(vo.IntentDetectorLLMParam)
	if !ok {
		return nil, fmt.Errorf("llm node's llmParam must be LLMParam, got %v", llmParam)
	}

	paramBytes, err := sonic.Marshal(param)
	if err != nil {
		return nil, err
	}
	var intentDetectorConfig = &vo.IntentDetectorLLMConfig{}

	err = sonic.Unmarshal(paramBytes, &intentDetectorConfig)
	if err != nil {
		return nil, err
	}

	modelLLMParams := &model.LLMParams{}
	modelLLMParams.ModelType = int64(intentDetectorConfig.ModelType)
	modelLLMParams.ModelName = intentDetectorConfig.ModelName
	modelLLMParams.TopP = intentDetectorConfig.TopP
	modelLLMParams.Temperature = intentDetectorConfig.Temperature
	modelLLMParams.MaxTokens = intentDetectorConfig.MaxTokens
	modelLLMParams.ResponseFormat = model.ResponseFormat(intentDetectorConfig.ResponseFormat)
	modelLLMParams.SystemPrompt = intentDetectorConfig.SystemPrompt.Value.Content.(string)

	ns.SetConfigKV("LLMParams", modelLLMParams)
	ns.SetConfigKV("SystemPrompt", modelLLMParams.SystemPrompt)

	var intents = make([]string, 0, len(n.Data.Inputs.Intents))
	for _, it := range n.Data.Inputs.Intents {
		intents = append(intents, it.Name)
	}
	ns.SetConfigKV("Intents", intents)

	if n.Data.Inputs.Mode == "top_speed" {
		ns.SetConfigKV("IsFastMode", true)
	}

	if err = SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err = SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toDatabaseCustomSQLSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeDatabaseCustomSQL,
		Name: n.Data.Meta.Title,
	}

	dsList := n.Data.Inputs.DatabaseInfoList
	if len(dsList) == 0 {
		return nil, fmt.Errorf("database info is requird")
	}
	databaseInfo := dsList[0]

	dsID, err := strconv.ParseInt(databaseInfo.DatabaseInfoID, 10, 64)
	if err != nil {
		return nil, err
	}
	ns.SetConfigKV("DatabaseInfoID", dsID)

	sql := n.Data.Inputs.SQL
	if len(sql) == 0 {
		return nil, fmt.Errorf("sql is requird")
	}

	ns.SetConfigKV("SQLTemplate", sql)

	if err = SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err = SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toDatabaseQuerySchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeDatabaseQuery,
		Name: n.Data.Meta.Title,
	}

	dsList := n.Data.Inputs.DatabaseInfoList
	if len(dsList) == 0 {
		return nil, fmt.Errorf("database info is requird")
	}
	databaseInfo := dsList[0]

	dsID, err := strconv.ParseInt(databaseInfo.DatabaseInfoID, 10, 64)
	if err != nil {
		return nil, err
	}
	ns.SetConfigKV("DatabaseInfoID", dsID)

	selectParam := n.Data.Inputs.SelectParam
	ns.SetConfigKV("Limit", selectParam.Limit)

	queryFields := make([]string, 0)
	for _, v := range selectParam.FieldList {
		queryFields = append(queryFields, strconv.FormatInt(v.FieldID, 10))
	}
	ns.SetConfigKV("QueryFields", queryFields)

	orderClauses := make([]*database.OrderClause, 0, len(selectParam.OrderByList))
	for _, o := range selectParam.OrderByList {
		orderClauses = append(orderClauses, &database.OrderClause{
			FieldID: strconv.FormatInt(o.FieldID, 10),
			IsAsc:   o.IsAsc,
		})
	}
	ns.SetConfigKV("OrderClauses", orderClauses)

	clauseGroup := &database.ClauseGroup{}

	if selectParam.Condition != nil {
		clauseGroup, err = buildClauseGroupFromCondition(selectParam.Condition)
		if err != nil {
			return nil, err
		}
	}

	ns.SetConfigKV("ClauseGroup", clauseGroup)

	if err = SetDatabaseInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err = SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toDatabaseInsertSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeDatabaseInsert,
		Name: n.Data.Meta.Title,
	}

	dsList := n.Data.Inputs.DatabaseInfoList
	if len(dsList) == 0 {
		return nil, fmt.Errorf("database info is requird")
	}
	databaseInfo := dsList[0]

	dsID, err := strconv.ParseInt(databaseInfo.DatabaseInfoID, 10, 64)
	if err != nil {
		return nil, err
	}
	ns.SetConfigKV("DatabaseInfoID", dsID)

	if err = SetDatabaseInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err = SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toDatabaseDeleteSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeDatabaseDelete,
		Name: n.Data.Meta.Title,
	}

	dsList := n.Data.Inputs.DatabaseInfoList
	if len(dsList) == 0 {
		return nil, fmt.Errorf("database info is requird")
	}
	databaseInfo := dsList[0]

	dsID, err := strconv.ParseInt(databaseInfo.DatabaseInfoID, 10, 64)
	if err != nil {
		return nil, err
	}
	ns.SetConfigKV("DatabaseInfoID", dsID)

	deleteParam := n.Data.Inputs.DeleteParam

	clauseGroup, err := buildClauseGroupFromCondition(&deleteParam.Condition)
	if err != nil {
		return nil, err
	}
	ns.SetConfigKV("ClauseGroup", clauseGroup)

	if err = SetDatabaseInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err = SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toDatabaseUpdateSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeDatabaseUpdate,
		Name: n.Data.Meta.Title,
	}

	dsList := n.Data.Inputs.DatabaseInfoList
	if len(dsList) == 0 {
		return nil, fmt.Errorf("database info is requird")
	}
	databaseInfo := dsList[0]

	dsID, err := strconv.ParseInt(databaseInfo.DatabaseInfoID, 10, 64)
	if err != nil {
		return nil, err
	}
	ns.SetConfigKV("DatabaseInfoID", dsID)

	updateParam := n.Data.Inputs.UpdateParam
	if updateParam == nil {
		return nil, fmt.Errorf("update param is requird")
	}
	clauseGroup, err := buildClauseGroupFromCondition(&updateParam.Condition)
	if err != nil {
		return nil, err
	}
	ns.SetConfigKV("ClauseGroup", clauseGroup)
	if err = SetDatabaseInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err = SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toHttpRequesterSchema(n *vo.Node, opts ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeHTTPRequester,
		Name: n.Data.Meta.Title,
	}
	option := &option{}
	for _, opt := range opts {
		opt(option)
	}

	implicitNodeDependencies := option.implicitNodeDependencies

	inputs := n.Data.Inputs

	md5FieldMapping := &httprequester.MD5FieldMapping{}

	method := inputs.APIInfo.Method
	ns.SetConfigKV("Method", method)
	url := inputs.APIInfo.URL
	ns.SetConfigKV("URLConfig", httprequester.URLConfig{
		Tpl: strings.TrimSpace(url),
	})

	urlVars := extractBracesContent(url)
	md5FieldMapping.SetURLFields(urlVars...)

	md5FieldMapping.SetHeaderFields(slices.Transform(inputs.Headers, func(a *vo.Param) string {
		return a.Name
	})...)

	md5FieldMapping.SetParamFields(slices.Transform(inputs.Params, func(a *vo.Param) string {
		return a.Name
	})...)

	if inputs.Auth != nil && inputs.Auth.AuthOpen {
		auth := &httprequester.AuthenticationConfig{}
		ty, err := ConvertAuthType(inputs.Auth.AuthType)
		if err != nil {
			return nil, err
		}
		auth.Type = ty
		location, err := ConvertLocation(inputs.Auth.AuthData.CustomData.AddTo)
		if err != nil {
			return nil, err
		}
		auth.Location = location

		ns.SetConfigKV("AuthConfig", auth)

	}

	bodyConfig := httprequester.BodyConfig{}

	bodyConfig.BodyType = httprequester.BodyType(inputs.Body.BodyType)
	switch httprequester.BodyType(inputs.Body.BodyType) {
	case httprequester.BodyTypeJSON:
		jsonTpl := inputs.Body.BodyData.Json
		bodyConfig.TextJsonConfig = &httprequester.TextJsonConfig{
			Tpl: jsonTpl,
		}
		jsonVars := extractBracesContent(jsonTpl)
		md5FieldMapping.SetBodyFields(jsonVars...)
	case httprequester.BodyTypeFormData:
		bodyConfig.FormDataConfig = &httprequester.FormDataConfig{
			FileTypeMapping: map[string]bool{},
		}
		formDataVars := make([]string, 0)
		for i := range inputs.Body.BodyData.FormData.Data {
			p := inputs.Body.BodyData.FormData.Data[i]
			if p.Input.Type == vo.VariableTypeString && p.Input.AssistType > vo.AssistTypeNotSet && p.Input.AssistType < vo.AssistTypeTime {
				bodyConfig.FormDataConfig.FileTypeMapping[p.Name] = true
				formDataVars = append(formDataVars, p.Name)
			}
		}
		md5FieldMapping.SetBodyFields(formDataVars...)
	case httprequester.BodyTypeRawText:
		TextTpl := inputs.Body.BodyData.RawText
		bodyConfig.TextPlainConfig = &httprequester.TextPlainConfig{
			Tpl: TextTpl,
		}
		textPlainVars := extractBracesContent(TextTpl)
		md5FieldMapping.SetBodyFields(textPlainVars...)
	case httprequester.BodyTypeFormURLEncoded:
		formURLEncodedVars := make([]string, 0)
		for _, p := range inputs.Body.BodyData.FormURLEncoded {
			formURLEncodedVars = append(formURLEncodedVars, p.Name)
		}
		md5FieldMapping.SetBodyFields(formURLEncodedVars...)
	}
	ns.SetConfigKV("BodyConfig", bodyConfig)
	ns.SetConfigKV("MD5FieldMapping", *md5FieldMapping)

	if inputs.Setting != nil {
		ns.SetConfigKV("Timeout", time.Duration(inputs.Setting.Timeout)*time.Second)
		ns.SetConfigKV("RetryTimes", uint64(inputs.Setting.RetryTimes))
	}

	if err := SetHttpRequesterInputsForNodeSchema(n, ns, implicitNodeDependencies); err != nil {
		return nil, err
	}
	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}
	return ns, nil
}

func toKnowledgeIndexerSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeKnowledgeIndexer,
		Name: n.Data.Meta.Title,
	}

	inputs := n.Data.Inputs
	datasetListInfoParam := inputs.DatasetParam[0]
	datasetIDs := datasetListInfoParam.Input.Value.Content.([]any)
	if len(datasetIDs) == 0 {
		return nil, fmt.Errorf("dataset ids is required")
	}
	knowledgeID, err := cast.ToInt64E(datasetIDs[0])
	if err != nil {
		return nil, err
	}

	ns.SetConfigKV("KnowledgeID", knowledgeID)
	ps := inputs.StrategyParam.ParsingStrategy
	parseMode, err := ConvertParsingType(ps.ParsingType)
	if err != nil {
		return nil, err
	}
	parsingStrategy := &knowledge.ParsingStrategy{
		ParseMode:    parseMode,
		ImageOCR:     ps.ImageOcr,
		ExtractImage: ps.ImageExtraction,
		ExtractTable: ps.TableExtraction,
	}

	ns.SetConfigKV("ParsingStrategy", parsingStrategy)
	cs := inputs.StrategyParam.ChunkStrategy
	chunkType, err := ConvertChunkType(cs.ChunkType)
	if err != nil {
		return nil, err
	}
	chunkingStrategy := &knowledge.ChunkingStrategy{
		ChunkType: chunkType,
		Separator: cs.Separator,
		ChunkSize: cs.MaxToken,
		Overlap:   int64(cs.Overlap * float64(cs.MaxToken)),
	}
	ns.SetConfigKV("ChunkingStrategy", chunkingStrategy)

	if err = SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err = SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toKnowledgeRetrieverSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeKnowledgeRetriever,
		Name: n.Data.Meta.Title,
	}

	inputs := n.Data.Inputs
	datasetListInfoParam := inputs.DatasetParam[0]
	datasetIDs := datasetListInfoParam.Input.Value.Content.([]any)
	knowledgeIDs := make([]int64, 0, len(datasetIDs))
	for _, id := range datasetIDs {
		k, err := cast.ToInt64E(id)
		if err != nil {
			return nil, err
		}
		knowledgeIDs = append(knowledgeIDs, k)
	}
	ns.SetConfigKV("KnowledgeIDs", knowledgeIDs)

	retrievalStrategy := &knowledge.RetrievalStrategy{}

	var getDesignatedParamContent = func(name string) (any, bool) {
		for _, param := range inputs.DatasetParam {
			if param.Name == name {
				return param.Input.Value.Content, true
			}
		}
		return nil, false

	}

	if content, ok := getDesignatedParamContent("topK"); ok {
		topK, err := cast.ToInt64E(content)
		if err != nil {
			return nil, err
		}
		retrievalStrategy.TopK = &topK
	}

	if content, ok := getDesignatedParamContent("useRerank"); ok {
		useRerank, err := cast.ToBoolE(content)
		if err != nil {
			return nil, err
		}
		retrievalStrategy.EnableRerank = useRerank
	}

	if content, ok := getDesignatedParamContent("useRewrite"); ok {
		useRewrite, err := cast.ToBoolE(content)
		if err != nil {
			return nil, err
		}
		retrievalStrategy.EnableQueryRewrite = useRewrite
	}

	if content, ok := getDesignatedParamContent("isPersonalOnly"); ok {
		isPersonalOnly, err := cast.ToBoolE(content)
		if err != nil {
			return nil, err
		}
		retrievalStrategy.IsPersonalOnly = isPersonalOnly
	}

	if content, ok := getDesignatedParamContent("useNl2sql"); ok {
		useNl2sql, err := cast.ToBoolE(content)
		if err != nil {
			return nil, err
		}
		retrievalStrategy.EnableNL2SQL = useNl2sql
	}

	if content, ok := getDesignatedParamContent("minScore"); ok {
		minScore, err := cast.ToFloat64E(content)
		if err != nil {
			return nil, err
		}
		retrievalStrategy.MinScore = &minScore
	}

	if content, ok := getDesignatedParamContent("strategy"); ok {
		strategy, err := cast.ToInt64E(content)
		if err != nil {
			return nil, err
		}
		searchType, err := ConvertRetrievalSearchType(strategy)
		if err != nil {
			return nil, err
		}
		retrievalStrategy.SearchType = searchType
	}

	ns.SetConfigKV("RetrievalStrategy", retrievalStrategy)

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toKnowledgeDeleterSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeKnowledgeDeleter,
		Name: n.Data.Meta.Title,
	}

	inputs := n.Data.Inputs
	datasetListInfoParam := inputs.DatasetParam[0]
	datasetIDs := datasetListInfoParam.Input.Value.Content.([]any)
	if len(datasetIDs) == 0 {
		return nil, fmt.Errorf("dataset ids is required")
	}
	knowledgeID, err := cast.ToInt64E(datasetIDs[0])
	if err != nil {
		return nil, err
	}

	ns.SetConfigKV("KnowledgeID", knowledgeID)

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toVariableAssignerSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeVariableAssigner,
		Name: n.Data.Meta.Title,
	}

	var pairs = make([]*variableassigner.Pair, 0, len(n.Data.Inputs.InputParameters))
	for i, param := range n.Data.Inputs.InputParameters {
		if param.Left == nil || param.Input == nil {
			return nil, fmt.Errorf("variable assigner node's param left or input is nil")
		}

		leftSources, err := CanvasBlockInputToFieldInfo(param.Left, einoCompose.FieldPath{fmt.Sprintf("left_%d", i)}, n.Parent())
		if err != nil {
			return nil, err
		}

		if leftSources[0].Source.Ref == nil {
			return nil, fmt.Errorf("variable assigner node's param left source ref is nil")
		}

		if leftSources[0].Source.Ref.VariableType == nil {
			return nil, fmt.Errorf("variable assigner node's param left source ref's variable type is nil")
		}

		if *leftSources[0].Source.Ref.VariableType == vo.GlobalSystem {
			return nil, fmt.Errorf("variable assigner node's param left's ref's variable type cannot be variable.GlobalSystem")
		}

		inputSource, err := CanvasBlockInputToFieldInfo(param.Input, leftSources[0].Source.Ref.FromPath, n.Parent())
		if err != nil {
			return nil, err
		}
		ns.AddInputSource(inputSource...)
		pair := &variableassigner.Pair{
			Left:  *leftSources[0].Source.Ref,
			Right: inputSource[0].Path,
		}
		pairs = append(pairs, pair)
	}
	ns.Configs = pairs

	return ns, nil
}

func toCodeRunnerSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeCodeRunner,
		Name: n.Data.Meta.Title,
	}
	inputs := n.Data.Inputs

	code := inputs.Code
	ns.SetConfigKV("Code", code)

	language, err := ConvertCodeLanguage(inputs.Language)
	if err != nil {
		return nil, err
	}
	ns.SetConfigKV("Language", language)

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toPluginSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypePlugin,
		Name: n.Data.Meta.Title,
	}
	inputs := n.Data.Inputs

	apiParams := slices.ToMap(inputs.APIParams, func(e *vo.Param) (string, *vo.Param) {
		return e.Name, e
	})

	ps, ok := apiParams["pluginID"]
	if !ok {
		return nil, fmt.Errorf("plugin id param is not found")
	}

	pID, err := strconv.ParseInt(ps.Input.Value.Content.(string), 10, 64)

	ns.SetConfigKV("PluginID", pID)

	ps, ok = apiParams["apiID"]
	if !ok {
		return nil, fmt.Errorf("plugin id param is not found")
	}

	tID, err := strconv.ParseInt(ps.Input.Value.Content.(string), 10, 64)
	if err != nil {
		return nil, err
	}

	ns.SetConfigKV("ToolID", tID)

	ps, ok = apiParams["pluginVersion"]
	if !ok {
		return nil, fmt.Errorf("plugin version param is not found")
	}
	version := ps.Input.Value.Content.(string)
	ns.SetConfigKV("PluginVersion", version)

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toVariableAggregatorSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeVariableAggregator,
		Name: n.Data.Meta.Title,
	}

	ns.SetConfigKV("MergeStrategy", variableaggregator.FirstNotNullValue)
	inputs := n.Data.Inputs

	groupToLen := make(map[string]int, len(inputs.VariableAggregator.MergeGroups))
	for i := range inputs.VariableAggregator.MergeGroups {
		group := inputs.VariableAggregator.MergeGroups[i]
		tInfo := &vo.TypeInfo{
			Type:       vo.DataTypeObject,
			Properties: make(map[string]*vo.TypeInfo),
		}
		ns.SetInputType(group.Name, tInfo)
		for ii, v := range group.Variables {
			name := strconv.Itoa(ii)
			valueTypeInfo, err := CanvasBlockInputToTypeInfo(v)
			if err != nil {
				return nil, err
			}
			tInfo.Properties[name] = valueTypeInfo
			sources, err := CanvasBlockInputToFieldInfo(v, einoCompose.FieldPath{group.Name, name}, n.Parent())
			if err != nil {
				return nil, err
			}
			ns.AddInputSource(sources...)
		}

		length := len(group.Variables)
		groupToLen[group.Name] = length
	}

	groupOrder := make([]string, 0, len(groupToLen))
	for i := range inputs.VariableAggregator.MergeGroups {
		group := inputs.VariableAggregator.MergeGroups[i]
		groupOrder = append(groupOrder, group.Name)
	}

	ns.SetConfigKV("GroupToLen", groupToLen)
	ns.SetConfigKV("GroupOrder", groupOrder)

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}
	return ns, nil
}

func toInputReceiverSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeInputReceiver,
		Name: n.Data.Meta.Title,
	}

	ns.SetConfigKV("OutputSchema", n.Data.Inputs.OutputSchema)

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toQASchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeQuestionAnswer,
		Name: n.Data.Meta.Title,
	}

	qaConf := n.Data.Inputs.QA
	if qaConf == nil {
		return nil, fmt.Errorf("qa config is nil")
	}
	ns.SetConfigKV("QuestionTpl", qaConf.Question)

	var llmParams *model.LLMParams
	if n.Data.Inputs.LLMParam != nil {
		llmParamBytes, err := sonic.Marshal(n.Data.Inputs.LLMParam)
		if err != nil {
			return nil, err
		}
		var qaLLMParams vo.QALLMParam
		err = sonic.Unmarshal(llmParamBytes, &qaLLMParams)
		if err != nil {
			return nil, err
		}

		llmParams, err = qaLLMParamsToLLMParams(qaLLMParams)
		if err != nil {
			return nil, err
		}

		ns.SetConfigKV("LLMParams", llmParams)
	}

	answerType, err := qaAnswerTypeToAnswerType(qaConf.AnswerType)
	if err != nil {
		return nil, err
	}
	ns.SetConfigKV("AnswerType", answerType)

	var choiceType qa.ChoiceType
	if len(qaConf.OptionType) > 0 {
		choiceType, err = qaOptionTypeToChoiceType(qaConf.OptionType)
		if err != nil {
			return nil, err
		}
		ns.SetConfigKV("ChoiceType", choiceType)
	}

	if answerType == qa.AnswerByChoices {
		switch choiceType {
		case qa.FixedChoices:
			var options []string
			for _, option := range qaConf.Options {
				options = append(options, option.Name)
			}
			ns.SetConfigKV("FixedChoices", options)
		case qa.DynamicChoices:
			inputSources, err := CanvasBlockInputToFieldInfo(qaConf.DynamicOption, einoCompose.FieldPath{qa.DynamicChoicesKey}, n.Parent())
			if err != nil {
				return nil, err
			}
			ns.AddInputSource(inputSources...)

			inputTypes, err := CanvasBlockInputToTypeInfo(qaConf.DynamicOption)
			if err != nil {
				return nil, err
			}
			ns.SetInputType(qa.DynamicChoicesKey, inputTypes)
		default:
			return nil, fmt.Errorf("qa node is answer by options, but option type not provided")
		}
	} else if answerType == qa.AnswerDirectly {
		ns.SetConfigKV("ExtractFromAnswer", qaConf.ExtractOutput)
		if qaConf.ExtractOutput {
			if llmParams == nil {
				return nil, fmt.Errorf("qa node needs to extract from answer, but LLMParams not provided")
			}
			ns.SetConfigKV("AdditionalSystemPromptTpl", llmParams.SystemPrompt)
			ns.SetConfigKV("MaxAnswerCount", qaConf.Limit)
			if err = SetOutputTypesForNodeSchema(n, ns); err != nil {
				return nil, err
			}
		}
	}

	if err = SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toJSONSerializeSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeJsonSerialization,
		Name: n.Data.Meta.Title,
	}

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toJSONDeserializeSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeJsonDeserialization,
		Name: n.Data.Meta.Title,
	}

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toCreateConversationSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeCreateConversation,
		Name: n.Data.Meta.Title,
	}
	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toConversationUpdateSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeConversationUpdate,
		Name: n.Data.Meta.Title,
	}
	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toConversationListSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeConversationList,
		Name: n.Data.Meta.Title,
	}

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toCreateMessageSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeCreateMessage,
		Name: n.Data.Meta.Title,
	}

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toMessageListSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeMessageList,
		Name: n.Data.Meta.Title,
	}

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toDeleteMessageSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeDeleteMessage,
		Name: n.Data.Meta.Title,
	}

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toEditMessageSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeEditMessage,
		Name: n.Data.Meta.Title,
	}

	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toConversationHistorySchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeConversationHistory,
		Name: n.Data.Meta.Title,
	}
	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}
	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toClearConversationHistorySchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeClearConversationHistory,
		Name: n.Data.Meta.Title,
	}
	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func toConversationDeleteSchema(n *vo.Node, _ ...OptionFn) (*compose.NodeSchema, error) {
	ns := &compose.NodeSchema{
		Key:  vo.NodeKey(n.ID),
		Type: entity.NodeTypeConversationDelete,
		Name: n.Data.Meta.Title,
	}
	if err := SetInputsForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	if err := SetOutputTypesForNodeSchema(n, ns); err != nil {
		return nil, err
	}

	return ns, nil
}

func buildClauseGroupFromCondition(condition *vo.DBCondition) (*database.ClauseGroup, error) {
	clauseGroup := &database.ClauseGroup{}
	if len(condition.ConditionList) == 1 {
		params := condition.ConditionList[0]
		clause, err := buildClauseFromParams(params)
		if err != nil {
			return nil, err
		}
		clauseGroup.Single = clause
	} else {
		relation, err := ConvertLogicTypeToRelation(condition.Logic)
		if err != nil {
			return nil, err
		}
		clauseGroup.Multi = &database.MultiClause{
			Clauses:  make([]*database.Clause, 0, len(condition.ConditionList)),
			Relation: relation,
		}
		for i := range condition.ConditionList {
			params := condition.ConditionList[i]
			clause, err := buildClauseFromParams(params)
			if err != nil {
				return nil, err
			}
			clauseGroup.Multi.Clauses = append(clauseGroup.Multi.Clauses, clause)
		}
	}

	return clauseGroup, nil
}

func PruneIsolatedNodes(nodes []*vo.Node, edges []*vo.Edge, parentNode *vo.Node) ([]*vo.Node, []*vo.Edge) {
	nodeDependencyCount := map[string]int{}
	if parentNode != nil {
		nodeDependencyCount[parentNode.ID] = 0
	}
	for _, node := range nodes {
		if len(node.Blocks) > 0 && len(node.Edges) > 0 {
			node.Blocks, node.Edges = PruneIsolatedNodes(node.Blocks, node.Edges, node)
		}
		nodeDependencyCount[node.ID] = 0
		if node.Type == entity.NodeTypeContinue.IDStr() || node.Type == entity.NodeTypeBreak.IDStr() {
			if parentNode != nil {
				nodeDependencyCount[parentNode.ID]++
			}
		}
	}

	nodeDependencyCount[entity.EntryNodeKey] = 1 // entry node is considered to be 1
	nodeDependencyCount[entity.ExitNodeKey] = 1  // exit node is considered to be 1
	for _, edge := range edges {
		if _, ok := nodeDependencyCount[edge.TargetNodeID]; ok {
			nodeDependencyCount[edge.TargetNodeID]++
		} else {
			panic(fmt.Errorf("node id %v not existed, but appears in the edge", edge.TargetNodeID))
		}
	}

	isolatedNodeIDs := make(map[string]struct{})
	for nodeId, count := range nodeDependencyCount {
		if count == 0 {
			isolatedNodeIDs[nodeId] = struct{}{}
		}
	}

	connectedNodes := make([]*vo.Node, 0)
	for _, node := range nodes {
		if _, ok := isolatedNodeIDs[node.ID]; !ok {
			connectedNodes = append(connectedNodes, node)
		}
	}

	connectedEdges := make([]*vo.Edge, 0)
	for _, edge := range edges {
		if _, ok := isolatedNodeIDs[edge.SourceNodeID]; !ok {
			connectedEdges = append(connectedEdges, edge)
		}
	}

	return connectedNodes, connectedEdges
}

func parseBatchMode(n *vo.Node) (
	batchN *vo.Node, // the new batch node
	enabled bool, // whether the node has enabled batch mode
	err error) {
	if n.Data == nil || n.Data.Inputs == nil {
		return nil, false, nil
	}

	batchInfo := n.Data.Inputs.NodeBatchInfo
	if batchInfo == nil || !batchInfo.BatchEnable {
		return nil, false, nil
	}

	enabled = true

	var (
		innerOutput []*vo.Variable
		outerOutput []*vo.Param
		innerInput  = n.Data.Inputs.InputParameters // inputs come from parent batch node or predecessors of parent
		outerInput  = n.Data.Inputs.NodeBatchInfo.InputLists
	)

	if len(n.Data.Outputs) != 1 {
		return nil, false, fmt.Errorf("node batch mode output should be one list, actual count: %d", len(n.Data.Outputs))
	}

	out := n.Data.Outputs[0] // extract original output type info from batch output list

	v, err := vo.ParseVariable(out)
	if err != nil {
		return nil, false, err
	}

	if v.Type != vo.VariableTypeList {
		return nil, false, fmt.Errorf("node batch mode output should be list, actual type: %s", v.Type)
	}

	objV, err := vo.ParseVariable(v.Schema)
	if err != nil {
		return nil, false, fmt.Errorf("node batch mode output schema should be variable, parse err: %w", err)
	}

	if objV.Type != vo.VariableTypeObject {
		return nil, false, fmt.Errorf("node batch mode output element should be object, actual type: %s", objV.Type)
	}

	objFieldStr, err := sonic.MarshalString(objV.Schema)
	if err != nil {
		return nil, false, err
	}

	err = sonic.UnmarshalString(objFieldStr, &innerOutput)
	if err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal obj schema into variable list: %w", err)
	}

	outerOutputP := &vo.Param{ // convert batch output from vo.Variable to vo.Param, adding field mapping
		Name: v.Name,
		Input: &vo.BlockInput{
			Type:   vo.VariableTypeList,
			Schema: objV,
			Value: &vo.BlockInputValue{
				Type: vo.BlockInputValueTypeRef,
				Content: &vo.BlockInputReference{
					Source:  vo.RefSourceTypeBlockOutput,
					BlockID: vo.GenerateNodeIDForBatchMode(n.ID),
					Name:    "", // keep this empty to signal an all out mapping
				},
			},
		},
	}

	outerOutput = append(outerOutput, outerOutputP)

	parentN := &vo.Node{
		ID:   n.ID,
		Type: entity.NodeTypeBatch.IDStr(),
		Data: &vo.Data{
			Meta: &vo.NodeMetaFE{
				Title: n.Data.Meta.Title,
			},
			Inputs: &vo.Inputs{
				InputParameters: outerInput,
				Batch: &vo.Batch{
					BatchSize: &vo.BlockInput{
						Type: vo.VariableTypeInteger,
						Value: &vo.BlockInputValue{
							Type:    vo.BlockInputValueTypeLiteral,
							Content: strconv.FormatInt(batchInfo.BatchSize, 10),
						},
					},
					ConcurrentSize: &vo.BlockInput{
						Type: vo.VariableTypeInteger,
						Value: &vo.BlockInputValue{
							Type:    vo.BlockInputValueTypeLiteral,
							Content: strconv.FormatInt(batchInfo.ConcurrentSize, 10),
						},
					},
				},
			},
			Outputs: slices.Transform(outerOutput, func(a *vo.Param) any {
				return a
			}),
		},
	}

	innerN := &vo.Node{
		ID:   n.ID + "_inner",
		Type: n.Type,
		Data: &vo.Data{
			Meta: &vo.NodeMetaFE{
				Title: n.Data.Meta.Title + "_inner",
			},
			Inputs: &vo.Inputs{
				InputParameters: innerInput,
				LLMParam:        n.Data.Inputs.LLMParam,       // for llm node
				LLM:             n.Data.Inputs.LLM,            // for llm node
				SettingOnError:  n.Data.Inputs.SettingOnError, // for llm, sub-workflow and plugin nodes
				SubWorkflow:     n.Data.Inputs.SubWorkflow,    // for sub-workflow node
				PluginAPIParam:  n.Data.Inputs.PluginAPIParam, // for plugin node
			},
			Outputs: slices.Transform(innerOutput, func(a *vo.Variable) any {
				return a
			}),
		},
	}

	parentN.Blocks = []*vo.Node{innerN}
	parentN.Edges = []*vo.Edge{
		{
			SourceNodeID: parentN.ID,
			TargetNodeID: innerN.ID,
			SourcePortID: "batch-function-inline-output",
		},
		{
			SourceNodeID: innerN.ID,
			TargetNodeID: parentN.ID,
			TargetPortID: "batch-function-inline-input",
		},
	}

	innerN.SetParent(parentN)

	return parentN, true, nil
}

// RegisterAllNodeAdaptors register all NodeType's NodeAdaptor.
func RegisterAllNodeAdaptors() {
	// register a generator function so that each time a NodeAdaptor is needed,
	// we can provide a brand new Config instance.
	nodes.RegisterNodeAdaptor(entity.NodeTypeEntry, func() nodes.NodeAdaptor {
		return &entry.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeSelector, func() nodes.NodeAdaptor {
		return &selector.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeBatch, func() nodes.NodeAdaptor {
		return &batch.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeBreak, func() nodes.NodeAdaptor {
		return &_break.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeContinue, func() nodes.NodeAdaptor {
		return &_continue.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeInputReceiver, func() nodes.NodeAdaptor {
		return &receiver.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeJsonSerialization, func() nodes.NodeAdaptor {
		return &json.SerializationConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeJsonDeserialization, func() nodes.NodeAdaptor {
		return &json.DeserializationConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeVariableAssigner, func() nodes.NodeAdaptor {
		return &variableassigner.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeVariableAssignerWithinLoop, func() nodes.NodeAdaptor {
		return &variableassigner.InLoopConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypePlugin, func() nodes.NodeAdaptor {
		return &plugin.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeCodeRunner, func() nodes.NodeAdaptor {
		return &code.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeOutputEmitter, func() nodes.NodeAdaptor {
		return &emitter.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeExit, func() nodes.NodeAdaptor {
		return &exit.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeVariableAggregator, func() nodes.NodeAdaptor {
		return &variableaggregator.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeTextProcessor, func() nodes.NodeAdaptor {
		return &textprocessor.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeIntentDetector, func() nodes.NodeAdaptor {
		return &intentdetector.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeQuestionAnswer, func() nodes.NodeAdaptor {
		return &qa.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeHTTPRequester, func() nodes.NodeAdaptor {
		return &httprequester.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeLoop, func() nodes.NodeAdaptor {
		return &loop.Config{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeKnowledgeIndexer, func() nodes.NodeAdaptor {
		return &knowledge.IndexerConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeKnowledgeRetriever, func() nodes.NodeAdaptor {
		return &knowledge.RetrieveConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeKnowledgeDeleter, func() nodes.NodeAdaptor {
		return &knowledge.DeleterConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeDatabaseInsert, func() nodes.NodeAdaptor {
		return &database.InsertConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeDatabaseUpdate, func() nodes.NodeAdaptor {
		return &database.UpdateConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeDatabaseQuery, func() nodes.NodeAdaptor {
		return &database.QueryConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeDatabaseDelete, func() nodes.NodeAdaptor {
		return &database.DeleteConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeDatabaseCustomSQL, func() nodes.NodeAdaptor {
		return &database.CustomSQLConfig{}
	})
	nodes.RegisterNodeAdaptor(entity.NodeTypeLLM, func() nodes.NodeAdaptor {
		return &llm.Config{}
	})

	// register branch adaptors
	nodes.RegisterBranchAdaptor(entity.NodeTypeSelector, func() nodes.BranchAdaptor {
		return &selector.Config{}
	})
	nodes.RegisterBranchAdaptor(entity.NodeTypeIntentDetector, func() nodes.BranchAdaptor {
		return &intentdetector.Config{}
	})
	nodes.RegisterBranchAdaptor(entity.NodeTypeQuestionAnswer, func() nodes.BranchAdaptor {
		return &qa.Config{}
	})
}
