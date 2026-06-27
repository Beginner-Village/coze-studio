package canvasautomation

import "strings"

type NodeSmokeCoverageReport struct {
	TotalNodes              int                     `json:"total_nodes"`
	ManifestNodes           int                     `json:"manifest_nodes"`
	ExecutableTypes         []string                `json:"executable_types"`
	ReadonlyTypes           []string                `json:"readonly_types"`
	SkippedTypes            []string                `json:"skipped_types"`
	ResourceFixtureTypes    []string                `json:"resource_fixture_types"`
	SubCanvasOrPartialTypes []string                `json:"sub_canvas_or_partial_types"`
	MissingManifestTypes    []string                `json:"missing_manifest_types"`
	MissingSpecTypes        []string                `json:"missing_spec_types"`
	CoverageGaps            []string                `json:"coverage_gaps"`
	Nodes                   []NodeSmokeCoverageItem `json:"nodes"`
}

type NodeSmokeCoverageItem struct {
	Type                         string   `json:"type"`
	Name                         string   `json:"name"`
	SupportLevel                 string   `json:"support_level"`
	Registry                     string   `json:"registry"`
	RuntimeSmoke                 string   `json:"runtime_smoke,omitempty"`
	Mode                         string   `json:"mode"`
	Isolation                    string   `json:"isolation,omitempty"`
	Ready                        bool     `json:"ready"`
	RequiredCommands             []string `json:"required_commands,omitempty"`
	PlannedCommands              []string `json:"planned_commands,omitempty"`
	CleanupCommands              []string `json:"cleanup_commands,omitempty"`
	MissingConcreteCommands      []string `json:"missing_concrete_commands,omitempty"`
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

func BuildNodeSmokeCoverageReport() NodeSmokeCoverageReport {
	manifest := BuildNodeSmokeManifest()
	manifestByType := make(map[string]NodeSmokeManifestNode, len(manifest.Nodes))
	for _, node := range manifest.Nodes {
		manifestByType[node.Type] = node
	}

	report := NodeSmokeCoverageReport{
		TotalNodes:              len(nodeCapabilities),
		ManifestNodes:           len(manifest.Nodes),
		ExecutableTypes:         []string{},
		ReadonlyTypes:           []string{},
		SkippedTypes:            []string{},
		ResourceFixtureTypes:    []string{},
		SubCanvasOrPartialTypes: []string{},
		MissingManifestTypes:    []string{},
		MissingSpecTypes:        []string{},
		CoverageGaps:            []string{},
		Nodes:                   make([]NodeSmokeCoverageItem, 0, len(nodeCapabilities)),
	}

	for _, cap := range nodeCapabilities {
		manifestNode, hasManifest := manifestByType[cap.Type]
		if !hasManifest {
			report.MissingManifestTypes = append(report.MissingManifestTypes, cap.Type)
		}
		spec, hasSpec := GetNodeSpec(cap.Type)
		if !hasSpec {
			report.MissingSpecTypes = append(report.MissingSpecTypes, cap.Type)
		}
		item := NodeSmokeCoverageItem{
			Type:                         cap.Type,
			Name:                         cap.Name,
			SupportLevel:                 cap.SupportLevel,
			Registry:                     cap.Registry,
			RuntimeSmoke:                 cap.RuntimeSmoke,
			Mode:                         manifestNode.Mode,
			Isolation:                    manifestNode.Isolation,
			Ready:                        manifestNode.Ready,
			RequiredCommands:             append([]string(nil), manifestNode.RequiredCommands...),
			PlannedCommands:              append([]string(nil), manifestNode.PlannedCommands...),
			CleanupCommands:              append([]string(nil), manifestNode.CleanupCommands...),
			MissingConcreteCommands:      append([]string(nil), manifestNode.MissingConcreteCommands...),
			TemporaryNodeTags:            append([]string(nil), manifestNode.TemporaryNodeTags...),
			HasSpec:                      hasSpec && strings.TrimSpace(spec.Description) != "",
			HasAssertions:                len(manifestNode.Assertions) > 0,
			Assertions:                   append([]string(nil), manifestNode.Assertions...),
			HasExpectedBindableVariables: len(manifestNode.ExpectedBindableVariables) > 0,
			ExpectedBindableVariables:    append([]string(nil), manifestNode.ExpectedBindableVariables...),
			CanBeDownstreamSource:        canNodeSmokeItemBeDownstreamSource(cap, manifestNode),
			RequiresResourceFixture:      manifestNode.RequiresResourceFixture,
			RequiresTemporaryWorkflow:    manifestNode.RequiresTemporaryWorkflowOptIn,
			SkipReason:                   manifestNode.SkipReason,
			Gaps:                         append([]string(nil), cap.Gaps...),
		}
		report.Nodes = append(report.Nodes, item)
		report.addNodeTypeToBuckets(item)
	}
	report.CoverageGaps = buildNodeSmokeCoverageGaps(report)
	return report
}

func (r *NodeSmokeCoverageReport) addNodeTypeToBuckets(item NodeSmokeCoverageItem) {
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
	if item.SupportLevel == SupportPartial || item.RuntimeSmoke == "sub-canvas" {
		r.SubCanvasOrPartialTypes = append(r.SubCanvasOrPartialTypes, item.Type)
	}
}

func buildNodeSmokeCoverageGaps(report NodeSmokeCoverageReport) []string {
	gaps := []string{}
	for _, typ := range report.MissingManifestTypes {
		gaps = append(gaps, typ+" 缺少 node smoke manifest 条目")
	}
	for _, typ := range report.MissingSpecTypes {
		gaps = append(gaps, typ+" 缺少 node spec")
	}
	return gaps
}

func canNodeSmokeItemBeDownstreamSource(cap NodeCapability, node NodeSmokeManifestNode) bool {
	if cap.Type == "13" || cap.SupportLevel == SupportDocumentationOnly || cap.SupportLevel == SupportSingleton {
		return false
	}
	if node.Mode == "skip" {
		return false
	}
	return len(node.ExpectedBindableVariables) > 0
}
