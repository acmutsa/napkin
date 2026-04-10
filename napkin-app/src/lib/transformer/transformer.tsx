import type { Node, Edge } from "@xyflow/react";

type TransformedEdge = {
  source: { node: string; id: string; port?: string };
  target: { node: string; id: string; port?: string };
  type: string;
};

type TransformResult = {
  nodes: Record<string, any[]>;
  edges: TransformedEdge[];
};

export function transformNodes(nodeMap: Node[], edges: Edge[]) {
  const result: TransformResult = {
    nodes: {} as Record<string, any[]>,
    edges: [] as any[]
  };

  // process the nodes
  nodeMap.forEach((node) => {
    const type = node.type || "Unknown";

    if (!result.nodes[type]) {
      result.nodes[type] = [];
    }

    result.nodes[type].push({
      id: node.id,
      ...node.data   // assuming this works, if not then TODO: create mapping layer
    });
  });


  // Build lookup map for nodes for easier edge processing
  const nodeById = new Map<string, Node>();
  nodeMap.forEach((node) => {
    nodeById.set(node.id, node);
  });

  // process the edges
  edges.forEach((edge) => {
    const sourceNode = nodeById.get(edge.source);
    const targetNode = nodeById.get(edge.target);

    
    // error catching
    if (!sourceNode || !targetNode) {
      console.warn("Invalid edge (missing node)", edge);
      return;
    }

    if (!sourceNode.type || !targetNode.type) {
      console.warn("Node missing type:", { sourceNode, targetNode });
      return;
    }

    result.edges.push({
      source: {
        node: sourceNode.type,
        id: sourceNode.id,
        port: edge.sourceHandle ?? undefined
      },
      target: {
        node: targetNode.type,
        id: targetNode.id,
        port: edge.targetHandle ?? undefined
      },
      type: "data-flow" // could be dynamic later, hard coding for right now
    });
  });

  return result;
}