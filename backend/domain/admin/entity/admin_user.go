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

const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
)

type AdminUser struct {
	ID        uint64  `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	UserID    uint64  `gorm:"not null;comment:关联的用户ID" json:"user_id"`
	Role      string  `gorm:"size:32;not null;default:'admin';comment:角色: super_admin/admin" json:"role"`
	CreatedAt uint64  `gorm:"not null;default:0;comment:创建时间（毫秒）" json:"created_at"`
	CreatedBy uint64  `gorm:"not null;default:0;comment:创建者ID" json:"created_by"`
	UpdatedAt uint64  `gorm:"not null;default:0;comment:更新时间（毫秒）" json:"updated_at"`
	DeletedAt *uint64 `gorm:"comment:删除时间（毫秒）" json:"deleted_at"`
}

func (AdminUser) TableName() string {
	return "admin_user"
}

func (a *AdminUser) IsSuperAdmin() bool {
	return a.Role == RoleSuperAdmin
}
