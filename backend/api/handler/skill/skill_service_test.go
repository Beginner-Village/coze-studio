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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ynet-dev/ynet-studio/backend/domain/skill/entity"
)

func TestEntityToResponseIncludesStandardSkillMetadata(t *testing.T) {
	got := entityToResponse(&entity.Skill{
		SkillID:     1,
		SpaceID:     2,
		Name:        "doc-tools",
		Description: "Create and inspect docs",
		Files: map[string]string{
			"SKILL.md": "---\nname: doc-tools\ndescription: Create and inspect docs\nversion: 1.2.3\ncategory: productivity\ntags: [documents, office]\nplatforms: [linux, macos]\n---\n# Doc Tools\n",
		},
	})

	assert.Equal(t, "productivity", got.Metadata.Category)
	assert.Equal(t, "1.2.3", got.Metadata.Version)
	assert.Equal(t, []string{"documents", "office"}, got.Metadata.Tags)
	assert.Equal(t, []string{"linux", "macos"}, got.Metadata.Platforms)
}

func TestEntityToResponseIncludesSkillAssetSummary(t *testing.T) {
	got := entityToResponse(&entity.Skill{
		SkillID: 1,
		SpaceID: 2,
		Name:    "brand-kit",
		Files: map[string]string{
			"SKILL.md":         "# Brand Kit\n",
			"assets/logo.png":  "data:image/png;base64,iVBORw0KGgo=",
			"assets/guide.jpg": "data:image/jpeg;base64,/9j/4AAQ",
		},
	})

	assert.Equal(t, int32(2), got.AssetSummary.Count)
	assert.Contains(t, got.AssetSummary.ImagePaths, "assets/logo.png")
	assert.Contains(t, got.AssetSummary.ImagePaths, "assets/guide.jpg")
}
