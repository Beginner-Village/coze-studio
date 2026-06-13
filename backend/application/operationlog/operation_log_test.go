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

package operationlog

import (
	"context"
	"errors"
	"testing"

	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
	userentity "github.com/ynet-dev/ynet-studio/backend/domain/user/entity"
	usersvc "github.com/ynet-dev/ynet-studio/backend/domain/user/service"
	"github.com/ynet-dev/ynet-studio/backend/pkg/ctxcache"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/types/consts"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// fakeUserSVC embeds the full usersvc.User interface (nil) so only the methods
// we override are usable; any other call would nil-panic, which is fine here.
type fakeUserSVC struct {
	usersvc.User
	canManage bool
	profiles  []*userentity.User
}

func (f *fakeUserSVC) CheckMemberPermission(_ context.Context, _, _ int64) (bool, int32, bool, bool, error) {
	return true, 0, false, f.canManage, nil
}

func (f *fakeUserSVC) MGetUserProfiles(_ context.Context, _ []int64) ([]*userentity.User, error) {
	return f.profiles, nil
}

// fakeDomainSVC is a minimal entity.OperationLog domain service.
type fakeDomainSVC struct {
	records []*entity.OperationLog
}

func (f *fakeDomainSVC) Collect(_ *entity.Event) {}
func (f *fakeDomainSVC) List(_ context.Context, _ *entity.ListFilter) ([]*entity.OperationLog, int64, error) {
	return f.records, int64(len(f.records)), nil
}
func (f *fakeDomainSVC) Start(_ context.Context) {}
func (f *fakeDomainSVC) DroppedCount() int64    { return 0 }

func ctxWithUID(uid int64) context.Context {
	ctx := ctxcache.Init(context.Background())
	ctxcache.Store(ctx, consts.SessionDataKeyInCtx, &userentity.Session{UserID: uid})
	return ctx
}

func TestListRejectsNonAdmin(t *testing.T) {
	svc := &OperationLogApplicationService{
		DomainSVC: &fakeDomainSVC{},
		userSVC:   &fakeUserSVC{canManage: false},
	}
	_, _, err := svc.ListOperationLogs(ctxWithUID(42), &entity.ListFilter{SpaceID: 1})
	if err == nil {
		t.Fatalf("expected permission error, got nil")
	}
	var sErr errorx.StatusError
	if !errors.As(err, &sErr) || sErr.Code() != int32(errno.ErrOperationLogPermissionCode) {
		t.Fatalf("expected ErrOperationLogPermissionCode (%d), got %v", errno.ErrOperationLogPermissionCode, err)
	}
}

func TestListRejectsAnonymous(t *testing.T) {
	svc := &OperationLogApplicationService{
		DomainSVC: &fakeDomainSVC{},
		userSVC:   &fakeUserSVC{canManage: true},
	}
	_, _, err := svc.ListOperationLogs(context.Background(), &entity.ListFilter{SpaceID: 1})
	if err == nil {
		t.Fatalf("expected permission error for anonymous caller, got nil")
	}
}

func TestListReturnsErrorWhenDisabled(t *testing.T) {
	// Feature disabled: Init never ran, so DomainSVC and userSVC are nil.
	svc := &OperationLogApplicationService{}
	_, _, err := svc.ListOperationLogs(context.Background(), &entity.ListFilter{SpaceID: 1})
	if err == nil {
		t.Fatalf("expected error when service is disabled, got nil")
	}
}

func TestListResolvesOperatorNames(t *testing.T) {
	svc := &OperationLogApplicationService{
		DomainSVC: &fakeDomainSVC{records: []*entity.OperationLog{
			{ID: 1, OperatorID: 100, Action: "create"},
			{ID: 2, OperatorID: 100, Action: "update"},
			{ID: 3, OperatorID: 200, Action: "delete"},
			{ID: 4, OperatorID: 0, Action: "system"},
		}},
		userSVC: &fakeUserSVC{
			canManage: true,
			profiles: []*userentity.User{
				{UserID: 100, Name: "alice"},
				{UserID: 200, Name: "bob"},
			},
		},
	}
	items, total, err := svc.ListOperationLogs(ctxWithUID(1), &entity.ListFilter{SpaceID: 1})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if total != 4 {
		t.Fatalf("want total 4, got %d", total)
	}
	if items[0].OperatorName != "alice" || items[1].OperatorName != "alice" {
		t.Fatalf("operator 100 should be alice, got %q %q", items[0].OperatorName, items[1].OperatorName)
	}
	if items[2].OperatorName != "bob" {
		t.Fatalf("operator 200 should be bob, got %q", items[2].OperatorName)
	}
	if items[3].OperatorName != "" {
		t.Fatalf("operator 0 should be empty, got %q", items[3].OperatorName)
	}
}
