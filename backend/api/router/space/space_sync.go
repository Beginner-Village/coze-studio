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
	"github.com/cloudwego/hertz/pkg/app/server"
	handler "github.com/ynet-dev/ynet-studio/backend/api/handler/space"
)

func RegisterSync(r *server.Hertz) {
	spaceGroup := r.Group("/api/space/:space_id")
	syncGroup := spaceGroup.Group("/sync")
	{
		syncGroup.POST("/export", handler.SyncExport)
		syncGroup.GET("/last-export", handler.SyncLastExport)
		importGroup := syncGroup.Group("/import")
		{
			importGroup.POST("/preview", handler.SyncImportPreview)
			importGroup.POST("/confirm", handler.SyncImportConfirm)
		}
		syncGroup.GET("/history", handler.SyncHistory)
	}
}
