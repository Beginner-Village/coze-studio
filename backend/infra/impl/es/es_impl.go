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

package es

import (
	"fmt"
	"os"
	"strings"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/es"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

type (
	Client          = es.Client
	Types           = es.Types
	BulkIndexer     = es.BulkIndexer
	BulkIndexerItem = es.BulkIndexerItem
	BoolQuery       = es.BoolQuery
	Query           = es.Query
	Response        = es.Response
	Request         = es.Request
)

// New 创建 ES 客户端，支持双写模式
// 环境变量:
//   - ES_VERSION: "v7" 或 "v8"
//   - ES_ADDR: 主 ES 地址
//   - ES_ADDRS: 双写地址列表，逗号分隔（如 "http://es1:9200,http://es2:9200"）
func New() (Client, error) {
	v := os.Getenv("ES_VERSION")

	// 检查是否有双写配置
	esAddrs := os.Getenv("ES_ADDRS")
	if esAddrs != "" {
		return newMulti(v, esAddrs)
	}

	return newSingle(v)
}

func newSingle(version string) (Client, error) {
	switch version {
	case "v8":
		return newES8()
	case "v7":
		return newES7()
	default:
		return nil, fmt.Errorf("unsupported es version %s", version)
	}
}

// newMulti 创建双写客户端：每个地址创建一个独立客户端，用 multiClient 包装
func newMulti(version, addrs string) (Client, error) {
	addrList := strings.Split(addrs, ",")
	var clients []Client

	origAddr := os.Getenv("ES_ADDR")
	for _, addr := range addrList {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		// 临时设置 ES_ADDR 给 newSingle 用
		os.Setenv("ES_ADDR", addr)
		cli, err := newSingle(version)
		if err != nil {
			logs.Warnf("ES dual-write: failed to connect %s: %v", addr, err)
			continue
		}
		clients = append(clients, cli)
		logs.Infof("ES dual-write: connected to %s", addr)
	}
	// 恢复原始值
	os.Setenv("ES_ADDR", origAddr)

	if len(clients) == 0 {
		return nil, fmt.Errorf("ES dual-write: no instances connected")
	}

	logs.Infof("ES dual-write enabled: %d instances", len(clients))
	return newMultiClient(clients), nil
}
