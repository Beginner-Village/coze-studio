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

package service

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// MinQueryLen is the minimum number of meaningful runes (after trim) a query must have.
// Queries shorter than this are considered junk (too short to convey intent).
const MinQueryLen = 3

// isJunkQuery returns true when a query is too short, all-same-rune, all-punctuation,
// or otherwise unsuitable for retrieval. Such queries are filtered at the Retrieve
// entry to prevent retrieval-system noise (e.g., embedding OOV-fallback collisions,
// BM25 top1-normalization artifacts) from producing spurious high-score hits.
func isJunkQuery(query string) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return true
	}
	if utf8.RuneCountInString(q) < MinQueryLen {
		return true
	}
	first, _ := utf8.DecodeRuneInString(q)
	allSame := true
	allPunctOrSpace := true
	for _, r := range q {
		if r != first {
			allSame = false
		}
		if !unicode.IsPunct(r) && !unicode.IsSpace(r) && !unicode.IsSymbol(r) {
			allPunctOrSpace = false
		}
		if !allSame && !allPunctOrSpace {
			break
		}
	}
	return allSame || allPunctOrSpace
}
