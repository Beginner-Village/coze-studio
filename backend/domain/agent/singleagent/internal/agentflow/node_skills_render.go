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
	"fmt"
	"strings"

	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
)

// skillsRender renders the available skills section for the system prompt.
// Only skill names and descriptions are included (progressive disclosure).
type skillsRender struct {
	skillInfoList []*singleagent.SkillReference
	// isSuper 为真时按「沙箱技能文件夹」模型渲染(/skills/<name>/SKILL.md);
	// 否则按普通智能体的懒加载 read_skill 模型渲染。
	isSuper bool
}

func newSkillsRender(skillInfoList []*singleagent.SkillReference, isSuper bool) *skillsRender {
	return &skillsRender{
		skillInfoList: skillInfoList,
		isSuper:       isSuper,
	}
}

// RenderSkills renders the available skills list for system prompt injection.
func (s *skillsRender) RenderSkills(_ context.Context, _ *AgentRequest) (string, error) {
	if len(s.skillInfoList) == 0 {
		return "", nil
	}

	var sb strings.Builder
	if s.isSuper {
		// 标准 Agent Skills(渐进式披露):元数据见下;用 read_skill 拿完整说明;脚本在 /skills/ 文件夹。
		sb.WriteString("You have the skills listed below. They follow progressive disclosure — only their name and " +
			"one-line description are shown here.\n\n" +
			"To use a skill, FIRST call the read_skill tool with the skill name to load its full SKILL.md instructions " +
			"(what it does and the exact steps). Do NOT improvise a skill's task before reading it.\n" +
			"Each skill is also installed as a folder at /skills/<name>/ in your sandbox, containing that SKILL.md and any " +
			"ready-to-run scripts. After reading the instructions, run the skill's scripts with run_bash " +
			"(e.g. `python /skills/<name>/<script>`); use list_files on /skills/<name>/ to see what it ships with.\n\n" +
			"Available skills:\n")
		for _, ref := range s.skillInfoList {
			sb.WriteString(fmt.Sprintf("- **%s** — %s  (folder: /skills/%s/)\n",
				ref.SkillName, ref.SkillDescription, ref.SkillName))
		}
	} else {
		sb.WriteString("The following skills are available. Call read_skill(skill_name) to get detailed instructions:\n\n")
		for _, ref := range s.skillInfoList {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", ref.SkillName, ref.SkillDescription))
		}
	}

	return sb.String(), nil
}
