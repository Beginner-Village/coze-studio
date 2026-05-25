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
	"context"

	"github.com/ynet-dev/ynet-studio/backend/infra/contract/es"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// multiClient 实现 ES 双写：写操作同时发到所有实例，读操作只走主实例
type multiClient struct {
	primary   Client   // 主实例（读写）
	secondaries []Client // 副实例（仅写）
}

func newMultiClient(clients []Client) Client {
	if len(clients) == 1 {
		return clients[0]
	}
	return &multiClient{
		primary:     clients[0],
		secondaries: clients[1:],
	}
}

func (m *multiClient) Create(ctx context.Context, index, id string, document any) error {
	// 主实例写入
	if err := m.primary.Create(ctx, index, id, document); err != nil {
		return err
	}
	// 异步写入副实例
	for _, cli := range m.secondaries {
		go func(c Client) {
			if err := c.Create(ctx, index, id, document); err != nil {
				logs.CtxWarnf(ctx, "ES dual-write Create failed: %v", err)
			}
		}(cli)
	}
	return nil
}

func (m *multiClient) Update(ctx context.Context, index, id string, document any) error {
	if err := m.primary.Update(ctx, index, id, document); err != nil {
		return err
	}
	for _, cli := range m.secondaries {
		go func(c Client) {
			if err := c.Update(ctx, index, id, document); err != nil {
				logs.CtxWarnf(ctx, "ES dual-write Update failed: %v", err)
			}
		}(cli)
	}
	return nil
}

func (m *multiClient) Delete(ctx context.Context, index, id string) error {
	if err := m.primary.Delete(ctx, index, id); err != nil {
		return err
	}
	for _, cli := range m.secondaries {
		go func(c Client) {
			if err := c.Delete(ctx, index, id); err != nil {
				logs.CtxWarnf(ctx, "ES dual-write Delete failed: %v", err)
			}
		}(cli)
	}
	return nil
}

func (m *multiClient) DeleteByQuery(ctx context.Context, index string, query map[string]any) (int64, error) {
	deleted, err := m.primary.DeleteByQuery(ctx, index, query)
	if err != nil {
		return 0, err
	}
	for _, cli := range m.secondaries {
		go func(c Client) {
			if _, err := c.DeleteByQuery(ctx, index, query); err != nil {
				logs.CtxWarnf(ctx, "ES dual-write DeleteByQuery failed: %v", err)
			}
		}(cli)
	}
	return deleted, nil
}

// Search 只走主实例
func (m *multiClient) Search(ctx context.Context, index string, req *es.Request) (*es.Response, error) {
	return m.primary.Search(ctx, index, req)
}

// Exists 只走主实例
func (m *multiClient) Exists(ctx context.Context, index string) (bool, error) {
	return m.primary.Exists(ctx, index)
}

func (m *multiClient) CreateIndex(ctx context.Context, index string, properties map[string]any) error {
	if err := m.primary.CreateIndex(ctx, index, properties); err != nil {
		return err
	}
	for _, cli := range m.secondaries {
		go func(c Client) {
			if err := c.CreateIndex(ctx, index, properties); err != nil {
				logs.CtxWarnf(ctx, "ES dual-write CreateIndex failed: %v", err)
			}
		}(cli)
	}
	return nil
}

func (m *multiClient) DeleteIndex(ctx context.Context, index string) error {
	if err := m.primary.DeleteIndex(ctx, index); err != nil {
		return err
	}
	for _, cli := range m.secondaries {
		go func(c Client) {
			if err := c.DeleteIndex(ctx, index); err != nil {
				logs.CtxWarnf(ctx, "ES dual-write DeleteIndex failed: %v", err)
			}
		}(cli)
	}
	return nil
}

func (m *multiClient) Types() Types {
	return m.primary.Types()
}

// NewBulkIndexer 返回支持双写的 BulkIndexer
func (m *multiClient) NewBulkIndexer(index string) (BulkIndexer, error) {
	primaryBI, err := m.primary.NewBulkIndexer(index)
	if err != nil {
		return nil, err
	}

	var secondaryBIs []BulkIndexer
	for _, cli := range m.secondaries {
		bi, err := cli.NewBulkIndexer(index)
		if err != nil {
			logs.Warnf("ES dual-write NewBulkIndexer failed: %v", err)
			continue
		}
		secondaryBIs = append(secondaryBIs, bi)
	}

	if len(secondaryBIs) == 0 {
		return primaryBI, nil
	}
	return &multiBulkIndexer{primary: primaryBI, secondaries: secondaryBIs}, nil
}

type multiBulkIndexer struct {
	primary     BulkIndexer
	secondaries []BulkIndexer
}

func (mb *multiBulkIndexer) Add(ctx context.Context, item BulkIndexerItem) error {
	if err := mb.primary.Add(ctx, item); err != nil {
		return err
	}
	for _, bi := range mb.secondaries {
		if err := bi.Add(ctx, item); err != nil {
			logs.CtxWarnf(ctx, "ES dual-write BulkIndexer.Add failed: %v", err)
		}
	}
	return nil
}

func (mb *multiBulkIndexer) Close(ctx context.Context) error {
	err := mb.primary.Close(ctx)
	for _, bi := range mb.secondaries {
		if e := bi.Close(ctx); e != nil {
			logs.CtxWarnf(ctx, "ES dual-write BulkIndexer.Close failed: %v", e)
		}
	}
	return err
}
