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

	"go.uber.org/mock/gomock"

	knowledgeModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/knowledge"
	"github.com/ynet-dev/ynet-studio/backend/domain/knowledge/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/knowledge/internal/dal/model"
	mockDao "github.com/ynet-dev/ynet-studio/backend/domain/knowledge/internal/mock/dal/dao"
	"github.com/ynet-dev/ynet-studio/backend/domain/knowledge/repository"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/document/searchstore"
	mockEventbus "github.com/ynet-dev/ynet-studio/backend/internal/mock/infra/contract/eventbus"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test fakes — kept in-package so we can hand them to the searchstore.Manager
// slice the production code holds (k.searchStoreManagers). We deliberately
// avoid the heavier MockKnowledgeSVC infrastructure here because
// ResyncSpaceSlices only touches knowledgeRepo + sliceRepo + producer +
// managers, so a tight gomock-style wiring is enough.
// ─────────────────────────────────────────────────────────────────────────────

// recordingSearchStore captures every DeleteIndex call so the test can
// assert which collection names were dropped per backend.
type recordingSearchStore struct {
	mu        sync.Mutex
	dropped   []string
	deleteErr error
}

func (r *recordingSearchStore) Store(ctx context.Context, _ []*schema.Document, _ ...indexer.Option) ([]string, error) {
	return nil, nil
}

func (r *recordingSearchStore) Retrieve(ctx context.Context, _ string, _ ...retriever.Option) ([]*schema.Document, error) {
	return nil, nil
}

func (r *recordingSearchStore) Delete(_ context.Context, _ []string) error { return nil }

func (r *recordingSearchStore) DeleteByQuery(_ context.Context, _ string, _ map[string]any) (int64, error) {
	return 0, nil
}

func (r *recordingSearchStore) DeleteIndex(_ context.Context, index string) error {
	r.mu.Lock()
	r.dropped = append(r.dropped, index)
	r.mu.Unlock()
	return r.deleteErr
}

// fakeManager satisfies searchstore.Manager and hands back a single shared
// recordingSearchStore for any collection name. GetType() lets the test
// distinguish vector vs text store traffic.
type fakeManager struct {
	storeType searchstore.SearchStoreType
	ss        *recordingSearchStore
	getErr    error
}

func (f *fakeManager) Create(context.Context, *searchstore.CreateRequest) error { return nil }
func (f *fakeManager) Drop(context.Context, *searchstore.DropRequest) error     { return nil }
func (f *fakeManager) GetType() searchstore.SearchStoreType                     { return f.storeType }
func (f *fakeManager) GetSearchStore(context.Context, string) (searchstore.SearchStore, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.ss, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

// TestResyncSpaceSlices_HappyPath verifies the full pipeline for 2 KBs ×
// multiple slices each:
//   - knowledgeRepo.ListBySpaceID called once with the input spaceID
//   - sliceRepo.FindSliceByCondition called per KB, filtered by KnowledgeID
//   - BOTH managers (vector + text) get DeleteIndex(openynet_<kb_id>) per KB
//   - sliceRepo.BatchSetStatus(ids, SliceStatusInit, "") per KB
//   - producer.Send called once per slice, sharded by DocumentID
//   - returned count == total slices across all KBs
func TestResyncSpaceSlices_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const spaceID int64 = 999

	kbRepo := mockDao.NewMockKnowledgeRepo(ctrl)
	sliceRepo := mockDao.NewMockKnowledgeDocumentSliceRepo(ctrl)
	producer := mockEventbus.NewMockProducer(ctrl)

	kbRepo.EXPECT().ListBySpaceID(gomock.Any(), spaceID, 0).Return([]*model.Knowledge{
		{ID: 100, SpaceID: spaceID},
		{ID: 200, SpaceID: spaceID},
	}, nil)

	kb100Slices := []*model.KnowledgeDocumentSlice{
		{ID: 1001, KnowledgeID: 100, DocumentID: 11},
		{ID: 1002, KnowledgeID: 100, DocumentID: 11},
		{ID: 1003, KnowledgeID: 100, DocumentID: 12},
	}
	kb200Slices := []*model.KnowledgeDocumentSlice{
		{ID: 2001, KnowledgeID: 200, DocumentID: 21},
		{ID: 2002, KnowledgeID: 200, DocumentID: 22},
	}

	sliceRepo.EXPECT().
		FindSliceByCondition(gomock.Any(), gomock.AssignableToTypeOf(&entity.WhereSliceOpt{})).
		DoAndReturn(func(_ context.Context, opt *entity.WhereSliceOpt) ([]*model.KnowledgeDocumentSlice, int64, error) {
			switch opt.KnowledgeID {
			case 100:
				return kb100Slices, int64(len(kb100Slices)), nil
			case 200:
				return kb200Slices, int64(len(kb200Slices)), nil
			default:
				t.Fatalf("unexpected KnowledgeID filter: %d", opt.KnowledgeID)
				return nil, 0, nil
			}
		}).Times(2)

	// BatchSetStatus per KB, with status=Init and the exact ID lists above.
	sliceRepo.EXPECT().
		BatchSetStatus(gomock.Any(), []int64{1001, 1002, 1003}, int32(knowledgeModel.SliceStatusInit), "").
		Return(nil)
	sliceRepo.EXPECT().
		BatchSetStatus(gomock.Any(), []int64{2001, 2002}, int32(knowledgeModel.SliceStatusInit), "").
		Return(nil)

	// Capture every published message body so we can assert sharding keys
	// + count.
	var publishedShardingKeys []string
	var publishedMu sync.Mutex
	totalSlices := len(kb100Slices) + len(kb200Slices)
	producer.EXPECT().
		Send(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ []byte, opts ...any) error {
			publishedMu.Lock()
			defer publishedMu.Unlock()
			// We can't introspect the opaque SendOpt cheaply; just count.
			publishedShardingKeys = append(publishedShardingKeys, "ok")
			return nil
		}).Times(totalSlices)

	vectorStore := &recordingSearchStore{}
	textStore := &recordingSearchStore{}
	vectorMgr := &fakeManager{storeType: searchstore.TypeVectorStore, ss: vectorStore}
	textMgr := &fakeManager{storeType: searchstore.TypeTextStore, ss: textStore}

	svc := &knowledgeSVC{
		knowledgeRepo: kbRepoAsRepo(kbRepo),
		sliceRepo:     sliceRepoAsRepo(sliceRepo),
		producer:      producer,
		// No managerFactory configured → falls back to legacy
		// searchStoreManagers (the path bank-internal POC uses).
		searchStoreManagers: []searchstore.Manager{vectorMgr, textMgr},
	}

	got, err := svc.ResyncSpaceSlices(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpaceSlices returned err: %v", err)
	}
	if got != totalSlices {
		t.Fatalf("returned count = %d, want %d", got, totalSlices)
	}

	// Both managers should have been asked to drop both KB indices.
	wantDropped := []string{"openynet_100", "openynet_200"}
	assertSameSet(t, "vector dropped", vectorStore.dropped, wantDropped)
	assertSameSet(t, "text dropped", textStore.dropped, wantDropped)

	if len(publishedShardingKeys) != totalSlices {
		t.Fatalf("publishedShardingKeys count = %d, want %d", len(publishedShardingKeys), totalSlices)
	}
}

// TestResyncSpaceSlices_NoKbs returns 0, nil without touching slice repo or
// producer.
func TestResyncSpaceSlices_NoKbs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kbRepo := mockDao.NewMockKnowledgeRepo(ctrl)
	sliceRepo := mockDao.NewMockKnowledgeDocumentSliceRepo(ctrl)
	producer := mockEventbus.NewMockProducer(ctrl)

	kbRepo.EXPECT().ListBySpaceID(gomock.Any(), int64(7), 0).Return([]*model.Knowledge{}, nil)
	// sliceRepo + producer get no EXPECT calls → gomock will fail the test
	// if anything fires unexpectedly.

	svc := &knowledgeSVC{
		knowledgeRepo:       kbRepoAsRepo(kbRepo),
		sliceRepo:           sliceRepoAsRepo(sliceRepo),
		producer:            producer,
		searchStoreManagers: nil,
	}

	got, err := svc.ResyncSpaceSlices(context.Background(), 7)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != 0 {
		t.Fatalf("count = %d, want 0", got)
	}
}

// TestResyncSpaceSlices_ListKbsErrorPropagates ensures the only fatal error
// path returns immediately.
func TestResyncSpaceSlices_ListKbsErrorPropagates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kbRepo := mockDao.NewMockKnowledgeRepo(ctrl)
	sliceRepo := mockDao.NewMockKnowledgeDocumentSliceRepo(ctrl)
	producer := mockEventbus.NewMockProducer(ctrl)

	want := errors.New("boom-list")
	kbRepo.EXPECT().ListBySpaceID(gomock.Any(), int64(42), 0).Return(nil, want)

	svc := &knowledgeSVC{
		knowledgeRepo:       kbRepoAsRepo(kbRepo),
		sliceRepo:           sliceRepoAsRepo(sliceRepo),
		producer:            producer,
		searchStoreManagers: nil,
	}

	_, err := svc.ResyncSpaceSlices(context.Background(), 42)
	if err == nil || !errors.Is(err, want) {
		t.Fatalf("err = %v, want wraps %v", err, want)
	}
}

// TestResyncSpaceSlices_PerKbFailuresAreSkipped covers the resilience
// contract: one KB blowing up at the slice list / batch-status / publish
// stage MUST NOT abort the other KB. The healthy KB still emits N publishes.
func TestResyncSpaceSlices_PerKbFailuresAreSkipped(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const spaceID int64 = 13

	kbRepo := mockDao.NewMockKnowledgeRepo(ctrl)
	sliceRepo := mockDao.NewMockKnowledgeDocumentSliceRepo(ctrl)
	producer := mockEventbus.NewMockProducer(ctrl)

	kbRepo.EXPECT().ListBySpaceID(gomock.Any(), spaceID, 0).Return([]*model.Knowledge{
		{ID: 100, SpaceID: spaceID}, // healthy
		{ID: 200, SpaceID: spaceID}, // FindSliceByCondition errors
	}, nil)

	healthySlices := []*model.KnowledgeDocumentSlice{
		{ID: 1, KnowledgeID: 100, DocumentID: 10},
		{ID: 2, KnowledgeID: 100, DocumentID: 11},
	}
	sliceRepo.EXPECT().
		FindSliceByCondition(gomock.Any(), gomock.AssignableToTypeOf(&entity.WhereSliceOpt{})).
		DoAndReturn(func(_ context.Context, opt *entity.WhereSliceOpt) ([]*model.KnowledgeDocumentSlice, int64, error) {
			if opt.KnowledgeID == 100 {
				return healthySlices, 2, nil
			}
			return nil, 0, errors.New("slice list boom for kb=200")
		}).Times(2)

	sliceRepo.EXPECT().
		BatchSetStatus(gomock.Any(), []int64{1, 2}, int32(knowledgeModel.SliceStatusInit), "").
		Return(nil)

	producer.EXPECT().
		Send(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		Times(len(healthySlices))

	svc := &knowledgeSVC{
		knowledgeRepo:       kbRepoAsRepo(kbRepo),
		sliceRepo:           sliceRepoAsRepo(sliceRepo),
		producer:            producer,
		searchStoreManagers: nil,
	}

	got, err := svc.ResyncSpaceSlices(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpaceSlices returned err: %v (per-kb failures must NOT fail the whole call)", err)
	}
	if got != len(healthySlices) {
		t.Fatalf("count = %d, want %d (only healthy KB should publish)", got, len(healthySlices))
	}
}

// TestResyncSpaceSlices_MQPublishMiddleFails verifies the per-slice resilience
// contract: if producer.Send returns an error for one slice in the middle of
// the loop, the function MUST keep going for the rest and count only the
// successful publishes (4 out of 5).
func TestResyncSpaceSlices_MQPublishMiddleFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const spaceID int64 = 5050

	kbRepo := mockDao.NewMockKnowledgeRepo(ctrl)
	sliceRepo := mockDao.NewMockKnowledgeDocumentSliceRepo(ctrl)
	producer := mockEventbus.NewMockProducer(ctrl)

	kbRepo.EXPECT().ListBySpaceID(gomock.Any(), spaceID, 0).Return([]*model.Knowledge{
		{ID: 500, SpaceID: spaceID},
	}, nil)

	slices := []*model.KnowledgeDocumentSlice{
		{ID: 1, KnowledgeID: 500, DocumentID: 10},
		{ID: 2, KnowledgeID: 500, DocumentID: 10},
		{ID: 3, KnowledgeID: 500, DocumentID: 11}, // this one will fail to publish
		{ID: 4, KnowledgeID: 500, DocumentID: 12},
		{ID: 5, KnowledgeID: 500, DocumentID: 13},
	}
	sliceRepo.EXPECT().
		FindSliceByCondition(gomock.Any(), gomock.AssignableToTypeOf(&entity.WhereSliceOpt{})).
		Return(slices, int64(len(slices)), nil)

	sliceRepo.EXPECT().
		BatchSetStatus(gomock.Any(), []int64{1, 2, 3, 4, 5}, int32(knowledgeModel.SliceStatusInit), "").
		Return(nil)

	// 5 producer.Send calls expected; the 3rd returns an error, others succeed.
	// We use a counter to flip the response on the 3rd invocation.
	var (
		sendMu    sync.Mutex
		sendCalls int
	)
	producer.EXPECT().
		Send(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ []byte, _ ...any) error {
			sendMu.Lock()
			defer sendMu.Unlock()
			sendCalls++
			if sendCalls == 3 {
				return errors.New("mq broker timeout")
			}
			return nil
		}).
		Times(len(slices))

	svc := &knowledgeSVC{
		knowledgeRepo:       kbRepoAsRepo(kbRepo),
		sliceRepo:           sliceRepoAsRepo(sliceRepo),
		producer:            producer,
		searchStoreManagers: nil,
	}

	got, err := svc.ResyncSpaceSlices(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpaceSlices returned err: %v (single-publish failures must NOT abort)", err)
	}
	if got != len(slices)-1 {
		t.Fatalf("count = %d, want %d (skip 1 failed publish)", got, len(slices)-1)
	}
	if sendCalls != len(slices) {
		t.Errorf("send was invoked %d times, want %d (loop must NOT short-circuit)", sendCalls, len(slices))
	}
}

// TestResyncSpaceSlices_DeleteIndexFails_StillProceedsWithEvents covers the
// resilience contract for step (1): if searchStore.DeleteIndex returns an
// error, the function MUST log a warn and continue with steps (2)–(4) so the
// per-doc upsert flow can self-heal ES.
func TestResyncSpaceSlices_DeleteIndexFails_StillProceedsWithEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const spaceID int64 = 6060

	kbRepo := mockDao.NewMockKnowledgeRepo(ctrl)
	sliceRepo := mockDao.NewMockKnowledgeDocumentSliceRepo(ctrl)
	producer := mockEventbus.NewMockProducer(ctrl)

	kbRepo.EXPECT().ListBySpaceID(gomock.Any(), spaceID, 0).Return([]*model.Knowledge{
		{ID: 600, SpaceID: spaceID},
	}, nil)

	slices := []*model.KnowledgeDocumentSlice{
		{ID: 1, KnowledgeID: 600, DocumentID: 10},
		{ID: 2, KnowledgeID: 600, DocumentID: 11},
	}
	sliceRepo.EXPECT().
		FindSliceByCondition(gomock.Any(), gomock.AssignableToTypeOf(&entity.WhereSliceOpt{})).
		Return(slices, int64(len(slices)), nil)

	sliceRepo.EXPECT().
		BatchSetStatus(gomock.Any(), []int64{1, 2}, int32(knowledgeModel.SliceStatusInit), "").
		Return(nil)

	producer.EXPECT().
		Send(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		Times(len(slices))

	// Both ES + vector managers blow up on DeleteIndex — the slice publish
	// loop must still run.
	failingTextStore := &recordingSearchStore{deleteErr: errors.New("es index drop transient err")}
	failingVectorStore := &recordingSearchStore{deleteErr: errors.New("vector store drop boom")}
	textMgr := &fakeManager{storeType: searchstore.TypeTextStore, ss: failingTextStore}
	vectorMgr := &fakeManager{storeType: searchstore.TypeVectorStore, ss: failingVectorStore}

	svc := &knowledgeSVC{
		knowledgeRepo:       kbRepoAsRepo(kbRepo),
		sliceRepo:           sliceRepoAsRepo(sliceRepo),
		producer:            producer,
		searchStoreManagers: []searchstore.Manager{textMgr, vectorMgr},
	}

	got, err := svc.ResyncSpaceSlices(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpaceSlices returned err: %v (DeleteIndex failure must not abort)", err)
	}
	if got != len(slices) {
		t.Fatalf("count = %d, want %d — events must still publish even when DeleteIndex fails", got, len(slices))
	}
	// Both managers must have been called (and recorded the attempt).
	if len(failingTextStore.dropped) != 1 || failingTextStore.dropped[0] != "openynet_600" {
		t.Errorf("text store dropped = %v, want [openynet_600]", failingTextStore.dropped)
	}
	if len(failingVectorStore.dropped) != 1 || failingVectorStore.dropped[0] != "openynet_600" {
		t.Errorf("vector store dropped = %v, want [openynet_600]", failingVectorStore.dropped)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// helpers
// ─────────────────────────────────────────────────────────────────────────────

// kbRepoAsRepo & sliceRepoAsRepo coerce the mock types to the repo
// interfaces. The mocks already implement the full interface — these are
// just typed wrappers to satisfy knowledgeSVC's named field types.
func kbRepoAsRepo(m *mockDao.MockKnowledgeRepo) repository.KnowledgeRepo       { return m }
func sliceRepoAsRepo(m *mockDao.MockKnowledgeDocumentSliceRepo) repository.KnowledgeDocumentSliceRepo {
	return m
}

// assertSameSet checks that two []string slices contain the same elements
// (multiset equality — order-insensitive, counts matter).
func assertSameSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: count = %d (%v), want %d (%v)", label, len(got), got, len(want), want)
		return
	}
	gotSet := map[string]int{}
	for _, s := range got {
		gotSet[s]++
	}
	for _, s := range want {
		if gotSet[s] == 0 {
			t.Errorf("%s: missing %q (got=%v)", label, s, got)
			return
		}
		gotSet[s]--
	}
	for s, n := range gotSet {
		if n != 0 {
			t.Errorf("%s: extra %q x %d (got=%v)", label, s, n, got)
		}
	}
}
