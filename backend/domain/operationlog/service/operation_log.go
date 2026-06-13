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

package service

import (
	"context"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
)

// OperationLog 是审计日志领域服务。
type OperationLog interface {
	// Collect 非阻塞投递一条采集事件；缓冲满时丢弃并计数，绝不阻塞调用方。
	Collect(event *entity.Event)
	// List 分页查询。
	List(ctx context.Context, f *entity.ListFilter) ([]*entity.OperationLog, int64, error)
	// Start 启动后台落库 worker 与清理循环。
	Start(ctx context.Context)
	// DroppedCount 返回因缓冲满被丢弃的事件数（用于可观测/测试）。
	DroppedCount() int64
}
