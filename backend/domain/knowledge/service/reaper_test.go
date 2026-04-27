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
	"testing"
	"time"
)

type fakeReaperRepo struct {
	mu          sync.Mutex
	stuckIDs    []int64
	findErr     error
	setStatuses []reaperSetCall
	setErr      error
}

type reaperSetCall struct {
	id     int64
	status int32
	reason string
}

func (f *fakeReaperRepo) FindStuckChunking(ctx context.Context, threshold time.Duration) ([]int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.findErr != nil {
		return nil, f.findErr
	}
	out := make([]int64, len(f.stuckIDs))
	copy(out, f.stuckIDs)
	return out, nil
}

func (f *fakeReaperRepo) SetStatus(ctx context.Context, id int64, status int32, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.setErr != nil {
		return f.setErr
	}
	f.setStatuses = append(f.setStatuses, reaperSetCall{id: id, status: status, reason: reason})
	return nil
}

func TestDocumentReaper_SweepNoStuck(t *testing.T) {
	repo := &fakeReaperRepo{}
	r := &DocumentReaper{repo: repo, interval: time.Hour, threshold: 30 * time.Minute}
	r.sweep(context.Background())
	if len(repo.setStatuses) != 0 {
		t.Errorf("expected 0 SetStatus calls, got %d", len(repo.setStatuses))
	}
}

func TestDocumentReaper_SweepFindError(t *testing.T) {
	repo := &fakeReaperRepo{findErr: errors.New("db down")}
	r := &DocumentReaper{repo: repo, interval: time.Hour, threshold: 30 * time.Minute}
	r.sweep(context.Background())
	if len(repo.setStatuses) != 0 {
		t.Errorf("expected no SetStatus on find error, got %d", len(repo.setStatuses))
	}
}

func TestDocumentReaper_SweepCleansStuck(t *testing.T) {
	repo := &fakeReaperRepo{stuckIDs: []int64{101, 102}}
	r := &DocumentReaper{repo: repo, interval: time.Hour, threshold: 30 * time.Minute}
	r.sweep(context.Background())
	if len(repo.setStatuses) != 2 {
		t.Fatalf("expected 2 SetStatus calls, got %d", len(repo.setStatuses))
	}
	for _, c := range repo.setStatuses {
		if c.status != int32(documentStatusFailed) {
			t.Errorf("unexpected status %d", c.status)
		}
		if c.reason != stuckCleanupReason {
			t.Errorf("unexpected reason %q", c.reason)
		}
	}
}

func TestDocumentReaper_SweepSetStatusErrorContinues(t *testing.T) {
	repo := &fakeReaperRepo{stuckIDs: []int64{1, 2, 3}, setErr: errors.New("db error")}
	r := &DocumentReaper{repo: repo, interval: time.Hour, threshold: 30 * time.Minute}
	r.sweep(context.Background())
}
