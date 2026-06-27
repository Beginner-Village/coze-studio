const API_NODE_TYPE = '4';
const START_NODE_TYPE = '1';
const END_NODE_TYPE = '2';
const START_NODE_ID = '100001';
const END_NODE_ID = '900001';

interface AgentBlockInput {
  name: string;
  input: {
    type: 'string';
    value: {
      type: 'literal';
      content: string;
      rawMeta: { type: 1 };
    };
  };
}

interface AgentPlaceholderNodeJson {
  data: {
    nodeMeta: {
      title: string;
      description: string;
    };
    inputs: {
      apiParam: AgentBlockInput[];
    };
    outputs: unknown[];
  };
}

const createStringBlockInput = (name: string, value = ''): AgentBlockInput => ({
  name,
  input: {
    type: 'string',
    value: {
      type: 'literal',
      content: value,
      rawMeta: { type: 1 },
    },
  },
});

export const isWorkflowAgentSingletonNodeType = (type: string): boolean =>
  type === START_NODE_TYPE || type === END_NODE_TYPE;

export const resolveWorkflowAgentSingletonNodeRef = (
  ref: string,
): string | undefined => {
  const normalized = ref.trim().toLowerCase();
  if (normalized === 'start' || normalized === START_NODE_ID) {
    return START_NODE_ID;
  }
  if (normalized === 'end' || normalized === END_NODE_ID) {
    return END_NODE_ID;
  }
  return undefined;
};

export const createWorkflowAgentPlaceholderNodeJson = (
  type: string,
  title?: string,
): AgentPlaceholderNodeJson | undefined => {
  if (type !== API_NODE_TYPE) {
    return undefined;
  }

  return {
    data: {
      nodeMeta: {
        title: title || 'Plugin/API - bind resource',
        description: 'Bind a real plugin/API resource before running.',
      },
      inputs: {
        apiParam: [
          createStringBlockInput('apiID'),
          createStringBlockInput('apiName'),
          createStringBlockInput('pluginID'),
          createStringBlockInput('pluginName'),
          createStringBlockInput('pluginVersion'),
          createStringBlockInput('tips'),
          createStringBlockInput('outDocLink'),
        ],
      },
      outputs: [],
    },
  };
};
