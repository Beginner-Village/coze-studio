// Copyright 2025 ynet-dev Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build integration

package space_sync_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanity_DBConnected(t *testing.T) {
	require.NotNil(t, testGormDB, "gorm DB should be initialized")
	require.NotNil(t, testRawDB, "raw DB should be initialized")
	require.NoError(t, testRawDB.PingContext(context.Background()))
}

func TestSanity_TablesCreated(t *testing.T) {
	rows, err := testRawDB.QueryContext(context.Background(),
		"SHOW TABLES")
	require.NoError(t, err)
	defer rows.Close()

	tables := map[string]bool{}
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables[name] = true
	}
	assert.True(t, tables["space_sync_mapping"], "space_sync_mapping table should exist")
	assert.True(t, tables["space_release"], "space_release table should exist")
}
