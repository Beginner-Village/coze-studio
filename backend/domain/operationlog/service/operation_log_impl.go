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
	"sync/atomic"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/repository"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

// Config 控制缓冲、批量与保留期。
type Config struct {
	BufferSize    int           // channel 容量
	BatchSize     int           // 单批落库最大条数
	FlushInterval time.Duration // 定时 flush 间隔
	RetentionDays int           // 保留天数 [30,90]
	CleanupEvery  time.Duration // 清理任务周期
}

type operationLogSvc struct {
	repo    repository.OperationLogRepository
	cfg     Config
	ch      chan *entity.Event
	dropped int64
}

func NewOperationLog(repo repository.OperationLogRepository, cfg Config) OperationLog {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 4096
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}
	if cfg.RetentionDays < 30 {
		cfg.RetentionDays = 30
	}
	if cfg.RetentionDays > 90 {
		cfg.RetentionDays = 90
	}
	if cfg.CleanupEvery <= 0 {
		cfg.CleanupEvery = 24 * time.Hour
	}
	return &operationLogSvc{
		repo: repo,
		cfg:  cfg,
		ch:   make(chan *entity.Event, cfg.BufferSize),
	}
}

func (s *operationLogSvc) Collect(event *entity.Event) {
	if event == nil {
		return
	}
	select {
	case s.ch <- event:
	default:
		atomic.AddInt64(&s.dropped, 1)
	}
}

func (s *operationLogSvc) DroppedCount() int64 {
	return atomic.LoadInt64(&s.dropped)
}

func (s *operationLogSvc) List(ctx context.Context, f *entity.ListFilter) ([]*entity.OperationLog, int64, error) {
	return s.repo.List(ctx, f)
}

func (s *operationLogSvc) Start(ctx context.Context) {
	go s.runWorker(ctx)
	go s.runCleanup(ctx)
}

func (s *operationLogSvc) runWorker(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.FlushInterval)
	defer ticker.Stop()
	buf := make([]*entity.Event, 0, s.cfg.BatchSize)

	flush := func() {
		if len(buf) == 0 {
			return
		}
		if err := s.repo.BatchCreate(ctx, buf); err != nil {
			logs.CtxErrorf(ctx, "[operationlog] batch create failed: %v", err)
		}
		buf = buf[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case ev := <-s.ch:
			buf = append(buf, ev)
			if len(buf) >= s.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *operationLogSvc) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.CleanupEvery)
	defer ticker.Stop()
	s.cleanupOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupOnce(ctx)
		}
	}
}

func (s *operationLogSvc) cleanupOnce(ctx context.Context) {
	cutoff := time.Now().Add(-time.Duration(s.cfg.RetentionDays) * 24 * time.Hour).UnixMilli()
	n, err := s.repo.DeleteBefore(ctx, cutoff)
	if err != nil {
		logs.CtxErrorf(ctx, "[operationlog] cleanup failed: %v", err)
		return
	}
	if n > 0 {
		logs.CtxInfof(ctx, "[operationlog] cleanup deleted %d records before %d", n, cutoff)
	}
}
