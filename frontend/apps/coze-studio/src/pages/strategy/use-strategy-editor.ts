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

import { useStrategyData } from './use-strategy-data';
import { useScenarioActions } from './use-scenario-actions';
import { useCapabilityActions } from './use-capability-actions';

/** Composes strategy data + scenario actions + capability actions into one API. */
export const useStrategyEditor = (strategyId: string | undefined) => {
  const data = useStrategyData(strategyId);

  const scenario = useScenarioActions({
    strategyId,
    selectedScenarioId: data.selectedScenarioId,
    setSelectedScenarioId: data.setSelectedScenarioId,
    reload: data.reload,
  });

  const capability = useCapabilityActions({
    strategyId,
    selectedScenarioId: data.selectedScenarioId,
    reload: data.reload,
  });

  return { ...data, ...scenario, ...capability };
};
