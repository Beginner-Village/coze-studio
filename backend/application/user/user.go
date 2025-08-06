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

package user

import (
	"context"
	"net/mail"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/coze-dev/coze-studio/backend/api/model/ocean/cloud/developer_api"
	"github.com/coze-dev/coze-studio/backend/api/model/ocean/cloud/playground"
	"github.com/coze-dev/coze-studio/backend/api/model/passport"
	"github.com/coze-dev/coze-studio/backend/application/base/ctxutil"
	"github.com/coze-dev/coze-studio/backend/domain/user/entity"
	user "github.com/coze-dev/coze-studio/backend/domain/user/service"
	"github.com/coze-dev/coze-studio/backend/infra/contract/storage"
	"github.com/coze-dev/coze-studio/backend/pkg/errorx"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/ptr"
	langSlices "github.com/coze-dev/coze-studio/backend/pkg/lang/slices"
	"github.com/coze-dev/coze-studio/backend/types/consts"
	"github.com/coze-dev/coze-studio/backend/types/errno"
)

var UserApplicationSVC = &UserApplicationService{}

type UserApplicationService struct {
	oss       storage.Storage
	DomainSVC user.User
}

// Add a simple email verification function
func isValidEmail(email string) bool {
	// If the email string is not in the correct format, it will return an error.
	_, err := mail.ParseAddress(email)
	return err == nil
}

func (u *UserApplicationService) PassportWebEmailRegisterV2(ctx context.Context, locale string, req *passport.PassportWebEmailRegisterV2PostRequest) (
	resp *passport.PassportWebEmailRegisterV2PostResponse, sessionKey string, err error,
) {
	// Verify that the email format is legitimate
	if !isValidEmail(req.GetEmail()) {
		return nil, "", errorx.New(errno.ErrUserInvalidParamCode, errorx.KV("msg", "Invalid email"))
	}

	// Allow Register Checker
	if !u.allowRegisterChecker(req.GetEmail()) {
		return nil, "", errorx.New(errno.ErrNotAllowedRegisterCode)
	}

	userInfo, err := u.DomainSVC.Create(ctx, &user.CreateUserRequest{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),

		Locale: locale,
	})
	if err != nil {
		return nil, "", err
	}

	userInfo, err = u.DomainSVC.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, "", err
	}

	return &passport.PassportWebEmailRegisterV2PostResponse{
		Data: userDo2PassportTo(userInfo),
		Code: 0,
	}, userInfo.SessionKey, nil
}

func (u *UserApplicationService) allowRegisterChecker(email string) bool {
	disableUserRegistration := os.Getenv(consts.DisableUserRegistration)
	if strings.ToLower(disableUserRegistration) != "true" {
		return true
	}

	allowedEmails := os.Getenv(consts.AllowRegistrationEmail)
	if allowedEmails == "" {
		return false
	}

	return slices.Contains(strings.Split(allowedEmails, ","), strings.ToLower(email))
}

// PassportWebLogoutGet handle user logout requests
func (u *UserApplicationService) PassportWebLogoutGet(ctx context.Context, req *passport.PassportWebLogoutGetRequest) (
	resp *passport.PassportWebLogoutGetResponse, err error,
) {
	uid := ctxutil.MustGetUIDFromCtx(ctx)

	err = u.DomainSVC.Logout(ctx, uid)
	if err != nil {
		return nil, err
	}

	return &passport.PassportWebLogoutGetResponse{
		Code: 0,
	}, nil
}

// PassportWebEmailLoginPost handle user email login requests
func (u *UserApplicationService) PassportWebEmailLoginPost(ctx context.Context, req *passport.PassportWebEmailLoginPostRequest) (
	resp *passport.PassportWebEmailLoginPostResponse, sessionKey string, err error,
) {
	userInfo, err := u.DomainSVC.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, "", err
	}

	return &passport.PassportWebEmailLoginPostResponse{
		Data: userDo2PassportTo(userInfo),
		Code: 0,
	}, userInfo.SessionKey, nil
}

func (u *UserApplicationService) PassportWebEmailPasswordResetGet(ctx context.Context, req *passport.PassportWebEmailPasswordResetGetRequest) (
	resp *passport.PassportWebEmailPasswordResetGetResponse, err error,
) {
	err = u.DomainSVC.ResetPassword(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	return &passport.PassportWebEmailPasswordResetGetResponse{
		Code: 0,
	}, nil
}

func (u *UserApplicationService) PassportAccountInfoV2(ctx context.Context, req *passport.PassportAccountInfoV2Request) (
	resp *passport.PassportAccountInfoV2Response, err error,
) {
	userID := ctxutil.MustGetUIDFromCtx(ctx)

	userInfo, err := u.DomainSVC.GetUserInfo(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &passport.PassportAccountInfoV2Response{
		Data: userDo2PassportTo(userInfo),
		Code: 0,
	}, nil
}

// UserUpdateAvatar Update user avatar
func (u *UserApplicationService) UserUpdateAvatar(ctx context.Context, mimeType string, req *passport.UserUpdateAvatarRequest) (
	resp *passport.UserUpdateAvatarResponse, err error,
) {
	// Get file suffix by MIME type
	var ext string
	switch mimeType {
	case "image/jpeg", "image/jpg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	case "image/gif":
		ext = "gif"
	case "image/webp":
		ext = "webp"
	default:
		return nil, errorx.WrapByCode(err, errno.ErrUserInvalidParamCode,
			errorx.KV("msg", "unsupported image type"))
	}

	uid := ctxutil.MustGetUIDFromCtx(ctx)

	url, err := u.DomainSVC.UpdateAvatar(ctx, uid, ext, req.GetAvatar())
	if err != nil {
		return nil, err
	}

	return &passport.UserUpdateAvatarResponse{
		Data: &passport.UserUpdateAvatarResponseData{
			WebURI: url,
		},
		Code: 0,
	}, nil
}

// UserUpdateProfile Update user profile
func (u *UserApplicationService) UserUpdateProfile(ctx context.Context, req *passport.UserUpdateProfileRequest) (
	resp *passport.UserUpdateProfileResponse, err error,
) {
	userID := ctxutil.MustGetUIDFromCtx(ctx)

	err = u.DomainSVC.UpdateProfile(ctx, &user.UpdateProfileRequest{
		UserID:      userID,
		Name:        req.Name,
		UniqueName:  req.UserUniqueName,
		Description: req.Description,
		Locale:      req.Locale,
	})
	if err != nil {
		return nil, err
	}

	return &passport.UserUpdateProfileResponse{
		Code: 0,
	}, nil
}

func (u *UserApplicationService) GetSpaceListV2(ctx context.Context, req *playground.GetSpaceListV2Request) (
	resp *playground.GetSpaceListV2Response, err error,
) {
	uid := ctxutil.MustGetUIDFromCtx(ctx)

	spaces, err := u.DomainSVC.GetUserSpaceList(ctx, uid)
	if err != nil {
		return nil, err
	}

	botSpaces := langSlices.Transform(spaces, func(space *entity.Space) *playground.BotSpaceV2 {
		return &playground.BotSpaceV2{
			ID:            space.ID,
			Name:          space.Name,
			Description:   space.Description,
			SpaceType:     playground.SpaceType(space.SpaceType),
			IconURL:       space.IconURL,
			SpaceRoleType: playground.SpaceRoleType(space.RoleType),
		}
	})

	return &playground.GetSpaceListV2Response{
		Data: &playground.SpaceInfo{
			BotSpaceList:          botSpaces,
			HasPersonalSpace:      true,
			TeamSpaceNum:          0,
			RecentlyUsedSpaceList: botSpaces,
			Total:                 ptr.Of(int32(len(botSpaces))),
			HasMore:               ptr.Of(false),
		},
		Code: 0,
	}, nil
}

func (u *UserApplicationService) CreateSpace(ctx context.Context, req *playground.CreateSpaceRequest) (
	resp *playground.CreateSpaceResponse, err error,
) {
	uid := ctxutil.MustGetUIDFromCtx(ctx)

	// 创建空间
	space, err := u.DomainSVC.CreateUserSpace(ctx, &user.CreateUserSpaceRequest{
		UserID:      uid,
		Name:        req.GetName(),
		Description: req.GetDescription(),
		IconURI:     req.GetIconURI(),
		SpaceType:   req.GetSpaceType(),
		SpaceMode:   req.GetSpaceMode(),
	})
	if err != nil {
		return nil, err
	}

	// 转换为API响应格式
	botSpace := &playground.BotSpaceV2{
		ID:          space.ID,
		Name:        space.Name,
		Description: space.Description,
		IconURL:     space.IconURL,
	}

	return &playground.CreateSpaceResponse{
		Data: botSpace,
		Code: 0,
	}, nil
}

func (u *UserApplicationService) MGetUserBasicInfo(ctx context.Context, req *playground.MGetUserBasicInfoRequest) (
	resp *playground.MGetUserBasicInfoResponse, err error,
) {
	userIDs, err := langSlices.TransformWithErrorCheck(req.GetUserIds(), func(s string) (int64, error) {
		return strconv.ParseInt(s, 10, 64)
	})
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrUserInvalidParamCode, errorx.KV("msg", "invalid user id"))
	}

	userInfos, err := u.DomainSVC.MGetUserProfiles(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	return &playground.MGetUserBasicInfoResponse{
		UserBasicInfoMap: langSlices.ToMap(userInfos, func(userInfo *entity.User) (string, *playground.UserBasicInfo) {
			return strconv.FormatInt(userInfo.UserID, 10), userDo2PlaygroundTo(userInfo)
		}),
		Code: 0,
	}, nil
}

func (u *UserApplicationService) UpdateUserProfileCheck(ctx context.Context, req *developer_api.UpdateUserProfileCheckRequest) (resp *developer_api.UpdateUserProfileCheckResponse, err error) {
	if req.GetUserUniqueName() == "" {
		return &developer_api.UpdateUserProfileCheckResponse{
			Code: 0,
			Msg:  "no content to update",
		}, nil
	}

	validateResp, err := u.DomainSVC.ValidateProfileUpdate(ctx, &user.ValidateProfileUpdateRequest{
		UniqueName: req.UserUniqueName,
	})
	if err != nil {
		return nil, err
	}

	return &developer_api.UpdateUserProfileCheckResponse{
		Code: int64(validateResp.Code),
		Msg:  validateResp.Msg,
	}, nil
}

func (u *UserApplicationService) ValidateSession(ctx context.Context, sessionKey string) (*entity.Session, error) {
	session, exist, err := u.DomainSVC.ValidateSession(ctx, sessionKey)
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, errorx.New(errno.ErrUserAuthenticationFailed, errorx.KV("reason", "session not exist"))
	}

	return session, nil
}

func userDo2PassportTo(userDo *entity.User) *passport.User {
	var locale *string
	if userDo.Locale != "" {
		locale = ptr.Of(userDo.Locale)
	}

	return &passport.User{
		UserIDStr:      userDo.UserID,
		Name:           userDo.Name,
		ScreenName:     ptr.Of(userDo.Name),
		UserUniqueName: userDo.UniqueName,
		Email:          userDo.Email,
		Description:    userDo.Description,
		AvatarURL:      userDo.IconURL,
		AppUserInfo: &passport.AppUserInfo{
			UserUniqueName: userDo.UniqueName,
		},
		Locale: locale,

		UserCreateTime: userDo.CreatedAt / 1000,
	}
}

func userDo2PlaygroundTo(userDo *entity.User) *playground.UserBasicInfo {
	return &playground.UserBasicInfo{
		UserId:         userDo.UserID,
		Username:       userDo.Name,
		UserUniqueName: ptr.Of(userDo.UniqueName),
		UserAvatar:     userDo.IconURL,
		CreateTime:     ptr.Of(userDo.CreatedAt / 1000),
	}
}

// GetSpaceMemberDetail retrieves detailed member information for a space
func (u *UserApplicationService) GetSpaceMemberDetail(ctx context.Context, req *playground.SpaceMemberDetailV2Request) (
	resp *playground.SpaceMemberDetailV2Response, err error,
) {
	uid := ctxutil.MustGetUIDFromCtx(ctx)
	
	// Convert string SpaceID to int64
	spaceID, err := strconv.ParseInt(req.SpaceID, 10, 64)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrUserInvalidParamCode, errorx.KV("msg", "invalid space id format"))
	}

	// Get the current user's role in the space
	currentUserRole, err := u.DomainSVC.GetUserSpaceRole(ctx, uid, spaceID)
	if err != nil {
		return nil, err
	}

	// Get space members with pagination and filtering
	members, total, err := u.DomainSVC.GetSpaceMembers(ctx, spaceID, req.SearchWord, int64(req.SpaceRoleType), req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	// Convert to API response format
	memberInfoList := langSlices.Transform(members, func(member *entity.SpaceUser) playground.MemberInfo {
		userInfo, _ := u.DomainSVC.GetUserInfo(ctx, member.UserID)
		return playground.MemberInfo{
			UserID:        strconv.FormatInt(member.UserID, 10),
			Name:          userInfo.Name,
			UserName:      userInfo.UniqueName,
			IconURL:       userInfo.IconURL,
			SpaceRoleType: playground.SpaceRoleType(member.RoleType),
			JoinDate:      formatTimestamp(member.CreatedAt),
		}
	})

	return &playground.SpaceMemberDetailV2Response{
		Code: 0,
		Data: &playground.SpaceMemberDetailData{
			MemberInfoList: memberInfoList,
			Total:          total,
			SpaceRoleType:  playground.SpaceRoleType(currentUserRole),
		},
	}, nil
}

// AddSpaceMembers adds new members to a space
func (u *UserApplicationService) AddSpaceMembers(ctx context.Context, req *playground.AddBotSpaceMemberV2Request) (
	resp *playground.AddBotSpaceMemberV2Response, err error,
) {
	uid := ctxutil.MustGetUIDFromCtx(ctx)
	
	// Convert string SpaceID to int64
	spaceID, err := strconv.ParseInt(req.SpaceID, 10, 64)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrUserInvalidParamCode, errorx.KV("msg", "invalid space id format"))
	}

	// Check if current user is owner
	currentUserRole, err := u.DomainSVC.GetUserSpaceRole(ctx, uid, spaceID)
	if err != nil {
		return nil, err
	}

	if currentUserRole != int32(playground.SpaceRoleType_Owner) {
		return nil, errorx.New(errno.ErrUserPermissionCode, errorx.KV("msg", "Only owners can add members"))
	}

	// Add members
	for _, member := range req.MemberInfoList {
		userID, err := strconv.ParseInt(member.UserID, 10, 64)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.ErrUserInvalidParamCode, errorx.KV("msg", "invalid user id"))
		}

		err = u.DomainSVC.AddSpaceMember(ctx, spaceID, userID, int32(member.SpaceRoleType))
		if err != nil {
			return nil, err
		}
	}

	return &playground.AddBotSpaceMemberV2Response{
		Code: 0,
	}, nil
}

// RemoveSpaceMember removes a member from a space
func (u *UserApplicationService) RemoveSpaceMember(ctx context.Context, req *playground.RemoveSpaceMemberV2Request) (
	resp *playground.RemoveSpaceMemberV2Response, err error,
) {
	uid := ctxutil.MustGetUIDFromCtx(ctx)
	
	// Convert string SpaceID to int64
	spaceID, err := strconv.ParseInt(req.SpaceID, 10, 64)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrUserInvalidParamCode, errorx.KV("msg", "invalid space id format"))
	}

	// Check if current user is owner
	currentUserRole, err := u.DomainSVC.GetUserSpaceRole(ctx, uid, spaceID)
	if err != nil {
		return nil, err
	}

	if currentUserRole != int32(playground.SpaceRoleType_Owner) {
		return nil, errorx.New(errno.ErrUserPermissionCode, errorx.KV("msg", "Only owners can remove members"))
	}

	// Parse user ID
	removeUserID, err := strconv.ParseInt(req.RemoveUserID, 10, 64)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrUserInvalidParamCode, errorx.KV("msg", "invalid user id"))
	}

	// Remove member
	err = u.DomainSVC.RemoveSpaceMember(ctx, spaceID, removeUserID)
	if err != nil {
		return nil, err
	}

	return &playground.RemoveSpaceMemberV2Response{
		Code: 0,
	}, nil
}

// UpdateSpaceMember updates a member's role in a space
func (u *UserApplicationService) UpdateSpaceMember(ctx context.Context, req *playground.UpdateSpaceMemberV2Request) (
	resp *playground.UpdateSpaceMemberV2Response, err error,
) {
	uid := ctxutil.MustGetUIDFromCtx(ctx)
	
	// Convert string SpaceID to int64
	spaceID, err := strconv.ParseInt(req.SpaceID, 10, 64)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrUserInvalidParamCode, errorx.KV("msg", "invalid space id format"))
	}

	// Check if current user is owner
	currentUserRole, err := u.DomainSVC.GetUserSpaceRole(ctx, uid, spaceID)
	if err != nil {
		return nil, err
	}

	if currentUserRole != int32(playground.SpaceRoleType_Owner) {
		return nil, errorx.New(errno.ErrUserPermissionCode, errorx.KV("msg", "Only owners can update member roles"))
	}

	// Parse user ID
	updateUserID, err := strconv.ParseInt(req.UserID, 10, 64)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.ErrUserInvalidParamCode, errorx.KV("msg", "invalid user id"))
	}

	// Update member role
	err = u.DomainSVC.UpdateSpaceMemberRole(ctx, spaceID, updateUserID, int32(req.SpaceRoleType))
	if err != nil {
		return nil, err
	}

	return &playground.UpdateSpaceMemberV2Response{
		Code: 0,
	}, nil
}

// SearchMembers searches for users to add as members
func (u *UserApplicationService) SearchMembers(ctx context.Context, req *playground.SearchMemberV2Request) (
	resp *playground.SearchMemberV2Response, err error,
) {
	// Search for users
	users, err := u.DomainSVC.SearchUsers(ctx, req.SearchList)
	if err != nil {
		return nil, err
	}

	// Convert to API response format
	memberInfoList := langSlices.Transform(users, func(user *entity.User) playground.MemberInfo {
		return playground.MemberInfo{
			UserID:   strconv.FormatInt(user.UserID, 10),
			Name:     user.Name,
			UserName: user.UniqueName,
			IconURL:  user.IconURL,
		}
	})

	return &playground.SearchMemberV2Response{
		MemberInfoList: memberInfoList,
		Code:           0,
	}, nil
}

func formatTimestamp(ts int64) string {
	if ts == 0 {
		return ""
	}
	t := time.Unix(ts/1000, 0)
	return t.Format("2006-01-02 15:04:05")
}
