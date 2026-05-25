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

	. "github.com/bytedance/mockey"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/ynet-dev/ynet-studio/backend/domain/app/internal/dal/model"
	"github.com/ynet-dev/ynet-studio/backend/domain/app/internal/dal/query"
	"github.com/ynet-dev/ynet-studio/backend/internal/mock/infra/contract/orm"
)

func TestAPPDraftSuite(t *testing.T) {
	suite.Run(t, new(APPDraftSuite))
}

type APPDraftSuite struct {
	suite.Suite

	ctx context.Context
	db  *gorm.DB
	dao *APPDraftDAO
}

func (s *APPDraftSuite) SetupSuite() {
	s.ctx = context.Background()
	mockDB := orm.NewMockDB()
	mockDB.AddTable(&model.AppDraft{})
	db, err := mockDB.DB()
	if err != nil {
		panic(err)
	}
	s.db = db
	s.dao = &APPDraftDAO{
		query: query.Use(db),
	}
}

func (s *APPDraftSuite) TearDownTest() {
	s.db.WithContext(s.ctx).Unscoped().Where("1 = 1").Delete(&model.AppDraft{})
}

func (s *APPDraftSuite) TestListBySpaceID() {
	PatchConvey("test list by space id", s.T(), func() {
		ctx := s.ctx
		q := s.dao.query.AppDraft

		// Seed 3 in space 100
		for _, id := range []int64{11, 12, 13} {
			So(q.WithContext(ctx).Create(&model.AppDraft{
				ID:      id,
				SpaceID: 100,
				OwnerID: 1,
				Name:    "a",
				IconURI: "u",
			}), ShouldBeNil)
		}
		// Seed 1 in space 200
		So(q.WithContext(ctx).Create(&model.AppDraft{
			ID:      21,
			SpaceID: 200,
			OwnerID: 1,
			Name:    "b",
			IconURI: "u",
		}), ShouldBeNil)

		// no limit
		got, err := s.dao.ListBySpaceID(ctx, 100, 0)
		So(err, ShouldBeNil)
		So(len(got), ShouldEqual, 3)
		So(got[0].ID, ShouldEqual, 11)
		So(got[2].ID, ShouldEqual, 13)

		// limit 2
		got, err = s.dao.ListBySpaceID(ctx, 100, 2)
		So(err, ShouldBeNil)
		So(len(got), ShouldEqual, 2)

		// non-existent space
		got, err = s.dao.ListBySpaceID(ctx, 999, 0)
		So(err, ShouldBeNil)
		So(len(got), ShouldEqual, 0)
	})
}
