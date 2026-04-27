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
	"time"

	"github.com/ynet-dev/ynet-studio/backend/domain/knowledge/entity"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
)

const (
	defaultReaperInterval  = 5 * time.Minute
	defaultReaperThreshold = 30 * time.Minute
	stuckCleanupReason     = "stuck in chunking - reaper cleanup"
)

var documentStatusFailed = int32(entity.DocumentStatusFailed)

// ReaperDocumentRepo 反映 reaper 用到的 repo 接口子集。
type ReaperDocumentRepo interface {
	FindStuckChunking(ctx context.Context, threshold time.Duration) ([]int64, error)
	SetStatus(ctx context.Context, id int64, status int32, reason string) error
}

type DocumentReaper struct {
	repo      ReaperDocumentRepo
	interval  time.Duration
	threshold time.Duration
}

func NewDocumentReaper(repo ReaperDocumentRepo) *DocumentReaper {
	return &DocumentReaper{
		repo:      repo,
		interval:  defaultReaperInterval,
		threshold: defaultReaperThreshold,
	}
}

func (r *DocumentReaper) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	logs.CtxInfof(ctx, "[reaper] started, interval=%v threshold=%v", r.interval, r.threshold)

	for {
		select {
		case <-ctx.Done():
			logs.CtxInfof(ctx, "[reaper] shutting down")
			return
		case <-ticker.C:
			r.sweep(ctx)
		}
	}
}

func (r *DocumentReaper) sweep(ctx context.Context) {
	ids, err := r.repo.FindStuckChunking(ctx, r.threshold)
	if err != nil {
		logs.CtxErrorf(ctx, "[reaper] FindStuckChunking failed: %v", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	logs.CtxInfof(ctx, "[reaper] cleaning %d stuck documents", len(ids))
	for _, id := range ids {
		if err := r.repo.SetStatus(ctx, id, documentStatusFailed, stuckCleanupReason); err != nil {
			logs.CtxErrorf(ctx, "[reaper] SetStatus(%d) failed: %v", id, err)
			continue
		}
		DocumentReaperCleanedTotal.Inc()
	}
}
