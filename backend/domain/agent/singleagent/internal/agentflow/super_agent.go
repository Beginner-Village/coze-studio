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

// SuperAgentType 是超级智能体的 agent_type 取值。
const SuperAgentType = "super"

// isSuperAgent 报告该 agent 是否为超级智能体。
func isSuperAgent(conf *Config) bool {
	return conf != nil && conf.Agent != nil && conf.Agent.AgentType == SuperAgentType
}
