import {
  createWorkflowCanvasCommandEnvelope,
  type WorkflowCanvasCommand,
  type WorkflowCanvasCommandEnvelope,
  type WorkflowCanvasCommandOp,
  type WorkflowCanvasCommandResult,
} from './workflow-agent-command-protocol';

type BrowserCommandRelayFetcher = (
  input: string,
  init?: RequestInit,
) => Promise<{
  ok: boolean;
  status?: number;
  json: () => Promise<unknown>;
}>;

interface BrowserCommandRelayPayload {
  status?: string;
  protocol?: string;
  mode?: string;
  space_id?: string;
  canvas_id?: string;
  cursor?: number;
  commands?: unknown[];
  requests?: unknown[];
  error?: string;
}

interface BrowserCommandRelayRequest {
  requestId: string;
  requiresResponse: boolean;
  command: WorkflowCanvasCommand;
}

export interface PollWorkflowCanvasBrowserCommandsOptions {
  workflowId: string;
  spaceId: string;
  afterId?: number;
  limit?: number;
  fetcher?: BrowserCommandRelayFetcher;
}

export interface PollWorkflowCanvasBrowserCommandsResult {
  cursor: number;
  envelope: WorkflowCanvasCommandEnvelope;
  envelopes: WorkflowCanvasCommandEnvelope[];
}

export interface PostWorkflowCanvasBrowserCommandResultOptions {
  workflowId: string;
  spaceId: string;
  result: WorkflowCanvasCommandResult;
  fetcher?: BrowserCommandRelayFetcher;
}

export const pollWorkflowCanvasBrowserCommands = async ({
  workflowId,
  spaceId,
  afterId = 0,
  limit,
  fetcher = fetch,
}: PollWorkflowCanvasBrowserCommandsOptions): Promise<PollWorkflowCanvasBrowserCommandsResult> => {
  const query = new URLSearchParams();
  query.set('workflow_id', workflowId);
  query.set('space_id', spaceId);
  query.set('after_id', String(afterId));
  if (limit !== undefined) {
    query.set('limit', String(limit));
  }

  const response = await fetcher(
    `/api/workflow_mcp/browser_commands?${query.toString()}`,
    { method: 'GET' },
  );
  const payload = (await response.json()) as BrowserCommandRelayPayload;
  if (!response.ok || payload.status !== 'ok') {
    throw new Error(payload.error || `browser command relay failed: ${response.status ?? 'unknown'}`);
  }

  const cursor =
    typeof payload.cursor === 'number' && Number.isFinite(payload.cursor)
      ? payload.cursor
      : afterId;
  const commands = Array.isArray(payload.commands)
    ? payload.commands.map(normalizeRelayCommand).filter(Boolean)
    : [];
  const requests = Array.isArray(payload.requests)
    ? payload.requests.map(normalizeRelayRequest).filter(Boolean)
    : [];
  const envelopes = requests.length
    ? requests.map(request =>
        createWorkflowCanvasCommandEnvelope([request.command], {
          requestId: request.requestId,
          surface: 'workflow',
          mode: 'browser_live',
          spaceId: payload.space_id || spaceId,
          canvasId: payload.canvas_id || workflowId,
          requiresResponse: request.requiresResponse,
        }),
      )
    : [
        createWorkflowCanvasCommandEnvelope(commands, {
          requestId: `workflow-mcp-relay-${cursor}`,
          surface: 'workflow',
          mode: 'browser_live',
          spaceId: payload.space_id || spaceId,
          canvasId: payload.canvas_id || workflowId,
        }),
      ];

  const envelope =
    envelopes.length === 1
      ? envelopes[0]
      : createWorkflowCanvasCommandEnvelope(
          requests.map(request => request.command),
          {
            requestId: `workflow-mcp-relay-${cursor}`,
            surface: 'workflow',
            mode: 'browser_live',
            spaceId: payload.space_id || spaceId,
            canvasId: payload.canvas_id || workflowId,
          },
        );
  return {
    cursor,
    envelope,
    envelopes,
  };
};

export const postWorkflowCanvasBrowserCommandResult = async ({
  workflowId,
  spaceId,
  result,
  fetcher = fetch,
}: PostWorkflowCanvasBrowserCommandResultOptions): Promise<void> => {
  const response = await fetcher('/api/workflow_mcp/browser_command_results', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      workflow_id: workflowId,
      space_id: spaceId,
      result,
    }),
  });
  const payload = (await response.json()) as { status?: string; error?: string };
  if (!response.ok || payload.status !== 'ok') {
    throw new Error(payload.error || `browser command result submission failed: ${response.status ?? 'unknown'}`);
  }
};

const normalizeRelayCommand = (
  value: unknown,
): WorkflowCanvasCommand | undefined => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return undefined;
  }
  const raw = value as Record<string, unknown>;
  if (typeof raw.op !== 'string' || !raw.op) {
    return undefined;
  }
  return {
    op: raw.op as WorkflowCanvasCommandOp,
    target: typeof raw.target === 'string' ? raw.target : undefined,
    args:
      raw.args && typeof raw.args === 'object' && !Array.isArray(raw.args)
        ? (raw.args as Record<string, unknown>)
        : undefined,
  };
};

const normalizeRelayRequest = (
  value: unknown,
): BrowserCommandRelayRequest | undefined => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return undefined;
  }
  const raw = value as Record<string, unknown>;
  const command = normalizeRelayCommand(raw.command);
  if (!command) {
    return undefined;
  }
  const requestId =
    typeof raw.request_id === 'string' && raw.request_id
      ? raw.request_id
      : typeof raw.id === 'number'
        ? `workflow-mcp-relay-request-${raw.id}`
        : '';
  if (!requestId) {
    return undefined;
  }
  return {
    requestId,
    requiresResponse: raw.requires_response === true,
    command,
  };
};
