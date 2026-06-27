package canvasautomation

import (
	"context"
	"testing"
	"time"
)

func TestMemoryWorkflowCommandRelayQueuesAndPollsByWorkflow(t *testing.T) {
	relay := NewMemoryWorkflowCommandRelay()

	if err := relay.DispatchWorkflowCommand(context.Background(), WorkflowCanvasDispatchCommand{
		WorkflowID: "wf-1",
		SpaceID:    "space-1",
		Command: WorkflowCanvasCommand{
			Op:     "add_node",
			Target: "text",
			Args:   map[string]any{"type": "15"},
		},
	}); err != nil {
		t.Fatalf("dispatch command failed: %v", err)
	}
	if err := relay.DispatchWorkflowCommand(context.Background(), WorkflowCanvasDispatchCommand{
		WorkflowID: "wf-2",
		SpaceID:    "space-1",
		Command: WorkflowCanvasCommand{
			Op:     "add_node",
			Target: "other",
			Args:   map[string]any{"type": "3"},
		},
	}); err != nil {
		t.Fatalf("dispatch other workflow command failed: %v", err)
	}

	firstBatch := relay.PollWorkflowCommands("space-1", "wf-1", 0, 10)
	if len(firstBatch) != 1 {
		t.Fatalf("expected one command for wf-1, got %+v", firstBatch)
	}
	if firstBatch[0].Command.Target != "text" || firstBatch[0].ID == 0 {
		t.Fatalf("unexpected queued command: %+v", firstBatch[0])
	}

	secondBatch := relay.PollWorkflowCommands("space-1", "wf-1", firstBatch[0].ID, 10)
	if len(secondBatch) != 0 {
		t.Fatalf("expected cursor to suppress old commands, got %+v", secondBatch)
	}
}

func TestMemoryWorkflowCommandRelayRejectsMissingIdentity(t *testing.T) {
	relay := NewMemoryWorkflowCommandRelay()

	if err := relay.DispatchWorkflowCommand(context.Background(), WorkflowCanvasDispatchCommand{
		WorkflowID: "",
		SpaceID:    "space-1",
		Command:    WorkflowCanvasCommand{Op: "auto_layout"},
	}); err == nil {
		t.Fatal("expected missing workflow id to fail")
	}
}

func TestMemoryWorkflowCommandRelayWaitsForBrowserCommandResult(t *testing.T) {
	relay := NewMemoryWorkflowCommandRelay()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		_ = relay.SubmitWorkflowCommandResult(context.Background(), WorkflowCanvasCommandResult{
			RequestID:          "req-1",
			Status:             "ok",
			CanvasContext:      "节点: Start -> Text -> End",
			BindingDiagnostics: []WorkflowCanvasDiagnostic{},
		})
	}()

	result, ok := relay.WaitWorkflowCommandResult(ctx, "req-1")
	if !ok {
		t.Fatal("expected browser command result before context timeout")
	}
	if result.CanvasContext != "节点: Start -> Text -> End" {
		t.Fatalf("unexpected browser command result: %+v", result)
	}
}

func TestMemoryWorkflowCommandRelayPollsRequestMetadataForResponseCommands(t *testing.T) {
	relay := NewMemoryWorkflowCommandRelay()

	if err := relay.DispatchWorkflowCommand(context.Background(), WorkflowCanvasDispatchCommand{
		WorkflowID:       "wf-1",
		SpaceID:          "space-1",
		RequestID:        "req-context",
		RequiresResponse: true,
		Command: WorkflowCanvasCommand{
			Op: "get_canvas_context",
		},
	}); err != nil {
		t.Fatalf("dispatch command failed: %v", err)
	}

	batch := relay.PollWorkflowCommands("space-1", "wf-1", 0, 10)
	if len(batch) != 1 {
		t.Fatalf("expected one command, got %+v", batch)
	}
	if batch[0].RequestID != "req-context" || !batch[0].RequiresResponse {
		t.Fatalf("expected request metadata to round-trip, got %+v", batch[0])
	}
}

func TestMemoryWorkflowCommandRelayTreatsCursorBeyondLatestAsStale(t *testing.T) {
	relay := NewMemoryWorkflowCommandRelay()

	if err := relay.DispatchWorkflowCommand(context.Background(), WorkflowCanvasDispatchCommand{
		WorkflowID: "wf-1",
		SpaceID:    "space-1",
		Command: WorkflowCanvasCommand{
			Op:     "delete_node",
			Target: "text",
		},
	}); err != nil {
		t.Fatalf("dispatch command failed: %v", err)
	}

	batch := relay.PollWorkflowCommands("space-1", "wf-1", 99, 10)
	if len(batch) != 1 {
		t.Fatalf("expected stale cursor to be reset, got %+v", batch)
	}
	if batch[0].Command.Op != "delete_node" {
		t.Fatalf("unexpected command after stale cursor reset: %+v", batch[0])
	}
}
