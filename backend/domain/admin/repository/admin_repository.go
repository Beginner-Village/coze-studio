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

package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/admin/entity"
)

type AdminRepository interface {
	// 查询是否已初始化（是否存在超级管理员）
	IsInitialized(ctx context.Context) (bool, error)

	// 查询用户是否是管理员
	IsAdmin(ctx context.Context, userID uint64) (bool, error)

	// 根据用户ID获取管理员信息
	GetByUserID(ctx context.Context, userID uint64) (*entity.AdminUser, error)

	// 获取所有管理员列表
	ListAdmins(ctx context.Context) ([]*entity.AdminUser, error)

	// 创建管理员
	Create(ctx context.Context, admin *entity.AdminUser) error

	// 删除管理员（软删除）
	Delete(ctx context.Context, userID uint64) error

	// 统计超级管理员数量
	CountSuperAdmins(ctx context.Context) (int64, error)
}

type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) IsInitialized(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.AdminUser{}).
		Where("role = ? AND deleted_at IS NULL", entity.RoleSuperAdmin).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check initialization: %w", err)
	}
	return count > 0, nil
}

func (r *adminRepository) IsAdmin(ctx context.Context, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.AdminUser{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check admin: %w", err)
	}
	return count > 0, nil
}

func (r *adminRepository) GetByUserID(ctx context.Context, userID uint64) (*entity.AdminUser, error) {
	var admin entity.AdminUser
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&admin).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get admin by user id: %w", err)
	}
	return &admin, nil
}

func (r *adminRepository) ListAdmins(ctx context.Context) ([]*entity.AdminUser, error) {
	var admins []*entity.AdminUser
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at ASC").
		Find(&admins).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list admins: %w", err)
	}
	return admins, nil
}

func (r *adminRepository) Create(ctx context.Context, admin *entity.AdminUser) error {
	now := uint64(time.Now().UnixMilli())
	if admin.CreatedAt == 0 {
		admin.CreatedAt = now
	}
	admin.UpdatedAt = now

	err := r.db.WithContext(ctx).Create(admin).Error
	if err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}
	return nil
}

func (r *adminRepository) Delete(ctx context.Context, userID uint64) error {
	now := uint64(time.Now().UnixMilli())
	err := r.db.WithContext(ctx).
		Model(&entity.AdminUser{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		}).Error
	if err != nil {
		return fmt.Errorf("failed to delete admin: %w", err)
	}
	return nil
}

func (r *adminRepository) CountSuperAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.AdminUser{}).
		Where("role = ? AND deleted_at IS NULL", entity.RoleSuperAdmin).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count super admins: %w", err)
	}
	return count, nil
}
