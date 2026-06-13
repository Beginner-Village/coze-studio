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

package dal

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/internal/dal/model"
)

type fakeIDGen struct{ next int64 }

func (f *fakeIDGen) GenID(ctx context.Context) (int64, error) {
	f.next++
	return f.next, nil
}

func (f *fakeIDGen) GenMultiIDs(ctx context.Context, counts int) ([]int64, error) {
	ids := make([]int64, counts)
	for i := 0; i < counts; i++ {
		f.next++
		ids[i] = f.next
	}
	return ids, nil
}

func ptr[T any](v T) *T { return &v }

func newTestDAO(t *testing.T) *OperationLogDAO {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.OperationLog{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return NewOperationLogDAO(db, &fakeIDGen{})
}

// TestListKeywordScopedToSpace 验证 keyword 过滤不会越过 space_id 边界。
// 修复前 SQL 为 `space_id=? AND ... OR resource_name LIKE ?`,会返回其它空间记录。
func TestListKeywordScopedToSpace(t *testing.T) {
	dao := newTestDAO(t)
	ctx := context.Background()

	if err := dao.BatchCreate(ctx, []*entity.OperationLog{
		{SpaceID: 1, ResourceName: "foo-resource-a", Action: "create", CreatedAt: 1},
		{SpaceID: 2, ResourceName: "foo-resource-b", Action: "create", CreatedAt: 2},
	}); err != nil {
		t.Fatalf("batch create: %v", err)
	}

	out, total, err := dao.List(ctx, &entity.ListFilter{
		SpaceID:  1,
		Keyword:  ptr("foo"),
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("want total 1, got %d", total)
	}
	if len(out) != 1 {
		t.Fatalf("want 1 record, got %d", len(out))
	}
	if out[0].SpaceID != 1 {
		t.Fatalf("want record from space 1, got space %d", out[0].SpaceID)
	}
}
