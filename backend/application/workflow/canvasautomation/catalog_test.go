package canvasautomation

import (
	"strings"
	"testing"
)

func TestCatalogContainsCoreWorkflowNodeRules(t *testing.T) {
	capabilities := ListNodeCapabilities()
	if len(capabilities) < 41 {
		t.Fatalf("expected all visible workflow node types, got %d", len(capabilities))
	}

	for _, typ := range []string{"1", "2", "3", "5", "8", "13", "15", "32", "45", "61", "100"} {
		if cap, ok := GetNodeCapability(typ); !ok {
			t.Fatalf("missing capability for type %s", typ)
		} else if strings.TrimSpace(cap.Name) == "" || strings.TrimSpace(cap.SupportLevel) == "" {
			t.Fatalf("capability for type %s is incomplete: %+v", typ, cap)
		}
	}
}

func TestOutputTextAndMergeSpecsProtectBindingRules(t *testing.T) {
	endSpec, ok := GetNodeSpec("2")
	if !ok {
		t.Fatal("missing end node spec")
	}
	for _, want := range []string{"返回变量", "返回文本", "流式输出", "多个变量"} {
		if !strings.Contains(endSpec.Description, want) {
			t.Fatalf("end spec should mention %q, got %q", want, endSpec.Description)
		}
	}

	outputSpec, ok := GetNodeSpec("13")
	if !ok {
		t.Fatal("missing output node spec")
	}
	for _, want := range []string{"display-only", "不要作为变量聚合", "End returns", "type=15"} {
		if !strings.Contains(outputSpec.Description, want) {
			t.Fatalf("output spec should mention %q, got %q", want, outputSpec.Description)
		}
	}

	textSpec, ok := GetNodeSpec("15")
	if !ok {
		t.Fatal("missing text node spec")
	}
	for _, want := range []string{"固定文案", "可以删除默认 input", "output:string"} {
		if !strings.Contains(textSpec.Description, want) {
			t.Fatalf("text spec should mention %q, got %q", want, textSpec.Description)
		}
	}

	mergeSpec, ok := GetNodeSpec("32")
	if !ok {
		t.Fatal("missing merge node spec")
	}
	for _, want := range []string{"workflow.get_bindable_variables", "真实可绑定变量", "不要聚合 type=13"} {
		if !strings.Contains(mergeSpec.Description, want) {
			t.Fatalf("merge spec should mention %q, got %q", want, mergeSpec.Description)
		}
	}
}

func TestCodeNodeSpecUsesPlatformPythonArgsRuntime(t *testing.T) {
	codeSpec, ok := GetNodeSpec("5")
	if !ok {
		t.Fatal("missing code node spec")
	}

	text := codeSpec.Description + "\n" + strings.Join(codeSpec.BindingRules, "\n")
	for _, want := range []string{
		`language 写 "python"`,
		"async def main(args: Args)",
		"args.params",
		"不要对 args 直接调用 strip/get",
		"outputs",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("code spec should mention %q, got %+v", want, codeSpec)
		}
	}
}

func TestCoreEditableNodeSpecsAreConcrete(t *testing.T) {
	for _, typ := range []string{"3", "4", "5", "6", "8", "13", "15", "18", "22", "30", "32", "58", "59", "99", "100"} {
		spec, ok := GetNodeSpec(typ)
		if !ok {
			t.Fatalf("missing spec for type %s", typ)
		}
		if strings.Contains(spec.Description, "节点当前支持等级为") {
			t.Fatalf("type %s should have a concrete spec, got fallback: %+v", typ, spec)
		}
		if len(spec.Commands) == 0 {
			t.Fatalf("type %s spec should declare usable commands: %+v", typ, spec)
		}
	}
}

func TestEveryNodeSpecAvoidsGenericFallback(t *testing.T) {
	for _, cap := range ListNodeCapabilities() {
		spec, ok := GetNodeSpec(cap.Type)
		if !ok {
			t.Fatalf("missing spec for type %s", cap.Type)
		}
		if strings.Contains(spec.Description, "节点当前支持等级为") {
			t.Fatalf("type %s should not expose generic fallback spec: %+v", cap.Type, spec)
		}
		if len(spec.Commands) == 0 {
			t.Fatalf("type %s spec should declare next commands: %+v", cap.Type, spec)
		}
	}
}

func TestCatalogExplainsUnsupportedAndResourceBoundNodes(t *testing.T) {
	httpCap, ok := GetNodeCapability("45")
	if !ok {
		t.Fatal("missing HTTP capability")
	}
	if httpCap.SupportLevel != SupportPartial {
		t.Fatalf("HTTP should remain partial until semantic config is implemented, got %s", httpCap.SupportLevel)
	}
	if len(httpCap.Gaps) == 0 || !strings.Contains(strings.Join(httpCap.Gaps, " "), "method/url") {
		t.Fatalf("HTTP gaps should explain missing semantic config, got %+v", httpCap.Gaps)
	}

	pluginCap, ok := GetNodeCapability("4")
	if !ok {
		t.Fatal("missing plugin capability")
	}
	if pluginCap.SupportLevel != SupportResourceBound || !pluginCap.RequiresResource {
		t.Fatalf("plugin/API should be resource-bound, got %+v", pluginCap)
	}
}
