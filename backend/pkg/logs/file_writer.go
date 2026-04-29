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

package logs

import (
	"io"
	"os"
	"strconv"

	"gopkg.in/natefinch/lumberjack.v2"
)

// NewWriter returns an io.Writer that writes to stderr, and additionally
// to a rolling file if LOG_FILE env is set. Configuration via env:
//
//	LOG_FILE          - empty = stderr only; non-empty = stderr + file
//	LOG_MAX_SIZE_MB   - single file max size (default 100)
//	LOG_MAX_BACKUPS   - old file count to keep (default 7)
//	LOG_MAX_AGE_DAYS  - old file max age in days (default 30)
//	LOG_COMPRESS      - gzip rotated files (default true)
func NewWriter() io.Writer {
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		return os.Stderr
	}

	lj := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    envInt("LOG_MAX_SIZE_MB", 100),
		MaxBackups: envInt("LOG_MAX_BACKUPS", 7),
		MaxAge:     envInt("LOG_MAX_AGE_DAYS", 30),
		Compress:   envBool("LOG_COMPRESS", true),
	}
	return io.MultiWriter(os.Stderr, lj)
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
