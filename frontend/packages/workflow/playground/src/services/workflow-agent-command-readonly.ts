import {
  type WorkflowCanvasCommand,
  type WorkflowCanvasCommandResultItem,
} from './workflow-agent-command-protocol';

export interface WorkflowCanvasReadonlyReaders {
  getCanvasSummary: () => Promise<string>;
  getBindableVariablesSummary: () => Promise<string>;
}

export interface WorkflowCanvasReadonlyCommandResult {
  item: WorkflowCanvasCommandResultItem;
  canvasContext?: string;
  bindableVariables?: string;
}

export const executeWorkflowCanvasReadonlyCommand = async (
  command: WorkflowCanvasCommand,
  readers: WorkflowCanvasReadonlyReaders,
): Promise<WorkflowCanvasReadonlyCommandResult | undefined> => {
  if (command.op === 'get_canvas_context') {
    const canvasContext = await readers.getCanvasSummary();
    return {
      item: {
        op: command.op,
        ok: true,
        target: command.target,
        diagnostics: [],
      },
      canvasContext,
    };
  }

  if (command.op === 'get_bindable_variables') {
    const bindableVariables = await readers.getBindableVariablesSummary();
    return {
      item: {
        op: command.op,
        ok: true,
        target: command.target,
        diagnostics: [],
      },
      bindableVariables,
    };
  }

  return undefined;
};
