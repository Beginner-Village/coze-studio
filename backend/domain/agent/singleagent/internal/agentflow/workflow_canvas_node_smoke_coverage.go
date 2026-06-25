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

import "strings"

type wfCanvasNodeSmokeCoverageReport struct {
	Status               string                          `json:"status"`
	TotalNodes           int                             `json:"total_nodes"`
	ManifestNodes        int                             `json:"manifest_nodes"`
	ExecutableTypes      []string                        `json:"executable_types"`
	ReadonlyTypes        []string                        `json:"readonly_types"`
	SkippedTypes         []string                        `json:"skipped_types"`
	ResourceFixtureTypes []string                        `json:"resource_fixture_types"`
	PartialTypes         []string                        `json:"partial_types"`
	MissingManifestTypes []string                        `json:"missing_manifest_types"`
	MissingSpecTypes     []string                        `json:"missing_spec_types"`
	CoverageGaps         []string                        `json:"coverage_gaps"`
	Nodes                []wfCanvasNodeSmokeCoverageItem `json:"nodes"`
	Instruction          string                          `json:"instruction"`
}

type wfCanvasNodeSmokeCoverageItem struct {
	Type                         string   `json:"type"`
	Name                         string   `json:"name"`
	SupportLevel                 string   `json:"support_level"`
	RuntimeSmoke                 string   `json:"runtime_smoke,omitempty"`
	Mode                         string   `json:"mode"`
	Isolation                    string   `json:"isolation,omitempty"`
	Ready                        bool     `json:"ready"`
	RequiredTools                []string `json:"required_tools,omitempty"`
	PlannedTools                 []string `json:"planned_tools,omitempty"`
	CleanupTools                 []string `json:"cleanup_tools,omitempty"`
	TemporaryNodeTags            []string `json:"temporary_node_tags,omitempty"`
	HasSpec                      bool     `json:"has_spec"`
	HasAssertions                bool     `json:"has_assertions"`
	Assertions                   []string `json:"assertions,omitempty"`
	HasExpectedBindableVariables bool     `json:"has_expected_bindable_variables"`
	ExpectedBindableVariables    []string `json:"expected_bindable_variables,omitempty"`
	CanBeDownstreamSource        bool     `json:"can_be_downstream_source"`
	RequiresResourceFixture      bool     `json:"requires_resource_fixture,omitempty"`
	RequiresTemporaryWorkflow    bool     `json:"requires_temporary_workflow,omitempty"`
	SkipReason                   string   `json:"skip_reason,omitempty"`
	Gaps                         []string `json:"gaps,omitempty"`
}

func buildWFCanvasNodeSmokeCoverageReport() wfCanvasNodeSmokeCoverageReport {
	manifest := buildWFCanvasNodeSmokeManifest()
	manifestByType := make(map[string]wfCanvasNodeSmokeManifestNode, len(manifest.Nodes))
	for _, node := range manifest.Nodes {
		manifestByType[node.Type] = node
	}

	report := wfCanvasNodeSmokeCoverageReport{
		Status:               "node_smoke_coverage",
		TotalNodes:           len(wfCanvasSmokeCapabilities),
		ManifestNodes:        len(manifest.Nodes),
		ExecutableTypes:      []string{},
		ReadonlyTypes:        []string{},
		SkippedTypes:         []string{},
		ResourceFixtureTypes: []string{},
		PartialTypes:         []string{},
		MissingManifestTypes: []string{},
		MissingSpecTypes:     []string{},
		CoverageGaps:         []string{},
		Nodes:                make([]wfCanvasNodeSmokeCoverageItem, 0, len(wfCanvasSmokeCapabilities)),
		Instruction: "复杂建图前先读本覆盖总表: full/execute 节点可优先使用; resource-bound 必须先发现真实资源;" +
			" partial/sub-canvas/add-only 节点不要猜配置; type=13 是 display-only,不能作为 VariableMerge 或 End returns 的稳定来源;" +
			" 配置变量聚合、End、IF、文本模板前必须调用 workflow_canvas_get_bindable_variables。",
	}

	for _, cap := range wfCanvasSmokeCapabilities {
		manifestNode, hasManifest := manifestByType[cap.Type]
		if !hasManifest {
			report.MissingManifestTypes = append(report.MissingManifestTypes, cap.Type)
		}
		spec, hasSpec := wfNodeSpecForType(cap.Type)
		if !hasSpec {
			report.MissingSpecTypes = append(report.MissingSpecTypes, cap.Type)
		}
		item := wfCanvasNodeSmokeCoverageItem{
			Type:                         cap.Type,
			Name:                         cap.Name,
			SupportLevel:                 cap.SupportLevel,
			RuntimeSmoke:                 cap.RuntimeSmoke,
			Mode:                         manifestNode.Mode,
			Isolation:                    manifestNode.Isolation,
			Ready:                        manifestNode.Ready,
			RequiredTools:                append([]string(nil), manifestNode.RequiredTools...),
			PlannedTools:                 append([]string(nil), manifestNode.PlannedTools...),
			CleanupTools:                 append([]string(nil), manifestNode.CleanupTools...),
			TemporaryNodeTags:            append([]string(nil), manifestNode.TemporaryNodeTags...),
			HasSpec:                      hasSpec && strings.TrimSpace(spec) != "",
			HasAssertions:                len(manifestNode.Assertions) > 0,
			Assertions:                   append([]string(nil), manifestNode.Assertions...),
			HasExpectedBindableVariables: len(manifestNode.ExpectedBindableVariables) > 0,
			ExpectedBindableVariables:    append([]string(nil), manifestNode.ExpectedBindableVariables...),
			CanBeDownstreamSource:        wfCanvasNodeSmokeCanBeDownstreamSource(cap, manifestNode),
			RequiresResourceFixture:      manifestNode.RequiresResourceFixture,
			RequiresTemporaryWorkflow:    manifestNode.RequiresTemporaryWorkflowOptIn,
			SkipReason:                   manifestNode.SkipReason,
			Gaps:                         append([]string(nil), cap.Gaps...),
		}
		report.Nodes = append(report.Nodes, item)
		report.addNodeTypeToBuckets(item)
	}
	report.CoverageGaps = buildWFCanvasNodeSmokeCoverageGaps(report)
	return report
}

func (r *wfCanvasNodeSmokeCoverageReport) addNodeTypeToBuckets(item wfCanvasNodeSmokeCoverageItem) {
	switch item.Mode {
	case "execute":
		r.ExecutableTypes = append(r.ExecutableTypes, item.Type)
	case "readonly":
		r.ReadonlyTypes = append(r.ReadonlyTypes, item.Type)
	case "skip":
		r.SkippedTypes = append(r.SkippedTypes, item.Type)
	}
	if item.RequiresResourceFixture {
		r.ResourceFixtureTypes = append(r.ResourceFixtureTypes, item.Type)
	}
	if item.SupportLevel == wfCanvasSmokeSupportPartial || item.RuntimeSmoke == "sub-canvas" {
		r.PartialTypes = append(r.PartialTypes, item.Type)
	}
}

func buildWFCanvasNodeSmokeCoverageGaps(report wfCanvasNodeSmokeCoverageReport) []string {
	gaps := []string{}
	for _, typ := range report.MissingManifestTypes {
		gaps = append(gaps, typ+" 缺少 node smoke manifest 条目")
	}
	for _, typ := range report.MissingSpecTypes {
		gaps = append(gaps, typ+" 缺少 node spec")
	}
	return gaps
}

func wfCanvasNodeSmokeCanBeDownstreamSource(cap wfCanvasSmokeCapability, node wfCanvasNodeSmokeManifestNode) bool {
	if cap.Type == "13" || cap.SupportLevel == wfCanvasSmokeSupportDocumentationOnly || cap.SupportLevel == wfCanvasSmokeSupportSingleton {
		return false
	}
	if node.Mode == "skip" {
		return false
	}
	return len(node.ExpectedBindableVariables) > 0
}
