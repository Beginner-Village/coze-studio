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
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/testcontainers/testcontainers-go"
	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
)

var (
	testGormDB    *gorm.DB
	testRawDB     *sql.DB
	teardowns     []func()
	testContainer testcontainers.Container
)

func TestMain(m *testing.M) {
	// Parse flags so testing.Short() works correctly.
	testing.Init()
	flag.Parse()
	if testing.Short() {
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := startMySQL(ctx); err != nil {
		log.Printf("failed to start MySQL container: %v", err)
		log.Println("integration tests require Docker — skipping")
		os.Exit(0)
	}
	defer runTeardowns()

	if err := runMigrations(ctx); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	code := m.Run()
	runTeardowns()
	os.Exit(code)
}

func runTeardowns() {
	for _, t := range teardowns {
		t()
	}
	teardowns = nil
}

func startMySQL(ctx context.Context) error {
	c, err := tcmysql.Run(ctx, "mysql:8.4",
		tcmysql.WithDatabase("ynet_test"),
		tcmysql.WithUsername("root"),
		tcmysql.WithPassword("root"),
	)
	if err != nil {
		return err
	}
	testContainer = c
	teardowns = append(teardowns, func() {
		_ = c.Terminate(context.Background())
	})

	connStr, err := c.ConnectionString(ctx, "parseTime=true", "multiStatements=true")
	if err != nil {
		return err
	}

	rawDB, err := sql.Open("mysql", connStr)
	if err != nil {
		return err
	}
	if err := rawDB.PingContext(ctx); err != nil {
		return err
	}
	testRawDB = rawDB

	gdb, err := gorm.Open(gormmysql.New(gormmysql.Config{Conn: rawDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}
	testGormDB = gdb
	return nil
}

// runMigrations creates the minimal subset of tables required by the integration tests.
// It avoids loading the full production schema (2k+ lines) and only creates tables that
// the targeted services actually touch.
func runMigrations(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS space_sync_mapping (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			source_space_id BIGINT NOT NULL,
			target_space_id BIGINT NOT NULL,
			resource_type VARCHAR(64) NOT NULL,
			source_resource_id BIGINT NOT NULL,
			target_resource_id BIGINT NOT NULL,
			source_updated_at BIGINT NOT NULL DEFAULT 0,
			content_hash VARCHAR(128) NOT NULL DEFAULT '',
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uniq_source_resource (source_space_id, resource_type, source_resource_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS space_release (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			space_id BIGINT NOT NULL,
			version VARCHAR(64) NOT NULL,
			tag VARCHAR(255) NULL,
			description TEXT NULL,
			sync_type VARCHAR(32) NOT NULL DEFAULT 'full',
			parent_version VARCHAR(64) NULL,
			manifest JSON NOT NULL,
			statistics JSON NOT NULL,
			package_key VARCHAR(512) NOT NULL,
			package_size BIGINT NOT NULL DEFAULT 0,
			content_hash VARCHAR(128) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'draft',
			created_by BIGINT NOT NULL,
			published_at BIGINT NULL,
			created_at BIGINT NOT NULL,
			updated_at BIGINT NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uniq_space_version (space_id, version)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, stmt := range stmts {
		if _, err := testRawDB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}
	return nil
}
