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

package singleagent

import "testing"

func TestArtifactObjectKeyIsUserSpaceScoped(t *testing.T) {
	got := artifactObjectKey(1, 9, 100, 50, "report.pdf")
	want := "artifacts/s1/u9/p100/c50/report.pdf"
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
