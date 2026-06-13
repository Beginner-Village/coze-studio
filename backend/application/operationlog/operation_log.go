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

	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/application/base/ctxutil"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/entity"
	"github.com/ynet-dev/ynet-studio/backend/domain/operationlog/repository"
	oplogsvc "github.com/ynet-dev/ynet-studio/backend/domain/operationlog/service"
	usersvc "github.com/ynet-dev/ynet-studio/backend/domain/user/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx"
	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"github.com/ynet-dev/ynet-studio/backend/types/errno"
)

// OperationLogApplicationService is the global singleton used by the
// collection middleware (Collect) and the query handler (ListOperationLogs).
type OperationLogApplicationService struct {
	DomainSVC oplogsvc.OperationLog
	userSVC   usersvc.User
}

// OperationLogApplicationSVC is the process-wide singleton. It is safe to call
// Collect on it before Init runs — Collect is a no-op until DomainSVC is set.
var OperationLogApplicationSVC = &OperationLogApplicationService{}

// Init constructs the domain service, starts its background worker + cleanup
// loops, and populates the global singleton. Called from application.Init.
func Init(ctx context.Context, db *gorm.DB, idgenSVC idgen.IDGenerator, userSVC usersvc.User, cfg oplogsvc.Config) *OperationLogApplicationService {
	repo := repository.NewOperationLogRepository(db, idgenSVC)
	domainSVC := oplogsvc.NewOperationLog(repo, cfg)
	domainSVC.Start(ctx)
	OperationLogApplicationSVC.DomainSVC = domainSVC
	OperationLogApplicationSVC.userSVC = userSVC
	return OperationLogApplicationSVC
}

// Collect is invoked by the middleware for every audited write. It is a safe
// no-op when the service has not been initialised (e.g. feature disabled).
func (s *OperationLogApplicationService) Collect(event *entity.Event) {
	if s == nil || s.DomainSVC == nil {
		return
	}
	s.DomainSVC.Collect(event)
}

// LogItem is one query result row, enriched with the operator's display name.
type LogItem struct {
	*entity.OperationLog
	OperatorName string
}

// ListOperationLogs checks that the caller is a space Owner/Admin (canManage),
// queries the log records, and batch-resolves operator display names.
func (s *OperationLogApplicationService) ListOperationLogs(ctx context.Context, f *entity.ListFilter) ([]*LogItem, int64, error) {
	if s == nil || s.DomainSVC == nil || s.userSVC == nil {
		return nil, 0, errorx.New(errno.ErrOperationLogPermissionCode, errorx.KV("msg", "operation log service is disabled"))
	}

	uidPtr := ctxutil.GetUIDFromCtx(ctx)
	if uidPtr == nil {
		return nil, 0, errorx.New(errno.ErrOperationLogPermissionCode, errorx.KV("msg", "not logged in"))
	}

	_, _, _, canManage, err := s.userSVC.CheckMemberPermission(ctx, f.SpaceID, *uidPtr)
	if err != nil {
		return nil, 0, err
	}
	if !canManage {
		return nil, 0, errorx.New(errno.ErrOperationLogPermissionCode, errorx.KV("msg", "only space owner/admin can view operation logs"))
	}

	records, total, err := s.DomainSVC.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}

	names := s.resolveOperatorNames(ctx, records)

	items := make([]*LogItem, 0, len(records))
	for _, r := range records {
		items = append(items, &LogItem{OperationLog: r, OperatorName: names[r.OperatorID]})
	}
	return items, total, nil
}

// resolveOperatorNames de-dupes operator ids and batch-fetches display names
// via usersvc.User.MGetUserProfiles. A lookup failure or a missing user never
// aborts the query — the affected names are simply left empty.
func (s *OperationLogApplicationService) resolveOperatorNames(ctx context.Context, records []*entity.OperationLog) map[int64]string {
	idset := make(map[int64]struct{})
	for _, r := range records {
		if r.OperatorID > 0 {
			idset[r.OperatorID] = struct{}{}
		}
	}
	out := make(map[int64]string, len(idset))
	if len(idset) == 0 {
		return out
	}
	ids := make([]int64, 0, len(idset))
	for id := range idset {
		ids = append(ids, id)
	}

	users, err := s.userSVC.MGetUserProfiles(ctx, ids)
	if err != nil {
		logs.CtxWarnf(ctx, "[operationlog] resolve operator names failed: %v", err)
		return out
	}
	for _, u := range users {
		if u == nil {
			continue
		}
		out[u.UserID] = u.Name
	}
	return out
}
