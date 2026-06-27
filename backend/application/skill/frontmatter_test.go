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
)

func TestParseSkillMetadataFromStandardSkillFrontmatter(t *testing.T) {
	md := `---
name: doc-tools
description: Create and inspect docs
version: 1.2.3
category: productivity
tags: [documents, office]
platforms:
  - linux
  - macos
---
# Doc Tools
`

	got := parseSkillMetadata(md)

	assert.Equal(t, "productivity", got.Category)
	assert.Equal(t, "1.2.3", got.Version)
	assert.Equal(t, []string{"documents", "office"}, got.Tags)
	assert.Equal(t, []string{"linux", "macos"}, got.Platforms)
}
