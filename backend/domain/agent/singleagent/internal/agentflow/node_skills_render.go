/*
 * Copyright 2025 coze-dev Authors
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

	"github.com/coze-dev/coze-studio/backend/api/model/crossdomain/singleagent"
)

// skillsRender renders the available skills section for the system prompt.
// Only skill names and descriptions are included (progressive disclosure).
type skillsRender struct {
	skillInfoList []*singleagent.SkillReference
}

func newSkillsRender(skillInfoList []*singleagent.SkillReference) *skillsRender {
	return &skillsRender{
		skillInfoList: skillInfoList,
	}
}

// RenderSkills renders the available skills list for system prompt injection.
func (s *skillsRender) RenderSkills(_ context.Context, _ *AgentRequest) (string, error) {
	if len(s.skillInfoList) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("The following skills are available. Call read_skill(skill_name) to get detailed instructions:\n\n")

	for _, ref := range s.skillInfoList {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", ref.SkillName, ref.SkillDescription))
	}

	return sb.String(), nil
}
