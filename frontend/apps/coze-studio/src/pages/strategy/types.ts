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

export interface AddCapabilityForm {
  type: string;
  ref_id: string;
  ref_sub_id: string;
  prompt_content: string;
  alias_name: string;
  alias_description: string;
}

export interface EditCapabilityForm {
  alias_name: string;
  alias_description: string;
  prompt_content: string;
}
