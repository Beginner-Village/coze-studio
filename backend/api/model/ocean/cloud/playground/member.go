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

package playground

// MemberInfo represents a space member
type MemberInfo struct {
	UserID        string        `json:"user_id" form:"user_id"`
	Name          string        `json:"name" form:"name"`
	UserName      string        `json:"user_name" form:"user_name"`
	IconURL       string        `json:"icon_url" form:"icon_url"`
	SpaceRoleType SpaceRoleType `json:"space_role_type" form:"space_role_type"`
	JoinDate      string        `json:"join_date" form:"join_date"`
}

// SpaceMemberDetailV2Request represents the request for fetching space members
type SpaceMemberDetailV2Request struct {
	SpaceID       string        `json:"space_id" form:"space_id"`
	SearchWord    string        `json:"search_word" form:"search_word"`
	SpaceRoleType SpaceRoleType `json:"space_role_type" form:"space_role_type"`
	Page          int           `json:"page" form:"page"`
	Size          int           `json:"size" form:"size"`
}

// SpaceMemberDetailV2Response represents the response for fetching space members
type SpaceMemberDetailV2Response struct {
	Code int64 `json:"code"`
	Msg  string `json:"msg"`
	Data *SpaceMemberDetailData `json:"data"`
}

// SpaceMemberDetailData represents the data for space member details
type SpaceMemberDetailData struct {
	MemberInfoList []MemberInfo  `json:"member_info_list"`
	Total          int           `json:"total"`
	SpaceRoleType  SpaceRoleType `json:"space_role_type"` // Current user's role
}

// AddBotSpaceMemberV2Request represents the request for adding space members
type AddBotSpaceMemberV2Request struct {
	SpaceID        string       `json:"space_id" form:"space_id"`
	MemberInfoList []MemberInfo `json:"member_info_list" form:"member_info_list"`
}

// AddBotSpaceMemberV2Response represents the response for adding space members
type AddBotSpaceMemberV2Response struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// RemoveSpaceMemberV2Request represents the request for removing a space member
type RemoveSpaceMemberV2Request struct {
	SpaceID      string `json:"space_id" form:"space_id"`
	RemoveUserID string `json:"remove_user_id" form:"remove_user_id"`
}

// RemoveSpaceMemberV2Response represents the response for removing a space member
type RemoveSpaceMemberV2Response struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// UpdateSpaceMemberV2Request represents the request for updating a space member
type UpdateSpaceMemberV2Request struct {
	SpaceID       string        `json:"space_id" form:"space_id"`
	UserID        string        `json:"user_id" form:"user_id"`
	SpaceRoleType SpaceRoleType `json:"space_role_type" form:"space_role_type"`
}

// UpdateSpaceMemberV2Response represents the response for updating a space member
type UpdateSpaceMemberV2Response struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
}

// SearchMemberV2Request represents the request for searching members
type SearchMemberV2Request struct {
	SearchList []string `json:"search_list" form:"search_list"`
}

// SearchMemberV2Response represents the response for searching members
type SearchMemberV2Response struct {
	Code           int64        `json:"code"`
	Msg            string       `json:"msg"`
	MemberInfoList []MemberInfo `json:"member_info_list"`
}