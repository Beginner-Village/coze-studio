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

// 能力 E:智能体自建数据库表(含字段定义)+ 可选写入 mock 行,返回真实 database_info_id + field_id,
// 直接用于数据库节点(12/42/43/44/46)。走 crossdatabase 合约 CreateDatabase/AddDatabaseRecord,不碰 application/memory,不成环。

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	dbmodel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/database"
	tabledata "github.com/ynet-dev/ynet-studio/backend/api/model/data/database/table"
	crossdatabase "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/database"
	dbservice "github.com/ynet-dev/ynet-studio/backend/domain/memory/database/service"
)

func init() {
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasCreateDatabaseTool{deps: deps}
	})
}

type wfCanvasCreateDatabaseTool struct{ deps superAgentToolDeps }

func (t *wfCanvasCreateDatabaseTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_create_database",
		Desc: "在当前空间【新建一张数据库表】(含字段定义),可选地写入若干 mock 行,返回真实 database_info_id 和字段 field_id,可直接用于数据库节点(查询43/新增46/更新42/删除44/SQL12)。字段 type 取值: string/number/float/date/boolean。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"table_name":  {Type: schema.String, Desc: "表名(英文/数字/下划线)", Required: true},
			"description": {Type: schema.String, Desc: "表说明", Required: false},
			"fields":      {Type: schema.String, Desc: `字段定义 JSON 数组,如 [{"name":"user_id","type":"string","required":true},{"name":"balance","type":"number"}]`, Required: true},
			"records":     {Type: schema.String, Desc: `可选,mock 行 JSON 数组(值都用字符串),如 [{"user_id":"u1","balance":"100"}]`, Required: false},
		}),
	}, nil
}

func (t *wfCanvasCreateDatabaseTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		TableName   string `json:"table_name"`
		Description string `json:"description"`
		Fields      string `json:"fields"`
		Records     string `json:"records"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return argParseErrMsg(err), nil
	}
	if args.TableName == "" {
		return wfCanvasDBErr("table_name 不能为空"), nil
	}
	space := wfCanvasSpaceID(t.deps)
	if space == 0 {
		return wfCanvasDBErr("无法确定 space_id"), nil
	}

	var fieldDefs []struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Required bool   `json:"required"`
		Desc     string `json:"desc"`
	}
	if err := json.Unmarshal([]byte(args.Fields), &fieldDefs); err != nil || len(fieldDefs) == 0 {
		return wfCanvasDBErr("fields 解析失败或为空,需 JSON 数组 [{name,type,required?}]"), nil
	}
	fieldList := make([]*dbmodel.FieldItem, 0, len(fieldDefs))
	for _, f := range fieldDefs {
		if f.Name == "" {
			continue
		}
		fieldList = append(fieldList, &dbmodel.FieldItem{
			Name:         f.Name,
			Desc:         f.Desc,
			Type:         wfDBFieldTypeFromStr(f.Type),
			MustRequired: f.Required,
		})
	}
	if len(fieldList) == 0 {
		return wfCanvasDBErr("没有有效字段"), nil
	}

	resp, err := crossdatabase.DefaultSVC().CreateDatabase(ctx, &dbservice.CreateDatabaseRequest{
		Database: &dbmodel.Database{
			SpaceID:   space,
			CreatorID: t.deps.UserID,
			TableName: args.TableName,
			TableDesc: args.Description,
			FieldList: fieldList,
		},
	})
	if err != nil {
		return wfCanvasDBErr("建表失败: " + err.Error()), nil
	}
	if resp == nil || resp.Database == nil {
		return wfCanvasDBErr("建表未返回数据库"), nil
	}
	db := resp.Database

	addedRows := 0
	if args.Records != "" {
		var records []map[string]string
		if jerr := json.Unmarshal([]byte(args.Records), &records); jerr == nil && len(records) > 0 {
			if recErr := crossdatabase.DefaultSVC().AddDatabaseRecord(ctx, &dbservice.AddDatabaseRecordRequest{
				DatabaseID: db.ID,
				TableType:  tabledata.TableType_OnlineTable,
				UserID:     t.deps.UserID,
				Records:    records,
			}); recErr == nil {
				addedRows = len(records)
			}
		}
	}

	type fieldOut struct {
		FieldID int64  `json:"field_id"`
		Name    string `json:"name"`
		Type    string `json:"type"`
	}
	fields := make([]fieldOut, 0, len(db.FieldList))
	for _, f := range db.FieldList {
		if f == nil || f.IsSystemField {
			continue
		}
		fields = append(fields, fieldOut{FieldID: f.AlterID, Name: f.Name, Type: wfDBFieldType(f.Type)})
	}
	b, _ := json.Marshal(map[string]any{
		"status":           "ok",
		"database_info_id": strconv.FormatInt(db.ID, 10),
		"table_name":       db.TableName,
		"fields":           fields,
		"added_rows":       addedRows,
		"instruction":      "用 database_info_id 作为数据库节点 config 的 table_id;过滤/写入字段用上面返回的 field_id。",
	})
	return string(b), nil
}

func wfDBFieldTypeFromStr(s string) tabledata.FieldItemType {
	switch s {
	case "number", "integer", "int":
		return tabledata.FieldItemType_Number
	case "float", "double":
		return tabledata.FieldItemType_Float
	case "boolean", "bool":
		return tabledata.FieldItemType_Boolean
	case "date", "datetime":
		return tabledata.FieldItemType_Date
	default:
		return tabledata.FieldItemType_Text
	}
}
