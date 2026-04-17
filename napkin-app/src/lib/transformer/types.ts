export type FlowNode = {
  id: string;
  type: string;
  data: Record<string, any>;
};

export type FlowEdge = {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
};

export type FlowJSON = {
    nodes: FlowNode[];
    edges: FlowEdge[];
}

