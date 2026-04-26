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
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLargeFileWorker_RespectsConcurrency(t *testing.T) {
	w := NewLargeFileWorker(2, 10)
	defer w.Close()

	var active int64
	var maxActive int64
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := w.Submit(context.Background(), func() error {
				cur := atomic.AddInt64(&active, 1)
				for {
					m := atomic.LoadInt64(&maxActive)
					if cur <= m || atomic.CompareAndSwapInt64(&maxActive, m, cur) {
						break
					}
				}
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt64(&active, -1)
				return nil
			})
			require.NoError(t, err)
		}()
	}

	wg.Wait()
	assert.LessOrEqual(t, atomic.LoadInt64(&maxActive), int64(2))
}

func TestLargeFileWorker_QueueFullReturnsBusy(t *testing.T) {
	w := NewLargeFileWorker(1, 1)
	defer w.Close()

	// 占满 worker（长任务）
	go func() {
		_ = w.Submit(context.Background(), func() error {
			time.Sleep(300 * time.Millisecond)
			return nil
		})
	}()
	time.Sleep(20 * time.Millisecond)

	// 占满队列
	go func() {
		_ = w.Submit(context.Background(), func() error { return nil })
	}()
	time.Sleep(20 * time.Millisecond)

	// 第三个应被立即拒绝
	err := w.Submit(context.Background(), func() error { return nil })
	assert.True(t, errors.Is(err, ErrSystemBusy), "expected ErrSystemBusy, got %v", err)
}

func TestLargeFileWorker_RecoverPanic(t *testing.T) {
	w := NewLargeFileWorker(1, 5)
	defer w.Close()

	err := w.Submit(context.Background(), func() error {
		panic("boom")
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
}
