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

// 能力:列出当前空间的数据库表及字段 schema,供智能体配置数据库节点(type 12/42/43/44/46)时
// 拿到真实 database_info_id(表ID)+ field_id,禁止编造。走 crossdatabase 合约(不碰 application/memory,不成环)。

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	tabledata "github.com/ynet-dev/ynet-studio/backend/api/model/data/database/table"
	crossdatabase "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/database"
	dbservice "github.com/ynet-dev/ynet-studio/backend/domain/memory/database/service"
)

func init() {
	registerSuperAgentExtension(func(deps superAgentToolDeps) tool.InvokableTool {
		return &wfCanvasListDatabasesTool{deps: deps}
	})
}

type wfCanvasListDatabasesTool struct{ deps superAgentToolDeps }

func (t *wfCanvasListDatabasesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "workflow_canvas_list_databases",
		Desc: "列出当前空间可用的数据库表及字段 schema,返回真实 database_info_id(表ID)和每个字段的 field_id/name/type/required。" +
			"配置数据库节点(查询43/新增46/更新42/删除44/SQL12)前必须先调用它取真实 ID,禁止编造。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"keyword": {Type: schema.String, Desc: "可选,按表名过滤", Required: false},
		}),
	}, nil
}

func (t *wfCanvasListDatabasesTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Keyword string `json:"keyword"`
	}
	_ = json.Unmarshal([]byte(argumentsInJSON), &args)
	space := wfCanvasSpaceID(t.deps)
	if space == 0 {
		return wfCanvasDBErr("无法确定 space_id"), nil
	}

	req := &dbservice.ListDatabaseRequest{
		SpaceID:   &space,
		TableType: tabledata.TableType_OnlineTable,
		Limit:     50,
	}
	if args.Keyword != "" {
		req.TableName = &args.Keyword
	}
	resp, err := crossdatabase.DefaultSVC().ListDatabase(ctx, req)
	if err != nil {
		return wfCanvasDBErr("列数据库失败: " + err.Error()), nil
	}

	type fieldOut struct {
		FieldID  int64  `json:"field_id"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Required bool   `json:"required"`
	}
	type dbOut struct {
		DatabaseInfoID string     `json:"database_info_id"`
		TableName      string     `json:"table_name"`
		TableDesc      string     `json:"table_desc,omitempty"`
		Fields         []fieldOut `json:"fields"`
	}
	out := make([]dbOut, 0)
	if resp != nil {
		for _, db := range resp.Databases {
			if db == nil {
				continue
			}
			d := dbOut{
				DatabaseInfoID: strconv.FormatInt(db.ID, 10),
				TableName:      db.TableName,
				TableDesc:      db.TableDesc,
			}
			for _, f := range db.FieldList {
				if f == nil || f.IsSystemField {
					continue
				}
				d.Fields = append(d.Fields, fieldOut{
					FieldID:  f.AlterID,
					Name:     f.Name,
					Type:     wfDBFieldType(f.Type),
					Required: f.MustRequired,
				})
			}
			out = append(out, d)
		}
	}

	b, _ := json.Marshal(map[string]any{
		"status":    "ok",
		"count":     len(out),
		"databases": out,
		"instruction": "用 database_info_id 作为数据库节点 config 的 table_id;条件/字段用 field_id。" +
			"查询用 fields(字段名数组)+conditions[{left:字段名,operator,right}];新增/更新用 values[{field_id,value}];SQL 用 {{变量}} 占位。",
	})
	return string(b), nil
}

func wfDBFieldType(t tabledata.FieldItemType) string {
	switch t {
	case tabledata.FieldItemType_Number:
		return "integer"
	case tabledata.FieldItemType_Float:
		return "number"
	case tabledata.FieldItemType_Boolean:
		return "boolean"
	case tabledata.FieldItemType_Date:
		return "date"
	default:
		return "string"
	}
}
