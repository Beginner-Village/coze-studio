/*
 * Copyright 2025 coze-dev Authors
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

// Package idgen provides a simple ID generator using a snowflake-like algorithm.
// This generates 64-bit unique IDs that are roughly time-ordered.
package idgen

import (
	"sync"
	"time"
)

const (
	// customEpoch is a custom epoch (2024-01-01 00:00:00 UTC)
	customEpoch int64 = 1704067200000

	// Bit allocation
	timestampBits = 41  // milliseconds since custom epoch
	machineBits   = 10  // machine ID (not used in single instance)
	sequenceBits  = 12  // sequence number

	// Maximum values
	maxSequence = (1 << sequenceBits) - 1

	// Bit shifts
	machineShift   = sequenceBits
	timestampShift = sequenceBits + machineBits
)

var (
	mu            sync.Mutex
	lastTimestamp int64
	sequence      int64
	machineID     int64 = 1 // Single instance, use fixed machine ID
)

// NextID generates a unique ID using snowflake algorithm
func NextID() uint64 {
	mu.Lock()
	defer mu.Unlock()

	timestamp := currentMillis()

	if timestamp == lastTimestamp {
		sequence = (sequence + 1) & maxSequence
		if sequence == 0 {
			// Sequence overflow, wait for next millisecond
			for timestamp <= lastTimestamp {
				timestamp = currentMillis()
			}
		}
	} else {
		sequence = 0
	}

	lastTimestamp = timestamp

	// Generate ID
	id := ((timestamp - customEpoch) << timestampShift) |
		(machineID << machineShift) |
		sequence

	return uint64(id)
}

// currentMillis returns the current time in milliseconds
func currentMillis() int64 {
	return time.Now().UnixMilli()
}
