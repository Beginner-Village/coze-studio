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

package database

import (
	"context"

	"github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/database"
	service "github.com/ynet-dev/ynet-studio/backend/domain/memory/database/service"
)

//go:generate  mockgen -destination databasemock/database_mock.go --package databasemock -source database.go
type Database interface {
	ExecuteSQL(ctx context.Context, req *database.ExecuteSQLRequest) (*database.ExecuteSQLResponse, error)
	PublishDatabase(ctx context.Context, req *database.PublishDatabaseRequest) (resp *database.PublishDatabaseResponse, err error)
	DeleteDatabase(ctx context.Context, req *database.DeleteDatabaseRequest) error
	BindDatabase(ctx context.Context, req *database.BindDatabaseToAgentRequest) error
	UnBindDatabase(ctx context.Context, req *database.UnBindDatabaseToAgentRequest) error
	MGetDatabase(ctx context.Context, req *database.MGetDatabaseRequest) (*database.MGetDatabaseResponse, error)
	GetAllDatabaseByAppID(ctx context.Context, req *database.GetAllDatabaseByAppIDRequest) (*database.GetAllDatabaseByAppIDResponse, error)
	ListDatabase(ctx context.Context, req *service.ListDatabaseRequest) (*service.ListDatabaseResponse, error)
	CreateDatabase(ctx context.Context, req *service.CreateDatabaseRequest) (*service.CreateDatabaseResponse, error)
	AddDatabaseRecord(ctx context.Context, req *service.AddDatabaseRecordRequest) error

	Execute(ctx context.Context, request *database.CustomSQLRequest) (*database.Response, error)
	Query(ctx context.Context, request *database.QueryRequest) (*database.Response, error)
	Update(context.Context, *database.UpdateRequest) (*database.Response, error)
	Insert(ctx context.Context, request *database.InsertRequest) (*database.Response, error)
	Delete(context.Context, *database.DeleteRequest) (*database.Response, error)
}

var defaultSVC Database

func DefaultSVC() Database {
	return defaultSVC
}

func SetDefaultSVC(c Database) {
	defaultSVC = c
}
