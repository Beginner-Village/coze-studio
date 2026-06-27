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

package conversation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActiveAgentRunRegistryCancelsAndUnregisters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	unregister := RegisterActiveAgentRun("run-123", cancel)
	defer unregister()

	assert.True(t, CancelActiveAgentRun("run-123"))
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	assert.False(t, CancelActiveAgentRun("run-123"))
}

func TestActiveAgentRunRegistryReportsStatus(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	unregister := RegisterActiveAgentRun(" run-status ", cancel)
	defer unregister()

	state, ok := GetActiveAgentRun("run-status")
	assert.True(t, ok)
	assert.Equal(t, "run-status", state.RunID)
	assert.Equal(t, "in_progress", state.Status)
	assert.Greater(t, state.CreatedAt, int64(0))
	assert.GreaterOrEqual(t, state.UpdatedAt, state.CreatedAt)

	assert.True(t, CancelActiveAgentRun("run-status"))
	_, ok = GetActiveAgentRun("run-status")
	assert.False(t, ok)
}
