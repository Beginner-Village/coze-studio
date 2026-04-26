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

//go:build integration

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reaperFakeRepoTimed 在 FindStuckChunking 第一次调用时返回卡死文档；
// SetStatus 调用后内部计数器 +1，可断言被清理。
type reaperFakeRepoTimed struct {
	mu       sync.Mutex
	stuck    map[int64]int32
	cleaned  int
	calls    int
	maxCalls int
}

func (f *reaperFakeRepoTimed) FindStuckChunking(ctx context.Context, threshold time.Duration) ([]int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	ids := make([]int64, 0, len(f.stuck))
	for id, status := range f.stuck {
		if status == int32(4) { // DocumentStatusChunking
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (f *reaperFakeRepoTimed) SetStatus(ctx context.Context, id int64, status int32, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stuck[id] = status
	f.cleaned++
	return nil
}

// TestIntegration_ReaperCleansStuck 启动 reaper 持续运行，扫到卡死文档时自动清理。
func TestIntegration_ReaperCleansStuck(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	repo := &reaperFakeRepoTimed{
		stuck: map[int64]int32{
			9001: int32(4), // DocumentStatusChunking
			9002: int32(4),
			9003: int32(1), // 已生效，不会被清理
		},
	}
	r := &DocumentReaper{repo: repo, interval: 50 * time.Millisecond, threshold: 30 * time.Minute}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go r.Start(ctx)

	require.Eventually(t, func() bool {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		return repo.cleaned >= 2
	}, 1*time.Second, 20*time.Millisecond)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	assert.Equal(t, int32(documentStatusFailed), repo.stuck[9001])
	assert.Equal(t, int32(documentStatusFailed), repo.stuck[9002])
	assert.Equal(t, int32(1), repo.stuck[9003], "non-stuck doc should not be touched")
}
