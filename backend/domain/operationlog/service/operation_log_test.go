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
	"sync"
	"testing"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
)

type fakeRepo struct {
	mu       sync.Mutex
	created  []*entity.OperationLog
	deleted  int64
	deleteTS int64
}

func (f *fakeRepo) BatchCreate(ctx context.Context, logs []*entity.OperationLog) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, logs...)
	return nil
}
func (f *fakeRepo) List(ctx context.Context, _ *entity.ListFilter) ([]*entity.OperationLog, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.created, int64(len(f.created)), nil
}
func (f *fakeRepo) DeleteBefore(ctx context.Context, ts int64) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteTS = ts
	return f.deleted, nil
}
func (f *fakeRepo) createdLen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.created)
}
func (f *fakeRepo) lastDeleteTS() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.deleteTS
}

func TestRetentionDefaultsTo180Days(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewOperationLog(repo, Config{}).(*operationLogSvc)
	if got := svc.cfg.RetentionDays; got != 180 {
		t.Fatalf("want default retention 180 days, got %d", got)
	}
}

func TestRetentionClampsBelow180Days(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewOperationLog(repo, Config{RetentionDays: 90}).(*operationLogSvc)
	if got := svc.cfg.RetentionDays; got != 180 {
		t.Fatalf("want retention below 180 to clamp to 180 days, got %d", got)
	}
}

func TestCleanupUses180DayCutoff(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewOperationLog(repo, Config{RetentionDays: 180}).(*operationLogSvc)

	before := time.Now().Add(-180 * 24 * time.Hour).UnixMilli()
	svc.cleanupOnce(context.Background())
	after := time.Now().Add(-180 * 24 * time.Hour).UnixMilli()

	got := repo.lastDeleteTS()
	if got < before || got > after {
		t.Fatalf("want cleanup cutoff between %d and %d, got %d", before, after, got)
	}
}

func TestCollectAndFlush(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewOperationLog(repo, Config{BufferSize: 10, BatchSize: 2, FlushInterval: 20 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)

	for i := 0; i < 3; i++ {
		svc.Collect(&entity.Event{SpaceID: 1, Action: "create"})
	}
	time.Sleep(100 * time.Millisecond)
	if repo.createdLen() != 3 {
		t.Fatalf("want 3 created, got %d", repo.createdLen())
	}
}

func TestCollectDropsWhenFull(t *testing.T) {
	repo := &fakeRepo{}
	// 不 Start ⇒ channel 不被消费，容量 2，投递 5 条 ⇒ 丢 3 条
	svc := NewOperationLog(repo, Config{BufferSize: 2, BatchSize: 2})
	for i := 0; i < 5; i++ {
		svc.Collect(&entity.Event{SpaceID: 1})
	}
	if got := svc.DroppedCount(); got != 3 {
		t.Fatalf("want 3 dropped, got %d", got)
	}
}
