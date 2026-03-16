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

package errno

import (
	"github.com/ynet-dev/ynet-studio/backend/pkg/errorx/code"
)

// skill: 111 000 000 ~ 111 999 999
const (
	ErrSkillInvalidParamCode  = 111000000
	ErrSkillNotFoundCode      = 111000001
	ErrSkillPermissionCode    = 111000002
	ErrSkillCreateCode        = 111000003
	ErrSkillUpdateCode        = 111000004
	ErrSkillDeleteCode        = 111000005
	ErrSkillListCode          = 111000006
	ErrSkillIDGenFailCode     = 111000007
	ErrSkillDuplicateNameCode = 111000008
)

func init() {
	code.Register(
		ErrSkillInvalidParamCode,
		"invalid parameter : {msg}",
		code.WithAffectStability(false),
	)

	code.Register(
		ErrSkillNotFoundCode,
		"skill not found: {id}",
		code.WithAffectStability(false),
	)

	code.Register(
		ErrSkillPermissionCode,
		"unauthorized access : {msg}",
		code.WithAffectStability(false),
	)

	code.Register(
		ErrSkillCreateCode,
		"create skill failed",
		code.WithAffectStability(true),
	)

	code.Register(
		ErrSkillUpdateCode,
		"update skill failed",
		code.WithAffectStability(true),
	)

	code.Register(
		ErrSkillDeleteCode,
		"delete skill failed",
		code.WithAffectStability(true),
	)

	code.Register(
		ErrSkillListCode,
		"list skills failed",
		code.WithAffectStability(true),
	)

	code.Register(
		ErrSkillIDGenFailCode,
		"gen id failed : {msg}",
		code.WithAffectStability(true),
	)

	code.Register(
		ErrSkillDuplicateNameCode,
		"skill name already exists: {name}",
		code.WithAffectStability(false),
	)
}
