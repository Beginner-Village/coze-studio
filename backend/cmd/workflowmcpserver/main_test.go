package main

import "testing"

func TestCommandUsesSystemWorkflowMCPServer(t *testing.T) {
	srv := newWorkflowMCPServer(nil)
	for _, name := range []string{
		"workflow.list_node_capabilities",
		"workflow.get_node_spec",
		"workflow.test_run",
	} {
		if srv.GetTool(name) == nil {
			t.Fatalf("expected command server to expose %s", name)
		}
	}
}
