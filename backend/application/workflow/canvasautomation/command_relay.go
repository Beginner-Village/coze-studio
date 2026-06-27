package canvasautomation

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

const defaultWorkflowCommandRelayLimit = 200

type WorkflowCanvasQueuedCommand struct {
	ID               int64                 `json:"id"`
	WorkflowID       string                `json:"workflow_id"`
	SpaceID          string                `json:"space_id"`
	RequestID        string                `json:"request_id,omitempty"`
	RequiresResponse bool                  `json:"requires_response,omitempty"`
	Command          WorkflowCanvasCommand `json:"command"`
	CreatedAt        time.Time             `json:"created_at"`
}

type WorkflowCanvasDiagnostic struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	NodeID  string `json:"node_id,omitempty"`
	NodeTag string `json:"node_tag,omitempty"`
	Op      string `json:"op,omitempty"`
}

type WorkflowCanvasCommandResultItem struct {
	Op          string                     `json:"op"`
	OK          bool                       `json:"ok"`
	NodeID      string                     `json:"node_id,omitempty"`
	LineID      string                     `json:"line_id,omitempty"`
	Target      string                     `json:"target,omitempty"`
	Message     string                     `json:"message,omitempty"`
	Diagnostics []WorkflowCanvasDiagnostic `json:"diagnostics,omitempty"`
}

type WorkflowCanvasCommandResult struct {
	Protocol           string                            `json:"protocol,omitempty"`
	RequestID          string                            `json:"request_id"`
	Status             string                            `json:"status"`
	Results            []WorkflowCanvasCommandResultItem `json:"results,omitempty"`
	CanvasContext      string                            `json:"canvas_context,omitempty"`
	BindableVariables  string                            `json:"bindable_variables,omitempty"`
	NodeSmokeReport    any                               `json:"node_smoke_report,omitempty"`
	BindingDiagnostics []WorkflowCanvasDiagnostic        `json:"binding_diagnostics,omitempty"`
}

type MemoryWorkflowCommandRelay struct {
	mu       sync.Mutex
	nextID   int64
	maxItems int
	queues   map[string][]WorkflowCanvasQueuedCommand
	results  map[string]WorkflowCanvasCommandResult
	waiters  map[string][]chan WorkflowCanvasCommandResult
}

func NewMemoryWorkflowCommandRelay() *MemoryWorkflowCommandRelay {
	return &MemoryWorkflowCommandRelay{
		maxItems: defaultWorkflowCommandRelayLimit,
		queues:   make(map[string][]WorkflowCanvasQueuedCommand),
		results:  make(map[string]WorkflowCanvasCommandResult),
		waiters:  make(map[string][]chan WorkflowCanvasCommandResult),
	}
}

func (r *MemoryWorkflowCommandRelay) DispatchWorkflowCommand(_ context.Context, command WorkflowCanvasDispatchCommand) error {
	workflowID := strings.TrimSpace(command.WorkflowID)
	spaceID := strings.TrimSpace(command.SpaceID)
	if workflowID == "" || spaceID == "" {
		return errors.New("workflow_id and space_id are required")
	}
	if strings.TrimSpace(command.Command.Op) == "" {
		return errors.New("command op is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	key := workflowRelayKey(spaceID, workflowID)
	r.queues[key] = append(r.queues[key], WorkflowCanvasQueuedCommand{
		ID:               r.nextID,
		WorkflowID:       workflowID,
		SpaceID:          spaceID,
		RequestID:        strings.TrimSpace(command.RequestID),
		RequiresResponse: command.RequiresResponse,
		Command:          command.Command,
		CreatedAt:        time.Now(),
	})
	if len(r.queues[key]) > r.maxItems {
		r.queues[key] = r.queues[key][len(r.queues[key])-r.maxItems:]
	}
	return nil
}

func (r *MemoryWorkflowCommandRelay) PollWorkflowCommands(spaceID, workflowID string, afterID int64, limit int) []WorkflowCanvasQueuedCommand {
	if limit <= 0 || limit > r.maxItems {
		limit = r.maxItems
	}
	key := workflowRelayKey(strings.TrimSpace(spaceID), strings.TrimSpace(workflowID))

	r.mu.Lock()
	defer r.mu.Unlock()

	items := r.queues[key]
	if len(items) > 0 && afterID > items[len(items)-1].ID {
		afterID = 0
	}
	out := make([]WorkflowCanvasQueuedCommand, 0, len(items))
	for _, item := range items {
		if item.ID <= afterID {
			continue
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func (r *MemoryWorkflowCommandRelay) SubmitWorkflowCommandResult(_ context.Context, result WorkflowCanvasCommandResult) error {
	requestID := strings.TrimSpace(result.RequestID)
	if requestID == "" {
		return errors.New("request_id is required")
	}
	if result.Protocol == "" {
		result.Protocol = "canvas_automation.v0"
	}
	if result.Status == "" {
		result.Status = "ok"
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.results[requestID] = result
	waiters := r.waiters[requestID]
	delete(r.waiters, requestID)
	for _, waiter := range waiters {
		select {
		case waiter <- result:
		default:
		}
		close(waiter)
	}
	return nil
}

func (r *MemoryWorkflowCommandRelay) WaitWorkflowCommandResult(ctx context.Context, requestID string) (WorkflowCanvasCommandResult, bool) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return WorkflowCanvasCommandResult{}, false
	}

	r.mu.Lock()
	if result, ok := r.results[requestID]; ok {
		delete(r.results, requestID)
		r.mu.Unlock()
		return result, true
	}
	waiter := make(chan WorkflowCanvasCommandResult, 1)
	r.waiters[requestID] = append(r.waiters[requestID], waiter)
	r.mu.Unlock()

	select {
	case result, ok := <-waiter:
		if !ok {
			return WorkflowCanvasCommandResult{}, false
		}
		return result, true
	case <-ctx.Done():
		r.mu.Lock()
		waiters := r.waiters[requestID]
		for i, existing := range waiters {
			if existing == waiter {
				r.waiters[requestID] = append(waiters[:i], waiters[i+1:]...)
				break
			}
		}
		if len(r.waiters[requestID]) == 0 {
			delete(r.waiters, requestID)
		}
		r.mu.Unlock()
		return WorkflowCanvasCommandResult{}, false
	}
}

func workflowRelayKey(spaceID, workflowID string) string {
	return spaceID + ":" + workflowID
}

var (
	sharedWorkflowCommandRelay     *MemoryWorkflowCommandRelay
	sharedWorkflowCommandRelayOnce sync.Once
)

// SharedWorkflowCommandRelay 是进程级共享的画布命令 relay 单例。HTTP 路由
// (/api/workflow_mcp/browser_commands、browser_command_results)与 agentflow 的画布工具
// (get_canvas_context)必须共用它,否则前端 poll 的 relay 与后端 agent dispatch 的 relay
// 不是同一个,谁也收不到谁的消息。
func SharedWorkflowCommandRelay() *MemoryWorkflowCommandRelay {
	sharedWorkflowCommandRelayOnce.Do(func() {
		sharedWorkflowCommandRelay = NewMemoryWorkflowCommandRelay()
	})
	return sharedWorkflowCommandRelay
}
