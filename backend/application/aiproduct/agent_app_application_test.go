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

package aiproduct

import (
	"context"
	"testing"

	productentity "github.com/ynet-dev/ynet-studio/backend/domain/aiproduct/entity"
)

// ---- fakeAgentReader -------------------------------------------------------

type fakeAgentReader struct {
	draft *AgentDraftView
}

func (f *fakeAgentReader) GetDraft(_ context.Context, _ int64) (*AgentDraftView, error) {
	if f.draft == nil {
		return &AgentDraftView{
			ModelID: "qwen3",
			Prompt:  "You are a helpful assistant.",
		}, nil
	}
	return f.draft, nil
}

// ---- fakeShadowAgentWriter -------------------------------------------------

type fakeShadowWriter struct {
	lastProductID int64
	nextID        int64
}

func (f *fakeShadowWriter) CreateShadowDraft(_ context.Context, _, _, productID int64, _ string, _ map[string]any) (int64, error) {
	f.lastProductID = productID
	f.nextID++
	return f.nextID, nil
}

// ---- fakeDomainSVC ---------------------------------------------------------

type fakeDomainSVC struct {
	lastSyncedType string
	nextProductID  int64
}

func (f *fakeDomainSVC) SyncProduct(_ context.Context, product *productentity.Product, _ *productentity.ProductVersion) (*productentity.Product, error) {
	f.lastSyncedType = string(product.Type)
	if product.ProductID == 0 {
		f.nextProductID++
		product.ProductID = f.nextProductID
	}
	return product, nil
}

func (f *fakeDomainSVC) ListMarketplace(_ context.Context, _ *productentity.ListProductsRequest) (*productentity.ListProductsResult, error) {
	return nil, nil
}

func (f *fakeDomainSVC) GetVisibleProduct(_ context.Context, _, _, _ int64) (*productentity.Product, error) {
	return nil, nil
}

func (f *fakeDomainSVC) Install(_ context.Context, productID, spaceID, userID int64, version string) (*productentity.ProductInstallation, error) {
	return &productentity.ProductInstallation{
		InstallationID: 1,
		ProductID:      productID,
		TargetSpaceID:  spaceID,
		TargetUserID:   userID,
		ProductVersion: version,
		Status:         productentity.AIProductInstallationActive,
	}, nil
}

func (f *fakeDomainSVC) Uninstall(_ context.Context, _, _, _ int64) error { return nil }

func (f *fakeDomainSVC) Upgrade(_ context.Context, _, _, _ int64) (*productentity.ProductInstallation, error) {
	return nil, nil
}

func (f *fakeDomainSVC) ListInstalled(_ context.Context, _, _ int64, _ productentity.AIProductType) ([]*productentity.ProductInstallation, error) {
	return nil, nil
}

func (f *fakeDomainSVC) RecordAudit(_ context.Context, _ *productentity.AuditLog) error {
	return nil
}

// ---- testAgentAppApp (wraps AgentAppApplication with exported fakes) -------

type testAgentAppApp struct {
	*AgentAppApplication
	fakeSvc     *fakeDomainSVC
	fakeSandbox *fakeSandbox
	fakeShadow  *fakeShadowWriter
}

func newTestAgentAppApp(_ *testing.T) *testAgentAppApp {
	svc := &fakeDomainSVC{nextProductID: 100}
	sb := &fakeSandbox{}
	shadow := &fakeShadowWriter{}

	app := &AgentAppApplication{
		DomainSVC:    svc,
		AgentReader:  &fakeAgentReader{},
		ShadowWriter: shadow,
		Sandbox:      sb,
		ObjectPrefix: "templates/agent_app",
	}
	return &testAgentAppApp{
		AgentAppApplication: app,
		fakeSvc:             svc,
		fakeSandbox:         sb,
		fakeShadow:          shadow,
	}
}

// ---- Tests -----------------------------------------------------------------

func TestPublishAgentAppFreezesSnapshotAndBuildsTemplate(t *testing.T) {
	app := newTestAgentAppApp(t) // helper wires fakes; build returns ready
	pid, ver, err := app.PublishAgentApp(context.Background(), PublishReq{
		AgentID: 7, SpaceID: 1, UserID: 9, Name: "PDF 助手", Version: "1",
	})
	if err != nil || pid == 0 || ver != "1" {
		t.Fatalf("publish failed: pid=%d ver=%s err=%v", pid, ver, err)
	}
	if got := app.fakeSvc.lastSyncedType; got != "agent_app" {
		t.Fatalf("product type not agent_app: %s", got)
	}
	if app.fakeSandbox.checkpoint == "" {
		t.Fatalf("template not built")
	}
}

func TestRecruitMaterializesReadOnlyShadow(t *testing.T) {
	app := newTestAgentAppApp(t)
	shadowID, err := app.RecruitAgentApp(context.Background(), 100, 1, 9)
	if err != nil || shadowID == 0 {
		t.Fatalf("recruit failed: id=%d err=%v", shadowID, err)
	}
	if app.fakeShadow.lastProductID != 100 {
		t.Fatalf("shadow not linked to product: %d", app.fakeShadow.lastProductID)
	}
}
