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

package agentflow

import (
	"context"
	"strings"
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	crosssandbox "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/sandbox"
	crossskill "github.com/ynet-dev/ynet-studio/backend/crossdomain/contract/skill"
	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
)

type fakeSkillSvc struct {
	skills map[int64]*entity.Skill
}

func (s *fakeSkillSvc) GetSkill(_ context.Context, skillID int64) (*entity.Skill, error) {
	return s.skills[skillID], nil
}

func (s *fakeSkillSvc) GetSkillByName(_ context.Context, spaceID int64, name string) (*entity.Skill, error) {
	for _, skill := range s.skills {
		if skill.SpaceID == spaceID && skill.Name == name {
			return skill, nil
		}
	}
	return nil, nil
}

func (s *fakeSkillSvc) MGetSkills(_ context.Context, skillIDs []int64) ([]*entity.Skill, error) {
	result := make([]*entity.Skill, 0, len(skillIDs))
	for _, skillID := range skillIDs {
		if skill := s.skills[skillID]; skill != nil {
			result = append(result, skill)
		}
	}
	return result, nil
}

func TestParseSkillFiles(t *testing.T) {
	prompt := `# PDF 技能
处理 PDF 时运行下面的脚本。

<skill-file path="scripts/run.py">
import sys
print("processing", sys.argv)
</skill-file>

完成后告诉用户结果。

<skill-file path="scripts/helper.sh">
echo helper
</skill-file>`

	cleaned, files := parseSkillFiles(prompt)
	if len(files) != 2 {
		t.Fatalf("want 2 files, got %d (%v)", len(files), files)
	}
	if !strings.Contains(string(files["scripts/run.py"]), `print("processing"`) {
		t.Fatalf("run.py content wrong: %q", files["scripts/run.py"])
	}
	if string(files["scripts/helper.sh"]) != "echo helper" {
		t.Fatalf("helper.sh content wrong: %q", files["scripts/helper.sh"])
	}
	if strings.Contains(cleaned, "<skill-file") {
		t.Fatalf("cleaned prompt should not contain file blocks: %q", cleaned)
	}
	if !strings.Contains(cleaned, "处理 PDF 时运行下面的脚本") {
		t.Fatalf("cleaned prompt lost instructions: %q", cleaned)
	}
}

func TestParseSkillFilesNone(t *testing.T) {
	prompt := "just instructions, no files"
	cleaned, files := parseSkillFiles(prompt)
	if files != nil {
		t.Fatalf("want nil files, got %v", files)
	}
	if cleaned != prompt {
		t.Fatalf("cleaned should equal original")
	}
}

func TestParseSkillFilesRejectsUnsafePaths(t *testing.T) {
	prompt := `<skill-file path="../etc/passwd">x</skill-file><skill-file path="/abs">y</skill-file><skill-file path="ok.py">z</skill-file>`
	_, files := parseSkillFiles(prompt)
	if len(files) != 1 {
		t.Fatalf("want only safe path kept, got %v", files)
	}
	if _, ok := files["ok.py"]; !ok {
		t.Fatalf("safe path not kept: %v", files)
	}
}

func TestReadSkillToolUsesStandardSkillFilesAsInstructions(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	crossskill.SetDefaultSVC(&fakeSkillSvc{skills: map[int64]*entity.Skill{
		7: {
			SkillID: 7,
			SpaceID: 11,
			Name:    "report",
			Prompt:  "legacy prompt should not be returned",
			Files: map[string]string{
				"SKILL.md":           "# Report Skill\nUse scripts/run.py.",
				"scripts/run.py":     "print('ok')",
				"references/spec.md": "details",
			},
		},
	}})
	defer crossskill.SetDefaultSVC(nil)

	readTool := newReadSkillTool(11, "sandbox-key", []*singleagent.SkillReference{
		{SkillID: 7, SkillName: "report"},
	}, true)

	out, err := readTool.InvokableRun(context.Background(), `{"skill_name":"report"}`)
	if err != nil {
		t.Fatalf("read_skill returned err: %v", err)
	}
	if !strings.Contains(out, "# Report Skill") {
		t.Fatalf("read_skill should return SKILL.md content, got %q", out)
	}
	if strings.Contains(out, "legacy prompt") {
		t.Fatalf("read_skill returned stale prompt instead of SKILL.md: %q", out)
	}
	if string(fm.files["/skills/report/SKILL.md"]) != "# Report Skill\nUse scripts/run.py." {
		t.Fatalf("SKILL.md not synced into sandbox: %v", fm.files)
	}
	if string(fm.files["/skills/report/scripts/run.py"]) != "print('ok')" {
		t.Fatalf("script file not synced into sandbox: %v", fm.files)
	}
	if string(fm.files["/skills/report/references/spec.md"]) != "details" {
		t.Fatalf("reference file not synced into sandbox: %v", fm.files)
	}
}

func TestSyncBoundSkillsToSandboxInstallsStandardFolderSkill(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	crossskill.SetDefaultSVC(&fakeSkillSvc{skills: map[int64]*entity.Skill{
		8: {
			SkillID: 8,
			SpaceID: 12,
			Name:    "slides",
			Prompt:  "fallback prompt",
			Files: map[string]string{
				"SKILL.md":        "# Slides Skill",
				"scripts/make.js": "console.log('slides')",
			},
		},
	}})
	defer crossskill.SetDefaultSVC(nil)

	syncBoundSkillsToSandbox(context.Background(), "sandbox-key", 12, []*singleagent.SkillReference{
		{SkillID: 8, SkillName: "slides"},
	})

	if string(fm.files["/skills/slides/SKILL.md"]) != "# Slides Skill" {
		t.Fatalf("SKILL.md not synced: %v", fm.files)
	}
	if string(fm.files["/skills/slides/scripts/make.js"]) != "console.log('slides')" {
		t.Fatalf("script not synced: %v", fm.files)
	}
	if len(fm.files[skillManifestPath]) == 0 {
		t.Fatalf("manifest should be written after successful sync: %v", fm.files)
	}
}

// TestReadSkillToolReadOnlyModeSkipsInjection verifies that when injectScripts=false
// (a normal agent in read-only skill mode), read_skill still returns the SKILL.md
// instructions but does NOT sync any scripts into the sandbox and does NOT tell the
// model to run them. This is the "read the skill as a document, never execute" path.
func TestReadSkillToolReadOnlyModeSkipsInjection(t *testing.T) {
	fm := &fakeSandboxMgr{}
	crosssandbox.SetDefaultSVC(fm)
	defer crosssandbox.SetDefaultSVC(nil)

	crossskill.SetDefaultSVC(&fakeSkillSvc{skills: map[int64]*entity.Skill{
		7: {
			SkillID: 7,
			SpaceID: 11,
			Name:    "report",
			Files: map[string]string{
				"SKILL.md":       "# Report Skill\nUse scripts/run.py.",
				"scripts/run.py": "print('ok')",
			},
		},
	}})
	defer crossskill.SetDefaultSVC(nil)

	readTool := newReadSkillTool(11, "sandbox-key", []*singleagent.SkillReference{
		{SkillID: 7, SkillName: "report"},
	}, false)

	out, err := readTool.InvokableRun(context.Background(), `{"skill_name":"report"}`)
	if err != nil {
		t.Fatalf("read_skill returned err: %v", err)
	}
	if !strings.Contains(out, "# Report Skill") {
		t.Fatalf("read-only read_skill must still return SKILL.md content, got %q", out)
	}
	if strings.Contains(out, "run them with run_bash") {
		t.Fatalf("read-only read_skill must NOT instruct the model to run scripts: %q", out)
	}
	if _, ok := fm.files["/skills/report/scripts/run.py"]; ok {
		t.Fatalf("read-only read_skill must NOT sync scripts into the sandbox: %v", fm.files)
	}
}
