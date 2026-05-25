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

// This file holds the "drift convergence" suite for ResyncSpace. The point
// is to verify that no matter what state ES is in before the call, the
// state *after* the call equals the MySQL projection — i.e. resync truly
// converges.
//
// We cover the three real-world starting points the user asked about:
//   (A) ES is completely empty (MySQL has rows)
//   (B) ES is partially populated (some rows missing, some present and
//       possibly stale-but-correct)
//   (C) ES is completely wrong (chaos — random extra docs in the target
//       space + docs from other spaces that must NOT be touched)
//
// We assert on project_draft + coze_resource only. kb_entries is excluded
// from the assertions on purpose — that index is owned by a different
// system (Guard's FAQ/risk-keyword store, ULID-keyed) and its inclusion
// in Studio's resync is being removed in a separate task. The stateful
// fake ES still allows writes to kb_entries so the call doesn't error;
// we just don't assert on it here.

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	agententity "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	appentity "github.com/ynet-dev/ynet-studio/backend/domain/app/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/search/entity"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/es"
	"github.com/ynet-dev/ynet-studio/backend/pkg/lang/ptr"
)

// statefulES is a fake es.Client that maintains a real per-index doc map
// across Create / DeleteByQuery calls, the same way real ES would —
// without the network. It supports the exact two query shapes ResyncSpace
// emits:
//
//	{"term":  {"space_id": <int64>}}           // per-space wipe
//	{"terms": {"kb_id":    [<int64>, ...]}}    // per-KB wipe (legacy)
//
// Anything else panics so the test surfaces accidental wiring changes.
type statefulES struct {
	// docs[indexName][docID] = doc body (always a typed *entity.* struct).
	docs map[string]map[string]any
}

func newStatefulES() *statefulES {
	return &statefulES{docs: map[string]map[string]any{}}
}

func (s *statefulES) put(index, id string, body any) {
	if s.docs[index] == nil {
		s.docs[index] = map[string]any{}
	}
	s.docs[index][id] = body
}

func (s *statefulES) Create(_ context.Context, index, id string, body any) error {
	s.put(index, id, body)
	return nil
}

// DeleteByQuery interprets the two query shapes ResyncSpace uses and deletes
// the matching docs from the index map. Returns the number actually deleted.
func (s *statefulES) DeleteByQuery(_ context.Context, index string, query map[string]any) (int64, error) {
	bucket := s.docs[index]
	if bucket == nil {
		return 0, nil
	}

	// Decode the query into (field, []int64 acceptable values).
	field, accept, err := decodeESQuery(query)
	if err != nil {
		return 0, err
	}
	acceptSet := map[int64]struct{}{}
	for _, v := range accept {
		acceptSet[v] = struct{}{}
	}

	var deleted int64
	for id, body := range bucket {
		v, ok := readInt64Field(body, field)
		if !ok {
			// Field absent on this doc → cannot match, leave it alone. This
			// mirrors real ES: term/terms on a missing field excludes the doc.
			continue
		}
		if _, hit := acceptSet[v]; hit {
			delete(bucket, id)
			deleted++
		}
	}
	return deleted, nil
}

func decodeESQuery(q map[string]any) (field string, accept []int64, err error) {
	if termClause, ok := q["term"].(map[string]any); ok {
		if len(termClause) != 1 {
			return "", nil, fmt.Errorf("statefulES: term must have exactly 1 field, got %d", len(termClause))
		}
		for k, v := range termClause {
			n, ok := v.(int64)
			if !ok {
				return "", nil, fmt.Errorf("statefulES: term field %q value must be int64, got %T", k, v)
			}
			return k, []int64{n}, nil
		}
	}
	if termsClause, ok := q["terms"].(map[string]any); ok {
		if len(termsClause) != 1 {
			return "", nil, fmt.Errorf("statefulES: terms must have exactly 1 field, got %d", len(termsClause))
		}
		for k, v := range termsClause {
			ns, ok := v.([]int64)
			if !ok {
				return "", nil, fmt.Errorf("statefulES: terms field %q value must be []int64, got %T", k, v)
			}
			return k, ns, nil
		}
	}
	return "", nil, fmt.Errorf("statefulES: unrecognized query shape %v", q)
}

// readInt64Field extracts an int64 value for `field` (the ES field name) from
// a typed doc body. Only the fields ResyncSpace's DeleteByQuery actually
// queries on are supported — space_id (always present on both ProjectDocument
// and ResourceDocument) and kb_id (never present on either, by design).
func readInt64Field(body any, field string) (int64, bool) {
	switch field {
	case "space_id":
		switch d := body.(type) {
		case *entity.ProjectDocument:
			if d.SpaceID != nil {
				return *d.SpaceID, true
			}
		case *entity.ResourceDocument:
			if d.SpaceID != nil {
				return *d.SpaceID, true
			}
		}
	case "kb_id":
		// ProjectDocument / ResourceDocument do not carry a kb_id field, so
		// the terms-on-kb_id query never matches anything we write. Real ES
		// would behave the same way. This is the documented bug that's
		// being removed in a follow-up task.
	}
	return 0, false
}

func (s *statefulES) snapshotIDs(index string) []string {
	if s.docs[index] == nil {
		return nil
	}
	ids := make([]string, 0, len(s.docs[index]))
	for id := range s.docs[index] {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Unused / panic-on-call methods (resync only touches Create + DeleteByQuery).
func (s *statefulES) Update(context.Context, string, string, any) error { panic("not used") }
func (s *statefulES) Delete(context.Context, string, string) error      { panic("not used") }
func (s *statefulES) Search(context.Context, string, *es.Request) (*es.Response, error) {
	panic("not used")
}
func (s *statefulES) Exists(context.Context, string) (bool, error) { panic("not used") }
func (s *statefulES) CreateIndex(context.Context, string, map[string]any) error {
	panic("not used")
}
func (s *statefulES) DeleteIndex(context.Context, string) error     { panic("not used") }
func (s *statefulES) Types() es.Types                               { panic("not used") }
func (s *statefulES) NewBulkIndexer(string) (es.BulkIndexer, error) { panic("not used") }

// driftFixture is the shared "MySQL state" all three drift scenarios run
// against. The expected post-resync ES state derives directly from it.
type driftFixture struct {
	spaceID   int64
	agents    []*agententity.SingleAgent
	apps      []*appentity.APP
	workflows []*WorkflowInfo
	plugins   []*PluginInfoView
	prompts   []*PromptInfo
	databases []*DatabaseInfo
	kbs       []*KbInfo
}

func makeDriftFixture() *driftFixture {
	const spaceID int64 = 7100
	return &driftFixture{
		spaceID: spaceID,
		agents: []*agententity.SingleAgent{
			newAgent(11001, spaceID, 7, "agent-A"),
			newAgent(11002, spaceID, 7, "agent-B"),
		},
		apps: []*appentity.APP{
			{ID: 12001, SpaceID: spaceID, OwnerID: 8, Name: ptr.Of("app-1"), CreatedAtMS: 300, UpdatedAtMS: 400},
		},
		workflows: []*WorkflowInfo{
			{ID: 21001, SpaceID: spaceID, OwnerID: 7, Name: "wf-1", Mode: 0, CreatedAtMs: 1000, UpdatedAtMs: 1100},
			{ID: 21002, SpaceID: spaceID, OwnerID: 7, Name: "wf-2", Mode: 1, CreatedAtMs: 1000, UpdatedAtMs: 1200},
			{ID: 21003, SpaceID: spaceID, OwnerID: 7, Name: "wf-3", Mode: 0, CreatedAtMs: 1000, UpdatedAtMs: 1300, HasPublish: true},
		},
		plugins: []*PluginInfoView{
			{ID: 22001, SpaceID: spaceID, OwnerID: 7, Name: "plug-1", PluginType: 1, CreatedAtMs: 2000, UpdatedAtMs: 2100},
			{ID: 22002, SpaceID: spaceID, OwnerID: 7, Name: "plug-2", PluginType: 1, CreatedAtMs: 2000, UpdatedAtMs: 2200},
		},
		prompts: []*PromptInfo{
			{ID: 23001, SpaceID: spaceID, OwnerID: 7, Name: "prm-1", CreatedAtMs: 3000, UpdatedAtMs: 3100},
		},
		databases: []*DatabaseInfo{
			{ID: 24001, SpaceID: spaceID, OwnerID: 7, Name: "db-1", CreatedAtMs: 4000, UpdatedAtMs: 4100},
		},
		kbs: []*KbInfo{
			{ID: 25001, SpaceID: spaceID, Name: "kb-A", OwnerID: 9, FormatType: 0, CreatedAtMs: 5000, UpdatedAtMs: 5100},
			{ID: 25002, SpaceID: spaceID, Name: "kb-B", OwnerID: 9, FormatType: 1, CreatedAtMs: 5000, UpdatedAtMs: 5200},
		},
	}
}

// expectedFinalIDs returns the deduped sorted ID lists each index must hold
// after a successful resync, given the fixture. Other-space docs that the
// scenario pre-populated for "must not touch" assertions are added in by
// the caller.
func (f *driftFixture) expectedFinalIDs() (projectDraft []string, cozeResource []string) {
	for _, a := range f.agents {
		projectDraft = append(projectDraft, fmt.Sprintf("%d", a.AgentID))
	}
	for _, a := range f.apps {
		projectDraft = append(projectDraft, fmt.Sprintf("%d", a.ID))
	}
	for _, w := range f.workflows {
		cozeResource = append(cozeResource, fmt.Sprintf("%d", w.ID))
	}
	for _, p := range f.plugins {
		cozeResource = append(cozeResource, fmt.Sprintf("%d", p.ID))
	}
	for _, p := range f.prompts {
		cozeResource = append(cozeResource, fmt.Sprintf("%d", p.ID))
	}
	for _, d := range f.databases {
		cozeResource = append(cozeResource, fmt.Sprintf("%d", d.ID))
	}
	for _, kb := range f.kbs {
		cozeResource = append(cozeResource, fmt.Sprintf("%d", kb.ID))
	}
	sort.Strings(projectDraft)
	sort.Strings(cozeResource)
	return
}

func (f *driftFixture) makeSvc(esCli es.Client) *searchImpl {
	return &searchImpl{
		esClient:     esCli,
		agentRepo:    &fakeAgentLister{ret: f.agents},
		appRepo:      &fakeAppLister{ret: f.apps},
		kbRepo:       &fakeKbLister{ret: f.kbs},
		workflowRepo: &fakeWorkflowLister{ret: f.workflows},
		pluginRepo:   &fakePluginLister{ret: f.plugins},
		promptRepo:   &fakePromptLister{ret: f.prompts},
		databaseRepo: &fakeDatabaseLister{ret: f.databases},
	}
}

// assertConverged is the shared end-state check used by all 3 drift tests.
// extraProjectIDs / extraResourceIDs are IDs the test pre-populated in
// *other* spaces that must survive untouched.
func assertConverged(t *testing.T, fake *statefulES, fx *driftFixture, extraProjectIDs, extraResourceIDs []string) {
	t.Helper()

	wantProject, wantResource := fx.expectedFinalIDs()
	wantProject = mergeSorted(wantProject, extraProjectIDs)
	wantResource = mergeSorted(wantResource, extraResourceIDs)

	gotProject := fake.snapshotIDs(projectIndexName)
	gotResource := fake.snapshotIDs(resourceIndexName)

	if !reflect.DeepEqual(gotProject, wantProject) {
		t.Errorf("project_draft IDs after resync:\n  got  %v\n  want %v", gotProject, wantProject)
	}
	if !reflect.DeepEqual(gotResource, wantResource) {
		t.Errorf("coze_resource IDs after resync:\n  got  %v\n  want %v", gotResource, wantResource)
	}
}

func mergeSorted(a, b []string) []string {
	out := append([]string{}, a...)
	out = append(out, b...)
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// Scenario A: ES completely empty + MySQL has data.
// ---------------------------------------------------------------------------
//
// Pre-condition: ES is fresh, no docs at all. The 3 list indices don't even
// have entries — DeleteByQuery sees empty buckets, Create writes everything
// from scratch.
//
// Expected: ES converges to exactly the MySQL projection.

func TestSearchSvc_ResyncSpace_Drift_EmptyES_ConvergesToMySQL(t *testing.T) {
	fx := makeDriftFixture()
	fake := newStatefulES()
	svc := fx.makeSvc(fake)

	counts, err := svc.ResyncSpace(context.Background(), fx.spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace: %v", err)
	}
	// Sanity-check counts (project_draft = 2 agents + 1 app = 3; coze_resource
	// = 3 wf + 2 plug + 1 prm + 1 db + 2 kb = 9).
	if counts.ProjectDraft != 3 || counts.CozeResource != 9 {
		t.Errorf("counts = %+v, want ProjectDraft=3 CozeResource=9", counts)
	}

	assertConverged(t, fake, fx, nil, nil)
}

// ---------------------------------------------------------------------------
// Scenario B: ES partially populated + MySQL has data.
// ---------------------------------------------------------------------------
//
// Pre-condition: ES already has SOME of the right docs (1 agent + 2 workflows
// + 1 KB-as-resource — about half the expected set). The other half is
// missing. This is what you'd see if a write outage truncated some
// production writes mid-flight.
//
// Expected: resync deletes them all (because delete is space-wide, not
// per-doc) and rewrites everything. Final state is identical to scenario A.

func TestSearchSvc_ResyncSpace_Drift_PartialES_ConvergesToMySQL(t *testing.T) {
	fx := makeDriftFixture()
	fake := newStatefulES()

	// Half the expected docs already exist in ES, identical content.
	fake.put(projectIndexName, "11001", &entity.ProjectDocument{
		ID:      11001,
		SpaceID: ptr.Of(fx.spaceID),
		Name:    ptr.Of("agent-A"),
	})
	fake.put(resourceIndexName, "21001", &entity.ResourceDocument{
		ResID:   21001,
		SpaceID: ptr.Of(fx.spaceID),
		Name:    ptr.Of("wf-1"),
	})
	fake.put(resourceIndexName, "21002", &entity.ResourceDocument{
		ResID:   21002,
		SpaceID: ptr.Of(fx.spaceID),
		Name:    ptr.Of("wf-2"),
	})
	fake.put(resourceIndexName, "25001", &entity.ResourceDocument{
		ResID:   25001,
		SpaceID: ptr.Of(fx.spaceID),
		Name:    ptr.Of("kb-A"),
	})

	svc := fx.makeSvc(fake)
	counts, err := svc.ResyncSpace(context.Background(), fx.spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace: %v", err)
	}
	// Counts are about MySQL→ES writes, not about pre-existing docs, so
	// they should still be 3 / 9.
	if counts.ProjectDraft != 3 || counts.CozeResource != 9 {
		t.Errorf("counts = %+v, want ProjectDraft=3 CozeResource=9", counts)
	}

	assertConverged(t, fake, fx, nil, nil)
}

// ---------------------------------------------------------------------------
// Scenario C: ES is full of stale/wrong docs + docs from other spaces.
// ---------------------------------------------------------------------------
//
// Pre-condition:
//   - project_draft has 2 stale docs in our space (IDs we no longer own),
//     plus 1 doc in a *different* space that must NOT be touched.
//   - coze_resource has 4 random docs in our space (no MySQL backing),
//     plus 2 docs in 2 *different* spaces — those must NOT be touched.
//
// Expected:
//   - Stale in-space docs are wiped by delete_by_query{space_id: ours}.
//   - Other-space docs survive (their space_id doesn't match).
//   - Final state = MySQL projection ∪ other-space pre-existing docs.

func TestSearchSvc_ResyncSpace_Drift_ChaosES_ConvergesToMySQL(t *testing.T) {
	fx := makeDriftFixture()
	const otherSpaceA int64 = 9001
	const otherSpaceB int64 = 9002
	fake := newStatefulES()

	// In-space chaos — should all get wiped.
	fake.put(projectIndexName, "999111", &entity.ProjectDocument{
		ID: 999111, SpaceID: ptr.Of(fx.spaceID), Name: ptr.Of("stale-agent"),
	})
	fake.put(projectIndexName, "999112", &entity.ProjectDocument{
		ID: 999112, SpaceID: ptr.Of(fx.spaceID), Name: ptr.Of("stale-app"),
	})
	fake.put(resourceIndexName, "888001", &entity.ResourceDocument{
		ResID: 888001, SpaceID: ptr.Of(fx.spaceID), Name: ptr.Of("stale-wf"),
	})
	fake.put(resourceIndexName, "888002", &entity.ResourceDocument{
		ResID: 888002, SpaceID: ptr.Of(fx.spaceID), Name: ptr.Of("stale-plug"),
	})
	fake.put(resourceIndexName, "888003", &entity.ResourceDocument{
		ResID: 888003, SpaceID: ptr.Of(fx.spaceID), Name: ptr.Of("stale-prm"),
	})
	fake.put(resourceIndexName, "888004", &entity.ResourceDocument{
		ResID: 888004, SpaceID: ptr.Of(fx.spaceID), Name: ptr.Of("stale-db"),
	})

	// Other-space docs — MUST survive.
	fake.put(projectIndexName, "777111", &entity.ProjectDocument{
		ID: 777111, SpaceID: ptr.Of(otherSpaceA), Name: ptr.Of("other-space-agent"),
	})
	fake.put(resourceIndexName, "666001", &entity.ResourceDocument{
		ResID: 666001, SpaceID: ptr.Of(otherSpaceA), Name: ptr.Of("other-space-wf"),
	})
	fake.put(resourceIndexName, "666002", &entity.ResourceDocument{
		ResID: 666002, SpaceID: ptr.Of(otherSpaceB), Name: ptr.Of("other-space-plug"),
	})

	svc := fx.makeSvc(fake)
	counts, err := svc.ResyncSpace(context.Background(), fx.spaceID)
	if err != nil {
		t.Fatalf("ResyncSpace: %v", err)
	}
	if counts.ProjectDraft != 3 || counts.CozeResource != 9 {
		t.Errorf("counts = %+v, want ProjectDraft=3 CozeResource=9", counts)
	}

	extraProject := []string{"777111"}
	extraResource := []string{"666001", "666002"}
	assertConverged(t, fake, fx, extraProject, extraResource)

	// Spot-check: the stale in-space IDs are GONE.
	for _, id := range []string{"999111", "999112"} {
		if _, ok := fake.docs[projectIndexName][id]; ok {
			t.Errorf("stale project_draft id %s should have been deleted", id)
		}
	}
	for _, id := range []string{"888001", "888002", "888003", "888004"} {
		if _, ok := fake.docs[resourceIndexName][id]; ok {
			t.Errorf("stale coze_resource id %s should have been deleted", id)
		}
	}
}
