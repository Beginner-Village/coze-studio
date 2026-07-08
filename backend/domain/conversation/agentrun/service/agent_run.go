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

package agentrun

import (
	"context"

	"github.com/cloudwego/eino/schema"

	"github.com/ynet-dev/ynet-studio/backend/domain/conversation/agentrun/entity"
)

type Run interface {
	AgentRun(ctx context.Context, req *entity.AgentRunMeta) (*schema.StreamReader[*entity.AgentRunResponse], error)
	Delete(ctx context.Context, runID []int64) error
	Create(ctx context.Context, runRecord *entity.AgentRunMeta) (*entity.RunRecordMeta, error)
	GetByID(ctx context.Context, runID int64) (*entity.RunRecordMeta, error)
	List(ctx context.Context, ListMeta *entity.ListRunRecordMeta) ([]*entity.RunRecordMeta, error)
	// ReleaseRunLock 主动释放某会话的活跃 run 锁。用于「清理会话」等场景：
	// 若上一条 run 因异常（模型报错、进程重启、流中断）未走到正常释放，锁会残留，
	// 用户重开/清理会话后仍被「已有正在进行的请求」卡住。这里显式清掉即可解锁。
	ReleaseRunLock(ctx context.Context, conversationID int64) error
}
