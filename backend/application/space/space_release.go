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
	"github.com/ynet-dev/ynet-studio/backend/application/space/release"
	"github.com/ynet-dev/ynet-studio/backend/infra/contract/storage"
)

var ReleaseSVC *release.ReleaseService

func InitReleaseService(db *gorm.DB, objectStorage storage.Storage) {
	exporter := spaceexport.NewSpaceExporter(db, objectStorage)
	ReleaseSVC = release.NewReleaseService(db, exporter, objectStorage)
}
