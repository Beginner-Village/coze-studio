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

package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	spaceexport "github.com/ynet-dev/ynet-studio/backend/application/space/export"
)

// HashPackage computes SHA256 of a ZIP byte slice
func HashPackage(zipContent []byte) string {
	h := sha256.Sum256(zipContent)
	return hex.EncodeToString(h[:])
}

// BuildResourceHashes computes a SHA256 hash for each resource in the export.
// Returns a map keyed by "resourceType:resourceID" -> hash.
func BuildResourceHashes(resources *spaceexport.SpaceResources) (map[string]string, error) {
	hashes := make(map[string]string)
	if resources == nil {
		return hashes, nil
	}

	for _, a := range resources.Agents {
		h, err := hashJSON(a)
		if err != nil {
			return nil, fmt.Errorf("hash agent %d: %w", a.ID, err)
		}
		hashes[fmt.Sprintf("agent:%d", a.ID)] = h
	}

	for _, p := range resources.Plugins {
		h, err := hashJSON(p)
		if err != nil {
			return nil, fmt.Errorf("hash plugin %d: %w", p.ID, err)
		}
		hashes[fmt.Sprintf("plugin:%d", p.ID)] = h
	}

	for _, w := range resources.Workflows {
		h, err := hashJSON(w)
		if err != nil {
			return nil, fmt.Errorf("hash workflow %d: %w", w.ID, err)
		}
		hashes[fmt.Sprintf("workflow:%d", w.ID)] = h
	}

	for _, v := range resources.Variables {
		h, err := hashJSON(v)
		if err != nil {
			return nil, fmt.Errorf("hash variable %d: %w", v.ID, err)
		}
		hashes[fmt.Sprintf("variable:%d", v.ID)] = h
	}

	for _, sm := range resources.SpaceModels {
		h, err := hashJSON(sm)
		if err != nil {
			return nil, fmt.Errorf("hash space_model %d: %w", sm.ID, err)
		}
		hashes[fmt.Sprintf("space_model:%d", sm.ID)] = h
	}

	for _, kb := range resources.KnowledgeBases {
		h, err := hashJSON(kb)
		if err != nil {
			return nil, fmt.Errorf("hash knowledge %d: %w", kb.ID, err)
		}
		hashes[fmt.Sprintf("knowledge:%d", kb.ID)] = h
	}

	for _, f := range resources.Folders {
		h, err := hashJSON(f)
		if err != nil {
			return nil, fmt.Errorf("hash folder %d: %w", f.ID, err)
		}
		hashes[fmt.Sprintf("folder:%d", f.ID)] = h
	}

	for _, ek := range resources.ExternalKnowledge {
		h, err := hashJSON(ek)
		if err != nil {
			return nil, fmt.Errorf("hash external_knowledge %d: %w", ek.ID, err)
		}
		hashes[fmt.Sprintf("external_knowledge:%d", ek.ID)] = h
	}

	return hashes, nil
}

func hashJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}
