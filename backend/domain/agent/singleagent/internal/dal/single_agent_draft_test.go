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

	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/internal/dal/model"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/internal/dal/query"
	"github.com/ynet-dev/ynet-studio/backend/internal/mock/infra/contract/orm"
)

func TestSingleAgentDraftSuite(t *testing.T) {
	suite.Run(t, new(SingleAgentDraftSuite))
}

type SingleAgentDraftSuite struct {
	suite.Suite

	ctx context.Context
	db  *gorm.DB
	dao *SingleAgentDraftDAO
}

func (s *SingleAgentDraftSuite) SetupSuite() {
	s.ctx = context.Background()
	mockDB := orm.NewMockDB()
	mockDB.AddTable(&model.SingleAgentDraft{})
	db, err := mockDB.DB()
	if err != nil {
		panic(err)
	}
	s.db = db
	query.SetDefault(db)
	s.dao = &SingleAgentDraftDAO{
		dbQuery: query.Use(db),
	}
}

func (s *SingleAgentDraftSuite) TearDownTest() {
	s.db.WithContext(s.ctx).Unscoped().Where("1 = 1").Delete(&model.SingleAgentDraft{})
}

func (s *SingleAgentDraftSuite) TestAgentTypePersist() {
	PatchConvey("agent_type 落库与读取", s.T(), func() {
		ctx := s.ctx
		at := "super"
		So(s.dao.dbQuery.SingleAgentDraft.WithContext(ctx).Create(&model.SingleAgentDraft{
			AgentID: 90001, SpaceID: 100, Name: "super_test", IconURI: "u", AgentType: &at,
		}), ShouldBeNil)
		got, err := s.dao.Get(ctx, 90001)
		So(err, ShouldBeNil)
		So(got.AgentType, ShouldEqual, "super")
	})
}

func (s *SingleAgentDraftSuite) TestListBySpaceID() {
	PatchConvey("test list by space id", s.T(), func() {
		ctx := s.ctx
		q := s.dao.dbQuery.SingleAgentDraft

		// Seed 3 in space 100
		for _, id := range []int64{1001, 1002, 1003} {
			So(q.WithContext(ctx).Create(&model.SingleAgentDraft{
				AgentID: id,
				SpaceID: 100,
				Name:    "a",
				IconURI: "u",
			}), ShouldBeNil)
		}
		// Seed 1 in space 200
		So(q.WithContext(ctx).Create(&model.SingleAgentDraft{
			AgentID: 2001,
			SpaceID: 200,
			Name:    "b",
			IconURI: "u",
		}), ShouldBeNil)

		// no limit
		got, err := s.dao.ListBySpaceID(ctx, 100, 0)
		So(err, ShouldBeNil)
		So(len(got), ShouldEqual, 3)
		So(got[0].AgentID, ShouldEqual, 1001)
		So(got[2].AgentID, ShouldEqual, 1003)

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
