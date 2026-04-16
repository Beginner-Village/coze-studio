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
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	spaceModel "github.com/ynet-dev/ynet-studio/backend/api/model/space"
	spaceApp "github.com/ynet-dev/ynet-studio/backend/application/space"
	"github.com/ynet-dev/ynet-studio/backend/application/space/release"
)

// CreateRelease creates a new release version for a space
// @router /api/space/{space_id}/release [POST]
func CreateRelease(ctx context.Context, c *app.RequestContext) {
	var req spaceModel.CreateReleaseRequest
	if err := c.BindAndValidate(&req); err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	result, err := spaceApp.ReleaseSVC.CreateRelease(ctx, &release.CreateReleaseRequest{
		SpaceID:     req.SpaceID,
		UserID:      0,
		Version:     req.Version,
		Tag:         req.Tag,
		Description: req.Description,
		SyncType:    req.SyncType,
	})
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(consts.StatusOK, &spaceModel.CreateReleaseResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.CreateReleaseData{
			Version:     result.Version,
			Status:      result.Status,
			PackageSize: result.PackageSize,
			ContentHash: result.ContentHash,
		},
	})
}

// ListReleases lists all releases for a space
// @router /api/space/{space_id}/release/list [GET]
func ListReleases(ctx context.Context, c *app.RequestContext) {
	spaceID, err := strconv.ParseInt(c.Param("space_id"), 10, 64)
	if err != nil {
		c.String(consts.StatusBadRequest, "invalid space_id")
		return
	}

	status := c.DefaultQuery("status", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	records, total, err := spaceApp.ReleaseSVC.ListReleases(ctx, spaceID, status, page, pageSize)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	items := make([]*spaceModel.ReleaseItem, 0, len(records))
	for _, r := range records {
		item := &spaceModel.ReleaseItem{
			Version:     r.Version,
			SyncType:    r.SyncType,
			PackageSize: r.PackageSize,
			Status:      r.Status,
			ContentHash: r.ContentHash,
			CreatedBy:   r.CreatedBy,
			PublishedAt: r.PublishedAt,
			CreatedAt:   r.CreatedAt,
		}
		if r.Tag != nil {
			item.Tag = *r.Tag
		}
		if r.Description != nil {
			item.Description = *r.Description
		}
		if r.ParentVersion != nil {
			item.ParentVersion = *r.ParentVersion
		}
		items = append(items, item)
	}

	c.JSON(consts.StatusOK, &spaceModel.ListReleasesResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.ListReleasesData{
			Items: items,
			Total: total,
		},
	})
}

// GetRelease returns details of a specific release version
// @router /api/space/{space_id}/release/{version} [GET]
func GetRelease(ctx context.Context, c *app.RequestContext) {
	spaceID, err := strconv.ParseInt(c.Param("space_id"), 10, 64)
	if err != nil {
		c.String(consts.StatusBadRequest, "invalid space_id")
		return
	}
	version := c.Param("version")

	detail, err := spaceApp.ReleaseSVC.GetRelease(ctx, spaceID, version)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	r := detail.Release
	item := spaceModel.ReleaseItem{
		Version:     r.Version,
		SyncType:    r.SyncType,
		PackageSize: r.PackageSize,
		Status:      r.Status,
		ContentHash: r.ContentHash,
		CreatedBy:   r.CreatedBy,
		PublishedAt: r.PublishedAt,
		CreatedAt:   r.CreatedAt,
	}
	if r.Tag != nil {
		item.Tag = *r.Tag
	}
	if r.Description != nil {
		item.Description = *r.Description
	}
	if r.ParentVersion != nil {
		item.ParentVersion = *r.ParentVersion
	}

	c.JSON(consts.StatusOK, &spaceModel.GetReleaseResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.GetReleaseData{
			ReleaseItem:  item,
			DownloadURL:  detail.DownloadURL,
			URLExpiresAt: detail.ExpiresAt,
			Statistics:   r.Statistics,
			Manifest:     r.Manifest,
		},
	})
}

// PublishRelease marks a release as published
// @router /api/space/{space_id}/release/{version}/publish [POST]
func PublishRelease(ctx context.Context, c *app.RequestContext) {
	spaceID, err := strconv.ParseInt(c.Param("space_id"), 10, 64)
	if err != nil {
		c.String(consts.StatusBadRequest, "invalid space_id")
		return
	}
	version := c.Param("version")

	if err = spaceApp.ReleaseSVC.PublishRelease(ctx, spaceID, version); err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(consts.StatusOK, &spaceModel.PublishReleaseResponse{Code: 0, Msg: "success"})
}

// DeprecateRelease marks a release as deprecated
// @router /api/space/{space_id}/release/{version}/deprecate [POST]
func DeprecateRelease(ctx context.Context, c *app.RequestContext) {
	spaceID, err := strconv.ParseInt(c.Param("space_id"), 10, 64)
	if err != nil {
		c.String(consts.StatusBadRequest, "invalid space_id")
		return
	}
	version := c.Param("version")

	if err = spaceApp.ReleaseSVC.DeprecateRelease(ctx, spaceID, version); err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(consts.StatusOK, &spaceModel.PublishReleaseResponse{Code: 0, Msg: "success"})
}

// DiffVersions returns the diff between two versions
// @router /api/space/{space_id}/release/{version}/diff [GET]
func DiffVersions(ctx context.Context, c *app.RequestContext) {
	spaceID, err := strconv.ParseInt(c.Param("space_id"), 10, 64)
	if err != nil {
		c.String(consts.StatusBadRequest, "invalid space_id")
		return
	}
	toVersion := c.Param("version")
	fromVersion := c.DefaultQuery("from", "")

	if fromVersion == "" {
		c.String(consts.StatusBadRequest, "from query parameter is required")
		return
	}

	diff, err := spaceApp.ReleaseSVC.DiffVersions(ctx, spaceID, fromVersion, toVersion)
	if err != nil {
		c.String(consts.StatusInternalServerError, err.Error())
		return
	}

	// Convert to API model
	added := make(map[string][]spaceModel.ResourceRef)
	modified := make(map[string][]spaceModel.ResourceRef)
	removed := make(map[string][]spaceModel.ResourceRef)

	for k, v := range diff.Added {
		for _, s := range v {
			added[k] = append(added[k], spaceModel.ResourceRef{ID: s.ID, Name: s.Name})
		}
	}
	for k, v := range diff.Modified {
		for _, s := range v {
			modified[k] = append(modified[k], spaceModel.ResourceRef{ID: s.ID, Name: s.Name})
		}
	}
	for k, v := range diff.Removed {
		for _, s := range v {
			removed[k] = append(removed[k], spaceModel.ResourceRef{ID: s.ID, Name: s.Name})
		}
	}

	c.JSON(consts.StatusOK, &spaceModel.VersionDiffResponse{
		Code: 0,
		Msg:  "success",
		Data: &spaceModel.VersionDiffData{
			FromVersion: diff.FromVersion,
			ToVersion:   diff.ToVersion,
			Added:       added,
			Modified:    modified,
			Removed:     removed,
		},
	})
}
