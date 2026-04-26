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
	"fmt"
	"sync"
)

// ErrSystemBusy 队列已满，本次大文件解析被拒绝。
var ErrSystemBusy = errors.New("knowledge: large file worker busy")

// LargeFileWorker 用有界并发 + 有界队列限制大文件解析的资源占用。
// 大文件（图片/PDF 等）解析高 CPU/内存，需限流避免单点压垮。
type LargeFileWorker struct {
	sem    chan struct{}
	queue  chan task
	wg     sync.WaitGroup
	closed chan struct{}
}

type task struct {
	ctx    context.Context
	fn     func() error
	result chan error
}

// NewLargeFileWorker 创建一个最大并发 maxConcurrent，最大排队长度 maxQueue 的 worker。
func NewLargeFileWorker(maxConcurrent, maxQueue int) *LargeFileWorker {
	w := &LargeFileWorker{
		sem:    make(chan struct{}, maxConcurrent),
		queue:  make(chan task, maxQueue),
		closed: make(chan struct{}),
	}
	w.wg.Add(1)
	go w.run()
	return w
}

func (w *LargeFileWorker) run() {
	defer w.wg.Done()
	for {
		// 先抢占并发槽（sem），再从 queue 拉任务。
		// 这样队列里的任务在 sem 满时会被滞留，从而触发 Submit 的 ErrSystemBusy 短路。
		select {
		case <-w.closed:
			return
		case w.sem <- struct{}{}:
		}

		select {
		case <-w.closed:
			<-w.sem
			return
		case t := <-w.queue:
			LargeFileWorkerActive.Inc()
			LargeFileWorkerQueueDepth.Set(float64(len(w.queue)))
			go w.process(t)
		}
	}
}

func (w *LargeFileWorker) process(t task) {
	defer func() {
		<-w.sem
		LargeFileWorkerActive.Dec()
	}()
	defer func() {
		if r := recover(); r != nil {
			t.result <- fmt.Errorf("large file worker panic: %v", r)
		}
	}()
	t.result <- t.fn()
}

// Submit 提交一个解析任务并阻塞等待完成。如果队列已满立即返回 ErrSystemBusy。
func (w *LargeFileWorker) Submit(ctx context.Context, fn func() error) error {
	t := task{ctx: ctx, fn: fn, result: make(chan error, 1)}
	select {
	case w.queue <- t:
		LargeFileWorkerQueueDepth.Set(float64(len(w.queue)))
	default:
		return ErrSystemBusy
	}
	select {
	case err := <-t.result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close 停止 worker 调度循环。已 Submit 的进行中任务不会被打断。
func (w *LargeFileWorker) Close() {
	close(w.closed)
	w.wg.Wait()
}

// isLargeFileExt 判断扩展名是否归类为大文件类型，需要走 LargeFileWorker。
func isLargeFileExt(ext string) bool {
	switch ext {
	case ".pdf", ".doc", ".docx", ".ppt", ".pptx",
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff":
		return true
	}
	return false
}
