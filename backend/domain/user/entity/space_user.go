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

package entity

// SpaceUser represents a user's membership in a space
type SpaceUser struct {
	ID        int64 `json:"id"`
	SpaceID   int64 `json:"space_id"`
	UserID    int64 `json:"user_id"`
	RoleType  int32 `json:"role_type"` // 1: owner, 2: admin, 3: member
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}