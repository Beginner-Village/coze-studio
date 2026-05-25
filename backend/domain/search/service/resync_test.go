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

// fakeWorkflowLister is a minimal fake for WorkflowLister.
type fakeWorkflowLister struct {
	gotSpaceID int64
	gotLimit   int
	ret        []*WorkflowInfo
	retErr     error
}

func (f *fakeWorkflowLister) ListBySpaceID(_ context.Context, spaceID int64, limit int) ([]*WorkflowInfo, error) {
	f.gotSpaceID = spaceID
	f.gotLimit = limit
	return f.ret, f.retErr
}

// fakePluginLister is a minimal fake for PluginLister.
type fakePluginLister struct {
	gotSpaceID int64
	gotLimit   int
	ret        []*PluginInfoView
	retErr     error
}

func (f *fakePluginLister) ListBySpaceID(_ context.Context, spaceID int64, limit int) ([]*PluginInfoView, error) {
	f.gotSpaceID = spaceID
	f.gotLimit = limit
	return f.ret, f.retErr
}

// fakePromptLister is a minimal fake for PromptLister.
type fakePromptLister struct {
	gotSpaceID int64
	gotLimit   int
	ret        []*PromptInfo
	retErr     error
}

func (f *fakePromptLister) ListBySpaceID(_ context.Context, spaceID int64, limit int) ([]*PromptInfo, error) {
	f.gotSpaceID = spaceID
	f.gotLimit = limit
	return f.ret, f.retErr
}

// fakeDatabaseLister is a minimal fake for DatabaseLister.
type fakeDatabaseLister struct {
	gotSpaceID int64
	gotLimit   int
	ret        []*DatabaseInfo
	retErr     error
}

func (f *fakeDatabaseLister) ListBySpaceID(_ context.Context, spaceID int64, limit int) ([]*DatabaseInfo, error) {
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
		{ID: 5001, SpaceID: spaceID, Name: "kb-a", OwnerID: 9, CreatedAtMs: 100, UpdatedAtMs: 200, FormatType: 0},
		{ID: 5002, SpaceID: spaceID, Name: "kb-b", OwnerID: 9, CreatedAtMs: 110, UpdatedAtMs: 210, FormatType: 1},
	}
	agents := []*agententity.SingleAgent{newAgent(11, spaceID, 7, "agent-1")}
	apps := []*appentity.APP{
		{ID: 10, SpaceID: spaceID, OwnerID: 8, Name: ptr.Of("app-1"), CreatedAtMS: 300, UpdatedAtMS: 400},
	}
	workflows := []*WorkflowInfo{
		{ID: 6001, SpaceID: spaceID, OwnerID: 7, Name: "wf-1", Mode: 0, CreatedAtMs: 1000, UpdatedAtMs: 1100},
	}
	plugins := []*PluginInfoView{
		{ID: 6101, SpaceID: spaceID, OwnerID: 7, Name: "plug-1", PluginType: 1, CreatedAtMs: 2000, UpdatedAtMs: 2100},
	}
	prompts := []*PromptInfo{
		{ID: 6201, SpaceID: spaceID, OwnerID: 7, Name: "prm-1", CreatedAtMs: 3000, UpdatedAtMs: 3100},
	}
	databases := []*DatabaseInfo{
		{ID: 6301, SpaceID: spaceID, OwnerID: 7, Name: "db-1", CreatedAtMs: 4000, UpdatedAtMs: 4100},
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
	workflowRepo := &fakeWorkflowLister{ret: workflows}
	pluginRepo := &fakePluginLister{ret: plugins}
	promptRepo := &fakePromptLister{ret: prompts}
	databaseRepo := &fakeDatabaseLister{ret: databases}

	svc := &searchImpl{
		esClient:     fakeES,
		agentRepo:    agentRepo,
		appRepo:      appRepo,
		kbRepo:       kbRepo,
		workflowRepo: workflowRepo,
		pluginRepo:   pluginRepo,
		promptRepo:   promptRepo,
		databaseRepo: databaseRepo,
	}

	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace returned err: %v", err)
	}
	if counts == nil {
		t.Fatal("counts is nil")
	}
	// project_draft: 1 agent + 1 app = 2
	if counts.ProjectDraft != 2 {
		t.Errorf("ProjectDraft = %d, want 2 (1 agent + 1 app)", counts.ProjectDraft)
	}
	// coze_resource: 1 workflow + 1 plugin + 1 prompt + 1 database + 2 kbs = 6
	if counts.CozeResource != 6 {
		t.Errorf("CozeResource = %d, want 6 (1 wf + 1 plug + 1 prm + 1 db + 2 kb)", counts.CozeResource)
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
	if workflowRepo.gotSpaceID != spaceID || pluginRepo.gotSpaceID != spaceID ||
		promptRepo.gotSpaceID != spaceID || databaseRepo.gotSpaceID != spaceID {
		t.Errorf("workflow/plugin/prompt/database lister space ids = (%d,%d,%d,%d), want all %d",
			workflowRepo.gotSpaceID, pluginRepo.gotSpaceID, promptRepo.gotSpaceID, databaseRepo.gotSpaceID, spaceID)
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

	// Verify creates:
	//   project_draft   ← 1 agent (11), 1 app (10)
	//   coze_resource   ← 1 wf (6001), 1 plug (6101), 1 prm (6201), 1 db (6301), 2 kb (5001/5002)
	//   kb_entries      ← 2 kb (5001/5002)
	wantCreates := map[string]map[string]bool{
		projectIndexName: {"11": true, "10": true},
		resourceIndexName: {
			"6001": true, "6101": true, "6201": true, "6301": true,
			"5001": true, "5002": true,
		},
		kbEntriesIndex: {"5001": true, "5002": true},
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
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
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
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
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
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
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
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}

	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace returned err: %v (per-doc failures must NOT fail the whole call)", err)
	}
	if counts.KbEntries != 2 {
		t.Errorf("KbEntries = %d, want 2 (skip the failing kb)", counts.KbEntries)
	}
}

// TestSearchSvc_ResyncSpace_FirstDeleteFails: first delete (project_draft) fails.
// Expectation: returns err immediately, counts is nil, no subsequent delete or
// create traffic happens.
func TestSearchSvc_ResyncSpace_FirstDeleteFails(t *testing.T) {
	spaceID := int64(610)
	wantErr := errors.New("boom-project-delete")

	fakeES := &fakeResyncESClient{
		deleteErr: map[string]error{projectIndexName: wantErr},
	}
	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: &fakeAgentLister{ret: []*agententity.SingleAgent{newAgent(1, spaceID, 7, "should-not-index")}},
		appRepo:   &fakeAppLister{ret: []*appentity.APP{{ID: 99, SpaceID: spaceID, Name: ptr.Of("nope")}}},
		kbRepo:    &fakeKbLister{ret: nil},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}

	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err == nil {
		t.Fatal("expected err, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wraps %v", err, wantErr)
	}
	if counts != nil {
		t.Errorf("counts = %+v, want nil on early-abort", counts)
	}
	// Exactly 1 delete (the failing project_draft) — no coze_resource, no kb_entries.
	if len(fakeES.deletes) != 1 || fakeES.deletes[0].index != projectIndexName {
		t.Errorf("deletes = %+v, want only project_draft attempted", fakeES.deletes)
	}
	if len(fakeES.creates) != 0 {
		t.Errorf("creates = %+v, want no index writes after first-delete failure", fakeES.creates)
	}
}

// TestSearchSvc_ResyncSpace_SecondDeleteFails: project_draft delete OK, then
// coze_resource delete fails. We assert that:
//   - the error propagates
//   - the project_draft delete *did* execute (partial-inconsistency state is the
//     trade-off, and the caller is expected to log it)
//   - no create traffic happens (index rewrites are skipped on abort)
func TestSearchSvc_ResyncSpace_SecondDeleteFails(t *testing.T) {
	spaceID := int64(620)
	wantErr := errors.New("boom-resource-delete")

	fakeES := &fakeResyncESClient{
		deleteErr: map[string]error{resourceIndexName: wantErr},
	}
	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: &fakeAgentLister{ret: []*agententity.SingleAgent{newAgent(2, spaceID, 7, "a")}},
		appRepo:   &fakeAppLister{ret: []*appentity.APP{{ID: 50, SpaceID: spaceID, Name: ptr.Of("nope")}}},
		kbRepo:    &fakeKbLister{ret: nil},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}

	_, err := svc.ResyncSpace(context.Background(), spaceID)
	if err == nil {
		t.Fatal("expected err, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wraps %v", err, wantErr)
	}
	// Two deletes attempted (project_draft cleared OK, coze_resource failed).
	if len(fakeES.deletes) != 2 {
		t.Fatalf("deletes count = %d, want 2 (got %+v)", len(fakeES.deletes), fakeES.deletes)
	}
	gotIdx := []string{fakeES.deletes[0].index, fakeES.deletes[1].index}
	if gotIdx[0] != projectIndexName || gotIdx[1] != resourceIndexName {
		t.Errorf("delete order = %v, want [%s, %s]", gotIdx, projectIndexName, resourceIndexName)
	}
	// No create writes — abort before index rebuild step.
	if len(fakeES.creates) != 0 {
		t.Errorf("creates = %+v, want none after delete abort", fakeES.creates)
	}
}

// TestSearchSvc_ResyncSpace_KbEntriesDeleteFails_WhenKbsExist exercises the
// 3rd delete (kb_entries terms). All prior steps succeed, but the terms-based
// delete blows up.
func TestSearchSvc_ResyncSpace_KbEntriesDeleteFails_WhenKbsExist(t *testing.T) {
	spaceID := int64(630)
	wantErr := errors.New("boom-kb-entries-delete")

	kbs := []*KbInfo{
		{ID: 7001, SpaceID: spaceID, Name: "kb-a"},
		{ID: 7002, SpaceID: spaceID, Name: "kb-b"},
	}

	fakeES := &fakeResyncESClient{
		deleteErr: map[string]error{kbEntriesIndex: wantErr},
	}
	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: &fakeAgentLister{},
		appRepo:   &fakeAppLister{},
		kbRepo:    &fakeKbLister{ret: kbs},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}

	_, err := svc.ResyncSpace(context.Background(), spaceID)
	if err == nil {
		t.Fatal("expected err, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wraps %v", err, wantErr)
	}
	// All 3 deletes attempted.
	if len(fakeES.deletes) != 3 {
		t.Fatalf("deletes count = %d, want 3 (got %+v)", len(fakeES.deletes), fakeES.deletes)
	}
	// Verify kb_entries delete used the terms filter with the right kb_id list.
	for _, d := range fakeES.deletes {
		if d.index != kbEntriesIndex {
			continue
		}
		terms, ok := d.query["terms"].(map[string]any)
		if !ok {
			t.Errorf("kb_entries delete missing terms clause: %+v", d.query)
			break
		}
		ids, ok := terms["kb_id"].([]int64)
		if !ok || len(ids) != 2 || ids[0] != 7001 || ids[1] != 7002 {
			t.Errorf("kb_entries delete kb_id = %v, want [7001 7002]", terms["kb_id"])
		}
	}
	// No index rewrites after abort.
	if len(fakeES.creates) != 0 {
		t.Errorf("creates = %+v, want none after kb_entries delete abort", fakeES.creates)
	}
}

// TestSearchSvc_ResyncSpace_KbEntriesDelete_Skipped_WhenZeroKbs guards against
// the empty-terms footgun: an empty kb_id array would either no-op or 400 in
// real ES. The impl must skip the call entirely when there are no KBs.
func TestSearchSvc_ResyncSpace_KbEntriesDelete_Skipped_WhenZeroKbs(t *testing.T) {
	spaceID := int64(640)

	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:     fakeES,
		agentRepo:    &fakeAgentLister{},
		appRepo:      &fakeAppLister{},
		kbRepo:       &fakeKbLister{ret: nil}, // zero KBs
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}

	if _, err := svc.ResyncSpace(context.Background(), spaceID); err != nil {
		t.Fatalf("ResyncSpace err: %v", err)
	}

	for _, d := range fakeES.deletes {
		if d.index == kbEntriesIndex {
			t.Errorf("MUST NOT call DeleteByQuery(kb_entries) when zero KBs (got %+v)", d)
		}
	}
}

// TestSearchSvc_ResyncSpace_IndexAgentFailsForSome_OthersContinue verifies the
// per-agent resilience contract: if Create on the project_draft index returns
// an error for one agent in the middle of the loop, the rest still get
// indexed and the call as a whole returns success. Counts reflect only the
// successful indexes.
func TestSearchSvc_ResyncSpace_IndexAgentFailsForSome_OthersContinue(t *testing.T) {
	spaceID := int64(700)

	agents := []*agententity.SingleAgent{
		newAgent(11, spaceID, 7, "a1"),
		newAgent(22, spaceID, 7, "a2"), // this one will fail to index
		newAgent(33, spaceID, 7, "a3"),
		newAgent(44, spaceID, 7, "a4"),
		newAgent(55, spaceID, 7, "a5"),
	}

	fakeES := &fakeResyncESClient{
		createErr: map[string]error{"22": errors.New("boom-index-agent-22")},
	}
	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: &fakeAgentLister{ret: agents},
		appRepo:   &fakeAppLister{},
		kbRepo:    &fakeKbLister{},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}

	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace returned err: %v (per-agent failure must NOT abort)", err)
	}
	if counts.ProjectDraft != len(agents)-1 {
		t.Errorf("ProjectDraft = %d, want %d (skip 1 failed)", counts.ProjectDraft, len(agents)-1)
	}
	// All 5 agent creates were attempted (the loop did not short-circuit).
	creates := 0
	for _, c := range fakeES.creates {
		if c.index == projectIndexName {
			creates++
		}
	}
	if creates != len(agents) {
		t.Errorf("project_draft Create attempts = %d, want %d", creates, len(agents))
	}
}

// TestSearchSvc_ResyncSpace_IndexResourceFailsForSome_OthersContinue verifies
// the per-resource (coze_resource) resilience contract. Apps go to
// project_draft, so coze_resource is populated from workflow / plugin /
// prompt / database / kb. We use workflow rows here to exercise the loop.
func TestSearchSvc_ResyncSpace_IndexResourceFailsForSome_OthersContinue(t *testing.T) {
	spaceID := int64(710)

	wfs := []*WorkflowInfo{
		{ID: 100, SpaceID: spaceID, OwnerID: 8, Name: "wf-a"},
		{ID: 200, SpaceID: spaceID, OwnerID: 8, Name: "wf-b"}, // fails
		{ID: 300, SpaceID: spaceID, OwnerID: 8, Name: "wf-c"},
		{ID: 400, SpaceID: spaceID, OwnerID: 8, Name: "wf-d"},
	}

	fakeES := &fakeResyncESClient{
		createErr: map[string]error{"200": errors.New("boom-index-wf-200")},
	}
	svc := &searchImpl{
		esClient:     fakeES,
		agentRepo:    &fakeAgentLister{},
		appRepo:      &fakeAppLister{},
		kbRepo:       &fakeKbLister{},
		workflowRepo: &fakeWorkflowLister{ret: wfs},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}

	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace returned err: %v (per-workflow failure must NOT abort)", err)
	}
	if counts.CozeResource != len(wfs)-1 {
		t.Errorf("CozeResource = %d, want %d (skip 1 failed)", counts.CozeResource, len(wfs)-1)
	}
	creates := 0
	for _, c := range fakeES.creates {
		if c.index == resourceIndexName {
			creates++
		}
	}
	if creates != len(wfs) {
		t.Errorf("coze_resource Create attempts = %d, want %d", creates, len(wfs))
	}
}

// TestSearchSvc_ResyncSpace_IndexKbEntryFailsForSome_OthersContinue verifies
// the per-KB (kb_entries) resilience contract. This is a slightly different
// shape than TestSearchSvc_ResyncSpace_PartialCreateFailureContinues — that
// existing test fails the *middle* KB at index time; this one fails 2 out of
// 5 to make sure the counter doesn't off-by-one.
func TestSearchSvc_ResyncSpace_IndexKbEntryFailsForSome_OthersContinue(t *testing.T) {
	spaceID := int64(720)

	kbs := []*KbInfo{
		{ID: 1001, SpaceID: spaceID, Name: "k1"},
		{ID: 1002, SpaceID: spaceID, Name: "k2"}, // fails
		{ID: 1003, SpaceID: spaceID, Name: "k3"},
		{ID: 1004, SpaceID: spaceID, Name: "k4"}, // fails
		{ID: 1005, SpaceID: spaceID, Name: "k5"},
	}

	fakeES := &fakeResyncESClient{
		createErr: map[string]error{
			"1002": errors.New("boom-index-kb-1002"),
			"1004": errors.New("boom-index-kb-1004"),
		},
	}
	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: &fakeAgentLister{},
		appRepo:   &fakeAppLister{},
		kbRepo:    &fakeKbLister{ret: kbs},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}

	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace returned err: %v (per-kb index failure must NOT abort)", err)
	}
	if counts.KbEntries != 3 {
		t.Errorf("KbEntries = %d, want 3 (5 attempted, 2 failed)", counts.KbEntries)
	}
	creates := 0
	for _, c := range fakeES.creates {
		if c.index == kbEntriesIndex {
			creates++
		}
	}
	if creates != len(kbs) {
		t.Errorf("kb_entries Create attempts = %d, want %d", creates, len(kbs))
	}
}

// TestSearchSvc_ResyncSpace_EmptySpace_ZeroAgentsResourcesKbs covers the
// "scratch space" case: no agents, no apps, no KBs. project_draft +
// coze_resource deletes still fire (they're keyed on space_id, not on result
// presence), kb_entries is skipped, and no index writes happen.
func TestSearchSvc_ResyncSpace_EmptySpace_ZeroAgentsResourcesKbs(t *testing.T) {
	spaceID := int64(650)

	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:  fakeES,
		agentRepo: &fakeAgentLister{ret: nil},
		appRepo:   &fakeAppLister{ret: nil},
		kbRepo:    &fakeKbLister{ret: nil},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}

	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace returned err: %v", err)
	}
	if counts == nil {
		t.Fatal("counts nil on empty space — expected zeroed struct")
	}
	if counts.ProjectDraft != 0 || counts.CozeResource != 0 || counts.KbEntries != 0 {
		t.Errorf("counts = %+v, want all zeros", counts)
	}
	// Exactly 2 deletes (project_draft + coze_resource); no kb_entries terms.
	if len(fakeES.deletes) != 2 {
		t.Fatalf("deletes count = %d, want 2 (got %+v)", len(fakeES.deletes), fakeES.deletes)
	}
	for _, d := range fakeES.deletes {
		if d.index == kbEntriesIndex {
			t.Errorf("kb_entries delete fired on empty space: %+v", d)
		}
	}
	// Zero index helper calls.
	if len(fakeES.creates) != 0 {
		t.Errorf("creates = %+v, want none on empty space", fakeES.creates)
	}
}

// TestSearchSvc_ResyncSpace_AppsGoToProjectDraft confirms that apps are
// indexed in project_draft (matching the live event-bus path), NOT in
// coze_resource. This guards the regression where the original resync wrote
// apps into coze_resource which then mismatched what the create-flow does.
func TestSearchSvc_ResyncSpace_AppsGoToProjectDraft(t *testing.T) {
	spaceID := int64(810)
	apps := []*appentity.APP{
		{ID: 11, SpaceID: spaceID, OwnerID: 8, Name: ptr.Of("app-a"), CreatedAtMS: 1, UpdatedAtMS: 2},
		{ID: 12, SpaceID: spaceID, OwnerID: 8, Name: ptr.Of("app-b"), CreatedAtMS: 3, UpdatedAtMS: 4},
	}
	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:     fakeES,
		agentRepo:    &fakeAgentLister{},
		appRepo:      &fakeAppLister{ret: apps},
		kbRepo:       &fakeKbLister{},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}
	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace err: %v", err)
	}
	if counts.ProjectDraft != 2 {
		t.Errorf("ProjectDraft = %d, want 2 (both apps)", counts.ProjectDraft)
	}
	if counts.CozeResource != 0 {
		t.Errorf("CozeResource = %d, want 0 (apps must NOT land here)", counts.CozeResource)
	}
	// All app creates landed in project_draft.
	for _, c := range fakeES.creates {
		if c.index != projectIndexName {
			t.Errorf("app create on wrong index %q (id=%s), want %q", c.index, c.id, projectIndexName)
		}
	}
}

// TestSearchSvc_ResyncSpace_WorkflowsToCozeResource: workflow rows land in
// coze_resource with res_type=2.
func TestSearchSvc_ResyncSpace_WorkflowsToCozeResource(t *testing.T) {
	spaceID := int64(820)
	wfs := []*WorkflowInfo{
		{ID: 8201, SpaceID: spaceID, OwnerID: 1, Name: "wf-published", Mode: 0, HasPublish: true, CreatedAtMs: 10, UpdatedAtMs: 20},
		{ID: 8202, SpaceID: spaceID, OwnerID: 1, AppID: 999, Name: "wf-in-app", Mode: 3, HasPublish: false},
	}
	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:     fakeES,
		agentRepo:    &fakeAgentLister{},
		appRepo:      &fakeAppLister{},
		kbRepo:       &fakeKbLister{},
		workflowRepo: &fakeWorkflowLister{ret: wfs},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}
	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace err: %v", err)
	}
	if counts.CozeResource != 2 {
		t.Errorf("CozeResource = %d, want 2", counts.CozeResource)
	}
	gotIDs := map[string]bool{}
	for _, c := range fakeES.creates {
		if c.index != resourceIndexName {
			t.Errorf("workflow create on wrong index %q (id=%s)", c.index, c.id)
			continue
		}
		gotIDs[c.id] = true
	}
	if !gotIDs["8201"] || !gotIDs["8202"] {
		t.Errorf("missing expected workflow doc ids: got %v", gotIDs)
	}
}

// TestSearchSvc_ResyncSpace_PluginsToCozeResource: plugin rows land in
// coze_resource with res_type=1.
func TestSearchSvc_ResyncSpace_PluginsToCozeResource(t *testing.T) {
	spaceID := int64(830)
	plugins := []*PluginInfoView{
		{ID: 8301, SpaceID: spaceID, OwnerID: 1, Name: "plug-1", PluginType: 1},
		{ID: 8302, SpaceID: spaceID, OwnerID: 1, AppID: 7777, Name: "plug-in-app", PluginType: 6},
	}
	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:     fakeES,
		agentRepo:    &fakeAgentLister{},
		appRepo:      &fakeAppLister{},
		kbRepo:       &fakeKbLister{},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{ret: plugins},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}
	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace err: %v", err)
	}
	if counts.CozeResource != 2 {
		t.Errorf("CozeResource = %d, want 2", counts.CozeResource)
	}
	for _, c := range fakeES.creates {
		if c.index != resourceIndexName {
			t.Errorf("plugin create on wrong index %q (id=%s)", c.index, c.id)
		}
	}
}

// TestSearchSvc_ResyncSpace_PromptsToCozeResource: prompt rows land in
// coze_resource with res_type=6.
func TestSearchSvc_ResyncSpace_PromptsToCozeResource(t *testing.T) {
	spaceID := int64(840)
	prompts := []*PromptInfo{
		{ID: 8401, SpaceID: spaceID, OwnerID: 1, Name: "prompt-1"},
		{ID: 8402, SpaceID: spaceID, OwnerID: 2, Name: "prompt-2"},
	}
	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:     fakeES,
		agentRepo:    &fakeAgentLister{},
		appRepo:      &fakeAppLister{},
		kbRepo:       &fakeKbLister{},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{ret: prompts},
		databaseRepo: &fakeDatabaseLister{},
	}
	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace err: %v", err)
	}
	if counts.CozeResource != 2 {
		t.Errorf("CozeResource = %d, want 2", counts.CozeResource)
	}
	for _, c := range fakeES.creates {
		if c.index != resourceIndexName {
			t.Errorf("prompt create on wrong index %q (id=%s)", c.index, c.id)
		}
	}
}

// TestSearchSvc_ResyncSpace_DatabasesToCozeResource: database rows land in
// coze_resource with res_type=7.
func TestSearchSvc_ResyncSpace_DatabasesToCozeResource(t *testing.T) {
	spaceID := int64(850)
	dbs := []*DatabaseInfo{
		{ID: 8501, SpaceID: spaceID, OwnerID: 1, Name: "db-1"},
		{ID: 8502, SpaceID: spaceID, OwnerID: 1, AppID: 1234, Name: "db-in-app"},
	}
	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:     fakeES,
		agentRepo:    &fakeAgentLister{},
		appRepo:      &fakeAppLister{},
		kbRepo:       &fakeKbLister{},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{ret: dbs},
	}
	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace err: %v", err)
	}
	if counts.CozeResource != 2 {
		t.Errorf("CozeResource = %d, want 2", counts.CozeResource)
	}
	for _, c := range fakeES.creates {
		if c.index != resourceIndexName {
			t.Errorf("database create on wrong index %q (id=%s)", c.index, c.id)
		}
	}
}

// TestSearchSvc_ResyncSpace_KbAlsoLandsInCozeResource: knowledge bases are
// written to BOTH kb_entries (as before) AND coze_resource (ResType=4) so
// SearchResources can find KBs. Guards against regression where kbs only
// landed in kb_entries.
func TestSearchSvc_ResyncSpace_KbAlsoLandsInCozeResource(t *testing.T) {
	spaceID := int64(860)
	kbs := []*KbInfo{
		{ID: 8601, SpaceID: spaceID, OwnerID: 1, Name: "kb-1"},
	}
	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:     fakeES,
		agentRepo:    &fakeAgentLister{},
		appRepo:      &fakeAppLister{},
		kbRepo:       &fakeKbLister{ret: kbs},
		workflowRepo: &fakeWorkflowLister{},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}
	counts, err := svc.ResyncSpace(context.Background(), spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace err: %v", err)
	}
	// 1 kb → 1 CozeResource (res_type=4) AND 1 KbEntries.
	if counts.CozeResource != 1 {
		t.Errorf("CozeResource = %d, want 1 (kb also writes to coze_resource)", counts.CozeResource)
	}
	if counts.KbEntries != 1 {
		t.Errorf("KbEntries = %d, want 1", counts.KbEntries)
	}
	// Confirm both indices saw the kb id.
	sawCozeRes := false
	sawKbEntries := false
	for _, c := range fakeES.creates {
		if c.id == "8601" && c.index == resourceIndexName {
			sawCozeRes = true
		}
		if c.id == "8601" && c.index == kbEntriesIndex {
			sawKbEntries = true
		}
	}
	if !sawCozeRes {
		t.Errorf("missing coze_resource entry for kb 8601")
	}
	if !sawKbEntries {
		t.Errorf("missing kb_entries entry for kb 8601")
	}
}

// TestSearchSvc_ResyncSpace_AbortsOnWorkflowListError verifies a list error
// on any of the new 4 listers aborts with a wrapped error (uniform abort
// behaviour with existing agent/app/kb listers).
func TestSearchSvc_ResyncSpace_AbortsOnWorkflowListError(t *testing.T) {
	spaceID := int64(870)
	wantErr := errors.New("workflow-list-boom")
	fakeES := &fakeResyncESClient{}
	svc := &searchImpl{
		esClient:     fakeES,
		agentRepo:    &fakeAgentLister{},
		appRepo:      &fakeAppLister{},
		kbRepo:       &fakeKbLister{},
		workflowRepo: &fakeWorkflowLister{retErr: wantErr},
		pluginRepo:   &fakePluginLister{},
		promptRepo:   &fakePromptLister{},
		databaseRepo: &fakeDatabaseLister{},
	}
	_, err := svc.ResyncSpace(context.Background(), spaceID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wraps %v", err, wantErr)
	}
}

// TestSearchSvc_ResyncSpace_FailsWhenNewDepsMissing verifies that the resync
// short-circuits with a wiring error if any of the new 4 listers are nil.
func TestSearchSvc_ResyncSpace_FailsWhenNewDepsMissing(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*searchImpl)
	}{
		{"missing workflowRepo", func(s *searchImpl) { s.workflowRepo = nil }},
		{"missing pluginRepo", func(s *searchImpl) { s.pluginRepo = nil }},
		{"missing promptRepo", func(s *searchImpl) { s.promptRepo = nil }},
		{"missing databaseRepo", func(s *searchImpl) { s.databaseRepo = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &searchImpl{
				esClient:     &fakeResyncESClient{},
				agentRepo:    &fakeAgentLister{},
				appRepo:      &fakeAppLister{},
				kbRepo:       &fakeKbLister{},
				workflowRepo: &fakeWorkflowLister{},
				pluginRepo:   &fakePluginLister{},
				promptRepo:   &fakePromptLister{},
				databaseRepo: &fakeDatabaseLister{},
			}
			tc.setup(svc)
			_, err := svc.ResyncSpace(context.Background(), 1)
			if err == nil {
				t.Fatal("expected wiring error, got nil")
			}
		})
	}
}

// TestExtractPluginName verifies the manifest-name extractor handles common
// shapes (name_for_human, name_for_model, missing, broken JSON).
func TestExtractPluginName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`{"name_for_human": "Cool Plugin"}`, "Cool Plugin"},
		{`{"name_for_model": "cool_plugin"}`, "cool_plugin"},
		// human takes precedence over model.
		{`{"name_for_human": "H", "name_for_model": "M"}`, "H"},
		{`{}`, ""},
		{``, ""},
		{`not json at all`, ""},
		{`{"name_for_human":  "ws-padded"  }`, "ws-padded"},
	}
	for _, tc := range cases {
		got := extractPluginName(tc.in)
		if got != tc.want {
			t.Errorf("extractPluginName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
