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

package rerank

import (
	"github.com/cloudwego/hertz/pkg/app/server"

	"github.com/ynet-dev/ynet-studio/backend/api/handler/rerank"
)

// Register registers all rerank routes
func Register(r *server.Hertz) {
	rerankGroup := r.Group("/api/rerank")
	{
		spaceGroup := rerankGroup.Group("/space")
		{
			spaceGroup.POST("/create", rerank.CreateSpaceRerank)
			spaceGroup.POST("/list", rerank.ListSpaceReranks)
			spaceGroup.POST("/default", rerank.GetSpaceDefaultRerank)
			spaceGroup.POST("/update", rerank.UpdateSpaceRerank)
			spaceGroup.POST("/delete", rerank.DeleteSpaceRerank)
			spaceGroup.POST("/set-default", rerank.SetDefaultSpaceRerank)
			spaceGroup.POST("/enable", rerank.EnableSpaceRerank)
			spaceGroup.POST("/disable", rerank.DisableSpaceRerank)
		}
	}
}
