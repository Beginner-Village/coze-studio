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

package dal

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/coze-dev/coze-studio/backend/domain/user/internal/dal/model"
	"github.com/coze-dev/coze-studio/backend/domain/user/internal/dal/query"
)

func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{
		query: query.Use(db),
	}
}

type UserDAO struct {
	query *query.Query
}

func (dao *UserDAO) GetUsersByEmail(ctx context.Context, email string) (*model.User, bool, error) {
	user, err := dao.query.User.WithContext(ctx).Where(dao.query.User.Email.Eq(email)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	return user, true, err
}

func (dao *UserDAO) UpdateSessionKey(ctx context.Context, userID int64, sessionKey string) error {
	_, err := dao.query.User.WithContext(ctx).Where(
		dao.query.User.ID.Eq(userID),
	).Updates(map[string]interface{}{
		"session_key": sessionKey,
		"updated_at":  time.Now().UnixMilli(),
	})
	return err
}

func (dao *UserDAO) ClearSessionKey(ctx context.Context, userID int64) error {
	_, err := dao.query.User.WithContext(ctx).
		Where(
			dao.query.User.ID.Eq(userID),
		).
		UpdateColumn(dao.query.User.SessionKey, "")

	return err
}

func (dao *UserDAO) UpdatePassword(ctx context.Context, email, password string) error {
	_, err := dao.query.User.WithContext(ctx).Where(
		dao.query.User.Email.Eq(email),
	).Updates(map[string]interface{}{
		"password":    password,
		"session_key": "", // clear session key
		"updated_at":  time.Now().UnixMilli(),
	})
	return err
}

func (dao *UserDAO) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	return dao.query.User.WithContext(ctx).Where(
		dao.query.User.ID.Eq(userID),
	).First()
}

func (dao *UserDAO) UpdateAvatar(ctx context.Context, userID int64, iconURI string) error {
	_, err := dao.query.User.WithContext(ctx).Where(
		dao.query.User.ID.Eq(userID),
	).Updates(map[string]interface{}{
		"icon_uri":   iconURI,
		"updated_at": time.Now().UnixMilli(),
	})
	return err
}

func (dao *UserDAO) CheckUniqueNameExist(ctx context.Context, uniqueName string) (bool, error) {
	_, err := dao.query.User.WithContext(ctx).Select(dao.query.User.ID).Where(
		dao.query.User.UniqueName.Eq(uniqueName),
	).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (dao *UserDAO) UpdateProfile(ctx context.Context, userID int64, updates map[string]interface{}) error {
	if _, ok := updates["updated_at"]; !ok {
		updates["updated_at"] = time.Now().UnixMilli()
	}

	_, err := dao.query.User.WithContext(ctx).Where(
		dao.query.User.ID.Eq(userID),
	).Updates(updates)
	return err
}

func (dao *UserDAO) CheckEmailExist(ctx context.Context, email string) (bool, error) {
	_, exist, err := dao.GetUsersByEmail(ctx, email)
	if !exist {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

// CreateUser Create a new user
func (dao *UserDAO) CreateUser(ctx context.Context, user *model.User) error {
	return dao.query.User.WithContext(ctx).Create(user)
}

// GetUserBySessionKey Query users based on session key
func (dao *UserDAO) GetUserBySessionKey(ctx context.Context, sessionKey string) (*model.User, bool, error) {
	sm, err := dao.query.User.WithContext(ctx).Where(
		dao.query.User.SessionKey.Eq(sessionKey),
	).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return sm, true, nil
}

// GetUsersByIDs Query user information in batches
func (dao *UserDAO) GetUsersByIDs(ctx context.Context, userIDs []int64) ([]*model.User, error) {
	return dao.query.User.WithContext(ctx).Where(
		dao.query.User.ID.In(userIDs...),
	).Find()
}

// GetSpaceUser gets a space user by space ID and user ID
func (dao *UserDAO) GetSpaceUser(ctx context.Context, spaceID, userID int64) (*model.SpaceUser, bool, error) {
	spaceUser, err := dao.query.SpaceUser.WithContext(ctx).Where(
		dao.query.SpaceUser.SpaceID.Eq(spaceID),
		dao.query.SpaceUser.UserID.Eq(userID),
	).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return spaceUser, true, nil
}

// GetSpaceMembers gets space members with pagination and filtering
func (dao *UserDAO) GetSpaceMembers(ctx context.Context, spaceID int64, searchWord string, roleType int32, offset, limit int) ([]*model.SpaceUser, int64, error) {
	q := dao.query.SpaceUser.WithContext(ctx).Where(dao.query.SpaceUser.SpaceID.Eq(spaceID))
	
	// Add role filter if specified
	if roleType > 0 {
		q = q.Where(dao.query.SpaceUser.RoleType.Eq(roleType))
	}
	
	// Add search filter if specified
	if searchWord != "" {
		// Join with user table to search by name
		var userIDs []int64
		err := dao.query.User.WithContext(ctx).
			Where(dao.query.User.Name.Like("%" + searchWord + "%")).
			Pluck(dao.query.User.ID, &userIDs)
		if err != nil {
			return nil, 0, err
		}
		if len(userIDs) > 0 {
			q = q.Where(dao.query.SpaceUser.UserID.In(userIDs...))
		} else {
			// No matching users, return empty result
			return []*model.SpaceUser{}, 0, nil
		}
	}
	
	// Get total count
	count, err := q.Count()
	if err != nil {
		return nil, 0, err
	}
	
	// Get paginated results
	members, err := q.Offset(offset).Limit(limit).Find()
	if err != nil {
		return nil, 0, err
	}
	
	return members, count, nil
}

// CreateSpaceUser creates a new space user
func (dao *UserDAO) CreateSpaceUser(ctx context.Context, spaceUser *model.SpaceUser) error {
	return dao.query.SpaceUser.WithContext(ctx).Create(spaceUser)
}

// DeleteSpaceUser deletes a space user
func (dao *UserDAO) DeleteSpaceUser(ctx context.Context, spaceID, userID int64) error {
	_, err := dao.query.SpaceUser.WithContext(ctx).Where(
		dao.query.SpaceUser.SpaceID.Eq(spaceID),
		dao.query.SpaceUser.UserID.Eq(userID),
	).Delete()
	return err
}

// UpdateSpaceUserRole updates a space user's role
func (dao *UserDAO) UpdateSpaceUserRole(ctx context.Context, spaceID, userID int64, roleType int32) error {
	_, err := dao.query.SpaceUser.WithContext(ctx).Where(
		dao.query.SpaceUser.SpaceID.Eq(spaceID),
		dao.query.SpaceUser.UserID.Eq(userID),
	).Update(dao.query.SpaceUser.RoleType, roleType)
	return err
}

// SearchUsers searches for users by name or email
func (dao *UserDAO) SearchUsers(ctx context.Context, searchList []string) ([]*model.User, error) {
	if len(searchList) == 0 {
		return []*model.User{}, nil
	}
	
	// For now, just search by name with the first search term
	// This is a simplified implementation
	searchPattern := "%" + searchList[0] + "%"
	
	// Search in name field only for now
	return dao.query.User.WithContext(ctx).Where(
		dao.query.User.Name.Like(searchPattern),
	).Find()
}
