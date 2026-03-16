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

package entity

// Skill is the domain entity for skill.
type Skill struct {
	SkillID     int64
	SpaceID     int64
	Name        string
	Description string
	Prompt      string
	IconURI     string
	CreatorID   int64
	Status      int8
	CreatedAt   int64
	UpdatedAt   int64
}

// SkillReference is a lightweight reference for Bot binding.
type SkillReference struct {
	SkillID          int64
	SkillName        string
	SkillDescription string
}

// ListRequest is the request for listing skills.
type ListRequest struct {
	SpaceID  int64
	Page     int32
	PageSize int32
	Keyword  string
}

// ListResponse is the response for listing skills.
type ListResponse struct {
	Skills []*Skill
	Total  int32
}
