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

package agentflow

import (
	"testing"

	crossentity "github.com/ynet-dev/ynet-studio/backend/api/model/crossdomain/singleagent"
	"github.com/ynet-dev/ynet-studio/backend/domain/agent/singleagent/entity"
)

func TestIsSuperAgent(t *testing.T) {
	mk := func(ty string) *Config {
		return &Config{Agent: &entity.SingleAgent{SingleAgent: &crossentity.SingleAgent{AgentType: ty}}}
	}
	if !isSuperAgent(mk("super")) {
		t.Fatal("super => true")
	}
	if isSuperAgent(mk("")) || isSuperAgent(mk("normal")) {
		t.Fatal("empty/normal => false")
	}
	if isSuperAgent(nil) {
		t.Fatal("nil => false")
	}
}
