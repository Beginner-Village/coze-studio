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
	"strings"
	"testing"
)

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
