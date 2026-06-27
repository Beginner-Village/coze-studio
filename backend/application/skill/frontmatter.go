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

import (
	"strings"

	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
	"gopkg.in/yaml.v3"
)

type skillFrontmatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version"`
	Category    string   `yaml:"category"`
	Tags        []string `yaml:"tags"`
	Platforms   []string `yaml:"platforms"`
}

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
	front, body, ok := splitSkillFrontmatter(skillMd)
	if !ok {
		return "", "", skillMd
	}

	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
		return "", "", body
	}

	return strings.TrimSpace(fm.Name), strings.TrimSpace(fm.Description), body
}

func ParseSkillMetadata(skillMd string) entity.SkillMetadata {
	return parseSkillMetadata(skillMd)
}

func parseSkillMetadata(skillMd string) entity.SkillMetadata {
	front, _, ok := splitSkillFrontmatter(skillMd)
	if !ok {
		return entity.SkillMetadata{}
	}

	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
		return entity.SkillMetadata{}
	}

	return entity.SkillMetadata{
		Version:   strings.TrimSpace(fm.Version),
		Category:  strings.TrimSpace(fm.Category),
		Tags:      compactStrings(fm.Tags),
		Platforms: compactStrings(fm.Platforms),
	}
}

func splitSkillFrontmatter(skillMd string) (front, body string, ok bool) {
	s := strings.ReplaceAll(skillMd, "\r\n", "\n")
	if !strings.HasPrefix(strings.TrimLeft(s, " \t\n"), "---") {
		return "", skillMd, false
	}
	// 定位首尾 ---
	trimmed := strings.TrimLeft(s, " \t\n")
	rest := strings.TrimPrefix(trimmed, "---")
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", skillMd, false
	}
	front = rest[:end]
	body = strings.TrimPrefix(rest[end+len("\n---"):], "\n")
	return front, body, true
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
