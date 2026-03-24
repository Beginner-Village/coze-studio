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

package space

import (
	"gorm.io/gorm"

	spaceexport "github.com/ynet-dev/ynet-studio/backend/application/space/export"
	spaceimport "github.com/ynet-dev/ynet-studio/backend/application/space/import"
	spacesync "github.com/ynet-dev/ynet-studio/backend/application/space/sync"
	"github.com/ynet-dev/ynet-studio/backend/domain/search/service"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/idgen"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/storage"
)

var SyncSVC *spacesync.SyncService

func InitSyncService(db *gorm.DB, objectStorage storage.Storage, idGen idgen.IDGenerator,
	eventBus service.ResourceEventBus, projectEventBus service.ProjectEventBus) {
	exporter := spaceexport.NewSpaceExporter(db, objectStorage)
	importer := spaceimport.NewSpaceImporter(db, idGen, eventBus, projectEventBus)
	SyncSVC = spacesync.NewSyncService(db, exporter, importer, objectStorage, idGen, eventBus, projectEventBus)
}
