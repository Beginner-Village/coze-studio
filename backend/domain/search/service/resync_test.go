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
	"reflect"
	"testing"

	agententity "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	appentity "github.com/ynet-dev/ynet-studio/backend/domain/app/entity"
	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/es"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
)

// fakeAgentLister is a minimal fake for AgentLister.
type fakeAgentLister struct {
	gotSpaceID int64
	gotLimit   int
	ret        []*agententity.SingleAgent
	retErr     error
}

func (f *fakeAgentLister) ListBySpaceID(_ context.Context, spaceID int64, limit int) ([]*agententity.SingleAgent, error) {
	f.gotSpaceID = spaceID
	f.gotLimit = limit
	return f.ret, f.retErr
}

// fakeAppLister is a minimal fake for AppLister.
type fakeAppLister struct {
	gotSpaceID int64
	gotLimit   int
	ret        []*appentity.APP
	retErr     error
}

func (f *fakeAppLister) ListBySpaceID(_ context.Context, spaceID int64, limit int) ([]*appentity.APP, error) {
	f.gotSpaceID = spaceID
	f.gotLimit = limit
	return f.ret, f.retErr
}

// fakeKbLister is a minimal fake for KbLister.
type fakeKbLister struct {
	gotSpaceID int64
	gotLimit   int
	ret        []*KbInfo
	retErr     error
}

func (f *fakeKbLister) ListBySpaceID(_ context.Context, spaceID int64, limit int) ([]*KbInfo, error) {
	f.gotSpaceID = spaceID
	f.gotLimit = limit
	return f.ret, f.retErr
}

// recordedDelete captures one DeleteByQuery call.
type recordedDelete struct {
	index string
	query map[string]any
}

// recordedCreate captures one Create call.
type recordedCreate struct {
	index string
	id    string
}

// fakeResyncESClient records DeleteByQuery + Create calls. Other es.Client
// methods panic to surface accidental wiring changes.
type fakeResyncESClient struct {
	deletes      []recordedDelete
	creates      []recordedCreate
	deleteReturn map[string]int64 // index -> deletedCount
	deleteErr    map[string]error
	createErr    map[string]error // id -> err
}

func (f *fakeResyncESClient) DeleteByQuery(_ context.Context, index string, query map[string]any) (int64, error) {
	f.deletes = append(f.deletes, recordedDelete{index: index, query: query})
	if f.deleteErr != nil {
		if err, ok := f.deleteErr[index]; ok {
			return 0, err
		}
	}
	if f.deleteReturn != nil {
		if n, ok := f.deleteReturn[index]; ok {
			return n, nil
		}
	}
	return 0, nil
}

func (f *fakeResyncESClient) Create(_ context.Context, index, id string, _ any) error {
	f.creates = append(f.creates, recordedCreate{index: index, id: id})
	if f.createErr != nil {
		if err, ok := f.createErr[id]; ok {
			return err
		}
	}
	return nil
}

func (f *fakeResyncESClient) Update(context.Context, string, string, any) error { panic("not used") }
func (f *fakeResyncESClient) Delete(context.Context, string, string) error      { panic("not used") }
func (f *fakeResyncESClient) Search(context.Context, string, *es.Request) (*es.Response, error) {
	panic("not used")
}
func (f *fakeResyncESClient) Exists(context.Context, string) (bool, error) { panic("not used") }
func (f *fakeResyncESClient) CreateIndex(context.Context, string, map[string]any) error {
	panic("not used")
}
func (f *fakeResyncESClient) DeleteIndex(context.Context, string) error     { panic("not used") }
func (f *fakeResyncESClient) Types() es.Types                               { panic("not used") }
func (f *fakeResyncESClient) NewBulkIndexer(string) (es.BulkIndexer, error) { panic("not used") }

func newAgent(agentID, spaceID, ownerID int64, name string) *agententity.SingleAgent {
	return &agententity.SingleAgent{
		SingleAgent: &crossagent.SingleAgent{
			AgentID:   agentID,
			SpaceID:   spaceID,
			CreatorID: ownerID,
			Name:      name,
			CreatedAt: 1000,
			UpdatedAt: 2000,
		},
	}
}

func TestSearchSvc_ResyncSpace_RewritesAllIndices(t *testing.T) {
	spaceID := int64(100)

	kbs := []*KbInfo{
		{ID: 5001, SpaceID: spaceID, Name: "kb-a", OwnerID: 9, CreatedAtMs: 100, UpdatedAtMs: 200},
		{ID: 5002, SpaceID: spaceID, Name: "kb-b", OwnerID: 9, CreatedAtMs: 110, UpdatedAtMs: 210},
	}
	agents := []*agententity.SingleAgent{newAgent(11, spaceID, 7, "agent-1")}
	apps := []*appentity.APP{
		{ID: 10, SpaceID: spaceID, OwnerID: 8, Name: ptr.Of("app-1"), CreatedAtMS: 300, UpdatedAtMS: 400},
	}

	fakeES := &fakeResyncESClient{
		deleteReturn: map[string]int64{
			projectIndexName:  2,
			resourceIndexName: 3,
			kbEntriesIndex:    2,
		},
	}
	agentRepo := &fakeAgentLister{ret: agents}
	appRepo := &fakeAppLister{ret: apps}
	kbRepo := &fakeKbLister{ret: kbs}

	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: agentRepo,
		appRepo:   appRepo,
		kbRepo:    kbRepo,
	}

	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace returned err: %v", err)
	}
	if counts == nil {
		t.Fatal("counts is nil")
	}
	if counts.ProjectDraft != 1 {
		t.Errorf("ProjectDraft = %d, want 1", counts.ProjectDraft)
	}
	if counts.CozeResource != 1 {
		t.Errorf("CozeResource = %d, want 1", counts.CozeResource)
	}
	if counts.KbEntries != 2 {
		t.Errorf("KbEntries = %d, want 2", counts.KbEntries)
	}

	// Verify lister calls
	if agentRepo.gotSpaceID != spaceID || agentRepo.gotLimit != 0 {
		t.Errorf("agent lister got (%d, %d), want (%d, 0)", agentRepo.gotSpaceID, agentRepo.gotLimit, spaceID)
	}
	if appRepo.gotSpaceID != spaceID || appRepo.gotLimit != 0 {
		t.Errorf("app lister got (%d, %d), want (%d, 0)", appRepo.gotSpaceID, appRepo.gotLimit, spaceID)
	}
	if kbRepo.gotSpaceID != spaceID || kbRepo.gotLimit != 0 {
		t.Errorf("kb lister got (%d, %d), want (%d, 0)", kbRepo.gotSpaceID, kbRepo.gotLimit, spaceID)
	}

	// Verify deletes: must contain project_draft + coze_resource + kb_entries
	if len(fakeES.deletes) != 3 {
		t.Fatalf("deletes count = %d, want 3 (got %+v)", len(fakeES.deletes), fakeES.deletes)
	}
	wantDeletes := map[string]map[string]any{
		projectIndexName:  {"term": map[string]any{"space_id": spaceID}},
		resourceIndexName: {"term": map[string]any{"space_id": spaceID}},
		kbEntriesIndex:    {"terms": map[string]any{"kb_id": []int64{5001, 5002}}},
	}
	for _, d := range fakeES.deletes {
		want, ok := wantDeletes[d.index]
		if !ok {
			t.Errorf("unexpected delete on index %q", d.index)
			continue
		}
		if !reflect.DeepEqual(d.query, want) {
			t.Errorf("delete on %q query = %#v, want %#v", d.index, d.query, want)
		}
	}

	// Verify creates: 1 agent into project_draft (id "11"), 1 app into coze_resource (id "10"),
	// 2 kbs into kb_entries (ids "5001" + "5002")
	wantCreates := map[string]map[string]bool{
		projectIndexName:  {"11": true},
		resourceIndexName: {"10": true},
		kbEntriesIndex:    {"5001": true, "5002": true},
	}
	gotCreates := map[string]map[string]bool{}
	for _, c := range fakeES.creates {
		if gotCreates[c.index] == nil {
			gotCreates[c.index] = map[string]bool{}
		}
		gotCreates[c.index][c.id] = true
	}
	if !reflect.DeepEqual(gotCreates, wantCreates) {
		t.Errorf("creates = %#v, want %#v", gotCreates, wantCreates)
	}
}

func TestSearchSvc_ResyncSpace_NoKbs_SkipsKbDelete(t *testing.T) {
	spaceID := int64(200)

	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: &fakeAgentLister{ret: nil},
		appRepo:   &fakeAppLister{ret: nil},
		kbRepo:    &fakeKbLister{ret: nil},
	}

	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace returned err: %v", err)
	}
	if counts.ProjectDraft != 0 || counts.CozeResource != 0 || counts.KbEntries != 0 {
		t.Errorf("unexpected counts: %+v", counts)
	}

	// only 2 deletes (project_draft + coze_resource) when no KBs
	if len(fakeES.deletes) != 2 {
		t.Fatalf("deletes count = %d, want 2 (got %+v)", len(fakeES.deletes), fakeES.deletes)
	}
	for _, d := range fakeES.deletes {
		if d.index == kbEntriesIndex {
			t.Errorf("must NOT call DeleteByQuery on %q when no KBs", kbEntriesIndex)
		}
	}
}

func TestSearchSvc_ResyncSpace_DeleteErrorPropagates(t *testing.T) {
	spaceID := int64(300)
	wantErr := errors.New("boom-delete")

	fakeES := &fakeResyncESClient{
		deleteErr: map[string]error{projectIndexName: wantErr},
	}
	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: &fakeAgentLister{},
		appRepo:   &fakeAppLister{},
		kbRepo:    &fakeKbLister{},
	}

	_, err := svc.ResyncSpace(context.Background(), spaceID)
	if err == nil {
		t.Fatal("expected err, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wraps %v", err, wantErr)
	}
}

func TestSearchSvc_ResyncSpace_KbsListErrorPropagates(t *testing.T) {
	spaceID := int64(400)
	wantErr := errors.New("boom-kb-list")

	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: &fakeAgentLister{},
		appRepo:   &fakeAppLister{},
		kbRepo:    &fakeKbLister{retErr: wantErr},
	}

	_, err := svc.ResyncSpace(context.Background(), spaceID)
	if err == nil {
		t.Fatal("expected err, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wraps %v", err, wantErr)
	}
	// no deletes should have been issued before kb list failed
	if len(fakeES.deletes) != 0 {
		t.Errorf("unexpected deletes before kb list failure: %+v", fakeES.deletes)
	}
}

func TestSearchSvc_ResyncSpace_PartialCreateFailureContinues(t *testing.T) {
	spaceID := int64(500)

	kbs := []*KbInfo{
		{ID: 9001, SpaceID: spaceID, Name: "kb-good"},
		{ID: 9002, SpaceID: spaceID, Name: "kb-bad"},
		{ID: 9003, SpaceID: spaceID, Name: "kb-good-2"},
	}

	fakeES := &fakeResyncESClient{
		createErr: map[string]error{"9002": errors.New("boom-create")},
	}
	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: &fakeAgentLister{},
		appRepo:   &fakeAppLister{},
		kbRepo:    &fakeKbLister{ret: kbs},
	}

	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace returned err: %v (per-doc failures must NOT fail the whole call)", err)
	}
	if counts.KbEntries != 2 {
		t.Errorf("KbEntries = %d, want 2 (skip the failing kb)", counts.KbEntries)
	}
}
