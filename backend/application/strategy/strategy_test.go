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

package strategy

import (
	"context"
	"fmt"
	"testing"

	knowledgeModel "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/knowledge"
	crossknowledge "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/knowledge"
	strategyEntity "github.com/ynet-dev/ynet-studio/backend/domain/strategy/entity"
)

// fakeKnowledgeSvc is a minimal crossknowledge.Knowledge fake: GetKnowledgeByID
// returns the space configured in spaceByID (missing id -> nil knowledge), every
// other method is a no-op. Mirrors the fakeSkillSvc / fakeSandboxMgr style used
// elsewhere in the codebase.
type fakeKnowledgeSvc struct {
	spaceByID map[int64]int64
	getErr    error
}

func (f *fakeKnowledgeSvc) GetKnowledgeByID(_ context.Context, req *knowledgeModel.GetKnowledgeByIDRequest) (*knowledgeModel.GetKnowledgeByIDResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	sp, ok := f.spaceByID[req.KnowledgeID]
	if !ok {
		return &knowledgeModel.GetKnowledgeByIDResponse{Knowledge: nil}, nil
	}
	return &knowledgeModel.GetKnowledgeByIDResponse{
		Knowledge: &knowledgeModel.Knowledge{Info: knowledgeModel.Info{SpaceID: sp}},
	}, nil
}

func (f *fakeKnowledgeSvc) ListKnowledge(context.Context, *knowledgeModel.ListKnowledgeRequest) (*knowledgeModel.ListKnowledgeResponse, error) {
	return nil, nil
}
func (f *fakeKnowledgeSvc) Retrieve(context.Context, *knowledgeModel.RetrieveRequest) (*knowledgeModel.RetrieveResponse, error) {
	return nil, nil
}
func (f *fakeKnowledgeSvc) DeleteKnowledge(context.Context, *knowledgeModel.DeleteKnowledgeRequest) error {
	return nil
}
func (f *fakeKnowledgeSvc) MGetKnowledgeByID(context.Context, *knowledgeModel.MGetKnowledgeByIDRequest) (*knowledgeModel.MGetKnowledgeByIDResponse, error) {
	return nil, nil
}
func (f *fakeKnowledgeSvc) Store(context.Context, *knowledgeModel.CreateDocumentRequest) (*knowledgeModel.CreateDocumentResponse, error) {
	return nil, nil
}
func (f *fakeKnowledgeSvc) Delete(context.Context, *knowledgeModel.DeleteDocumentRequest) (*knowledgeModel.DeleteDocumentResponse, error) {
	return nil, nil
}
func (f *fakeKnowledgeSvc) ListKnowledgeDetail(context.Context, *knowledgeModel.ListKnowledgeDetailRequest) (*knowledgeModel.ListKnowledgeDetailResponse, error) {
	return nil, nil
}

func TestValidateCapabilityRefInSpaceKnowledge(t *testing.T) {
	svc := &StrategyApplicationService{}
	crossknowledge.SetDefaultSVC(&fakeKnowledgeSvc{spaceByID: map[int64]int64{100: 11, 200: 22}})
	defer crossknowledge.SetDefaultSVC(nil)

	// A knowledge base living in the same space is allowed.
	if err := svc.validateCapabilityRefInSpace(context.Background(), strategyEntity.CapabilityTypeKnowledge, 100, 0, 11); err != nil {
		t.Fatalf("knowledge 100 lives in space 11 and must be allowed, got: %v", err)
	}
	// A knowledge base owned by another space must be rejected (cross-space IDOR).
	if err := svc.validateCapabilityRefInSpace(context.Background(), strategyEntity.CapabilityTypeKnowledge, 200, 0, 11); err == nil {
		t.Fatal("knowledge 200 lives in space 22 and must be rejected for space 11")
	}
	// A knowledge id that cannot be resolved must fail closed (rejected).
	if err := svc.validateCapabilityRefInSpace(context.Background(), strategyEntity.CapabilityTypeKnowledge, 999, 0, 11); err == nil {
		t.Fatal("unresolvable knowledge must fail closed (rejected), got nil")
	}
}

func TestValidateCapabilityRefInSpaceKnowledgeServiceError(t *testing.T) {
	svc := &StrategyApplicationService{}
	crossknowledge.SetDefaultSVC(&fakeKnowledgeSvc{getErr: fmt.Errorf("boom")})
	defer crossknowledge.SetDefaultSVC(nil)

	// A transient service error surfaces as a rejection (fail closed), never as
	// silent acceptance.
	if err := svc.validateCapabilityRefInSpace(context.Background(), strategyEntity.CapabilityTypeKnowledge, 100, 0, 11); err == nil {
		t.Fatal("knowledge resolution error must fail closed (rejected), got nil")
	}
}
