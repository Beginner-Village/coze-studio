export interface AgentCanvasCommandLike {
  op: string;
}

export interface CommandResultLike {
  ok: boolean;
}

export const shouldAutoLayoutAfterAgentCommand = (
  cmd: AgentCanvasCommandLike,
): boolean =>
  cmd.op === 'addNode' ||
  cmd.op === 'connect' ||
  cmd.op === 'deleteNode' ||
  cmd.op === 'deleteLine' ||
  cmd.op === 'clearCanvas' ||
  cmd.op === 'setNodeParams' ||
  cmd.op === 'configureNode';

export const autoLayoutAfterAgentCommandIfNeeded = async <
  T extends CommandResultLike,
>(
  cmd: AgentCanvasCommandLike,
  result: T,
  autoLayout: () => Promise<unknown>,
): Promise<T> => {
  if (result.ok && shouldAutoLayoutAfterAgentCommand(cmd)) {
    await autoLayout();
  }
  return result;
};
