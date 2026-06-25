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

package singleagent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	crossagent "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/application/skill"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	agententity "github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
	openauthentity "github.com/ynet-dev/ynet-studio/backend/domain/openauth/openapiauth/entity"
	skillentity "github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	sbx "github.com/ynet-dev/ynet-studio/backend/pkg/agentsandbox/contract"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
)

type fakeRuntimeSkillSandboxManager struct {
	fakeSandboxManager
	requested string
	files     map[string][]byte
}

func (f *fakeRuntimeSkillSandboxManager) Exec(_ context.Context, _ string, cmd string, _ int) (*sbx.ExecResponse, error) {
	f.requested = cmd
	payload := struct {
		OK    bool `json:"ok"`
		Files []struct {
			Path          string `json:"path"`
			ContentBase64 string `json:"content_base64"`
		} `json:"files"`
	}{OK: true}
	for p, content := range f.files {
		payload.Files = append(payload.Files, struct {
			Path          string `json:"path"`
			ContentBase64 string `json:"content_base64"`
		}{
			Path:          p,
			ContentBase64: base64.StdEncoding.EncodeToString(content),
		})
	}
	raw, _ := json.Marshal(payload)
	return &sbx.ExecResponse{Stdout: string(raw)}, nil
}

type fakeRuntimeSkillDomain struct {
	nextID    int64
	skills    map[int64]*skillentity.Skill
	published map[int64]int8
}

func (f *fakeRuntimeSkillDomain) CreateSkill(_ context.Context, s *skillentity.Skill) (int64, error) {
	if f.nextID == 0 {
		f.nextID = 900
	}
	if f.skills == nil {
		f.skills = map[int64]*skillentity.Skill{}
	}
	cloned := *s
	cloned.SkillID = f.nextID
	cloned.Version = 1
	cloned.Status = skillentity.SkillStatusActive
	f.skills[cloned.SkillID] = &cloned
	f.nextID++
	return cloned.SkillID, nil
}

func (f *fakeRuntimeSkillDomain) GetSkill(_ context.Context, skillID int64) (*skillentity.Skill, error) {
	return f.skills[skillID], nil
}

func (f *fakeRuntimeSkillDomain) GetSkillByName(_ context.Context, spaceID int64, name string) (*skillentity.Skill, error) {
	for _, s := range f.skills {
		if s.SpaceID == spaceID && s.Name == name {
			return s, nil
		}
	}
	return nil, nil
}

func (f *fakeRuntimeSkillDomain) UpdateSkill(_ context.Context, s *skillentity.Skill) error {
	existing := f.skills[s.SkillID]
	existing.Name = s.Name
	existing.Description = s.Description
	existing.Prompt = s.Prompt
	existing.IconURI = s.IconURI
	existing.Files = s.Files
	existing.Version++
	return nil
}

func (f *fakeRuntimeSkillDomain) DeleteSkill(context.Context, int64) error { return nil }

func (f *fakeRuntimeSkillDomain) PublishSkill(_ context.Context, skillID int64, scope, reviewStatus int8, version, publisherID, publishedAt int64) error {
	if f.published == nil {
		f.published = map[int64]int8{}
	}
	f.published[skillID] = scope
	if s := f.skills[skillID]; s != nil {
		s.PublishScope = scope
		s.ReviewStatus = reviewStatus
		s.PublishedVersion = version
		s.PublishedBy = publisherID
		s.PublishedAt = publishedAt
	}
	return nil
}

func (f *fakeRuntimeSkillDomain) ReviewSkill(_ context.Context, skillID int64, reviewStatus int8, note string, reviewerID, reviewedAt int64) error {
	if s := f.skills[skillID]; s != nil {
		s.ReviewStatus = reviewStatus
		s.ReviewNote = note
		s.ReviewerID = reviewerID
		s.ReviewedAt = reviewedAt
	}
	return nil
}

func (f *fakeRuntimeSkillDomain) ListPendingReviews(context.Context, *skillentity.PendingReviewListRequest) (*skillentity.ListResponse, error) {
	return nil, nil
}

func (f *fakeRuntimeSkillDomain) ListSkills(context.Context, *skillentity.ListRequest) (*skillentity.ListResponse, error) {
	return nil, nil
}

func (f *fakeRuntimeSkillDomain) ListMarketplaceSkills(context.Context, *skillentity.MarketplaceListRequest) (*skillentity.ListResponse, error) {
	return nil, nil
}

func (f *fakeRuntimeSkillDomain) MGetSkills(context.Context, []int64) ([]*skillentity.Skill, error) {
	return nil, nil
}

func (f *fakeRuntimeSkillDomain) GetSkillVersion(context.Context, int64, int64) (*skillentity.SkillVersion, error) {
	return nil, nil
}

func TestImportSuperAgentRuntimeSkillCreatesAndPublishesStandardSkill(t *testing.T) {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.OpenapiAuthKeyInCtx, &openauthentity.ApiKey{UserID: 77})

	fakeSandbox := &fakeRuntimeSkillSandboxManager{files: map[string][]byte{
		"SKILL.md":            []byte("---\nname: report-kit\ndescription: Build polished reports\n---\n# Report Kit\nUse scripts/render.py.\n"),
		"scripts/render.py":   []byte("print('render')\n"),
		"references/style.md": []byte("# Style\n"),
		"assets/logo.png":     []byte{0x89, 'P', 'N', 'G'},
	}}
	prevSandbox := crosssandbox.DefaultSVC()
	crosssandbox.SetDefaultSVC(fakeSandbox)
	defer crosssandbox.SetDefaultSVC(prevSandbox)

	fakeSkillDomain := &fakeRuntimeSkillDomain{}
	prevSkillApp := skill.SkillApplicationSVC
	skill.SkillApplicationSVC = &skill.SkillApplicationService{DomainSVC: fakeSkillDomain}
	defer func() { skill.SkillApplicationSVC = prevSkillApp }()

	agentID := int64(123)
	svc := &SingleAgentApplicationService{
		DomainSVC: &fakeSingleAgentDomain{
			draft: &agententity.SingleAgent{
				SingleAgent: &crossagent.SingleAgent{
					AgentID:   agentID,
					CreatorID: 77,
					SpaceID:   66,
				},
			},
		},
	}

	got, err := svc.ImportSuperAgentRuntimeSkill(ctx, &SuperAgentRuntimeSkillImportRequest{
		AgentID:      agentID,
		Name:         "report-kit",
		PublishScope: skillentity.SkillPublishScopeGlobal,
	})

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(66), got.SpaceID)
	assert.Equal(t, "report-kit", got.Name)
	assert.Equal(t, "Build polished reports", got.Description)
	assert.Equal(t, "# Report Kit\nUse scripts/render.py.", strings.TrimSpace(got.Prompt))
	assert.Equal(t, "print('render')\n", got.Files["scripts/render.py"])
	assert.Equal(t, "# Style\n", got.Files["references/style.md"])
	assert.True(t, strings.HasPrefix(got.Files["assets/logo.png"], "data:image/png;base64,"))
	assert.Equal(t, skillentity.SkillPublishScopeGlobal, fakeSkillDomain.published[got.SkillID])
	assert.Contains(t, fakeSandbox.requested, "/skills/report-kit")
}
