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

package skill

import (
	"github.com/cloudwego/hertz/pkg/app/server"

	skillHandler "github.com/ynet-dev/ynet-studio/backend/api/handler/skill"
)

// Register registers skill routes.
func Register(r *server.Hertz) {
	root := r.Group("/")
	{
		api := root.Group("/api")
		{
			skillGroup := api.Group("/skill")
			skillGroup.POST("/create", skillHandler.CreateSkill)
			skillGroup.GET("/get", skillHandler.GetSkill)
			skillGroup.POST("/update", skillHandler.UpdateSkill)
			skillGroup.POST("/delete", skillHandler.DeleteSkill)
			skillGroup.POST("/publish", skillHandler.PublishSkill)
			skillGroup.POST("/review", skillHandler.ReviewSkill)
			skillGroup.GET("/review/pending", skillHandler.ListPendingReviews)
			skillGroup.GET("/list", skillHandler.ListSkills)
			skillGroup.GET("/marketplace/list", skillHandler.ListMarketplaceSkills)
			skillGroup.GET("/marketplace/get", skillHandler.GetMarketplaceSkill)
			skillGroup.POST("/marketplace/install", skillHandler.InstallMarketplaceSkill)
		}
	}
}
