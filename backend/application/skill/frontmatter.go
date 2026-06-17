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

package skill

import "strings"

// parseSkillFrontmatter 解析 SKILL.md 的 YAML frontmatter(对齐 Anthropic / LangChain Agent Skills)。
// 形如:
//
//	---
//	name: docx
//	description: "Use this skill when ..."
//	---
//	<正文>
//
// 返回 name、description、以及去掉 frontmatter 后的正文。无 frontmatter 时返回空 name/desc + 原文。
func parseSkillFrontmatter(skillMd string) (name, description, body string) {
	s := strings.ReplaceAll(skillMd, "\r\n", "\n")
	if !strings.HasPrefix(strings.TrimLeft(s, " \t\n"), "---") {
		return "", "", skillMd
	}
	// 定位首尾 ---
	trimmed := strings.TrimLeft(s, " \t\n")
	rest := strings.TrimPrefix(trimmed, "---")
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", "", skillMd
	}
	front := rest[:end]
	body = strings.TrimPrefix(rest[end+len("\n---"):], "\n")

	for _, line := range strings.Split(front, "\n") {
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)
		switch strings.ToLower(key) {
		case "name":
			name = val
		case "description":
			description = val
		}
	}
	return name, description, body
}
