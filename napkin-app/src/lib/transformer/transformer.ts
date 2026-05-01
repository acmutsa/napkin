import type { Node, Edge } from "@xyflow/react";
import { awsRegionDefault } from "@/lib/apiBase";
import { getKind, type KindDef, type PortDef } from "@/lib/kinds";

type TransformedEdge = {
  source: { node: string; id: string; port?: string };
  target: { node: string; id: string; port?: string };
  type: string;
};

type TransformedPorts = {
  inputs: PortDef[];
  outputs: PortDef[];
};

type TransformResult = {
  region: string;
  nodes: Record<string, unknown[]>;
  edges: TransformedEdge[];
};

type CanvasNodeData = {
  kind?: string;
  spec?: Partial<KindDef> & { label?: string };
  attributes?: Record<string, string>;
};

export function transformNodes(nodeMap: Node[], edges: Edge[]) {
  const result: TransformResult = {
    region: awsRegionDefault(),
    nodes: {},
    edges: [],
  };

  nodeMap.forEach((node) => {
    const type = node.type || "Unknown";

    if (!result.nodes[type]) {
      result.nodes[type] = [];
    }

    const data = (node.data ?? {}) as CanvasNodeData;
    const def = getKind(data.kind);

    const label = data.spec?.label ?? def?.label ?? "";
    const terraformType = def?.terraformType ?? data.spec?.terraformType;

    const ports: TransformedPorts = {
      inputs: data.spec?.inputs ?? def?.inputs ?? [],
      outputs: data.spec?.outputs ?? def?.outputs ?? [],
    };

    result.nodes[type].push({
      id: node.id,
      ...(data.kind ? { kind: data.kind } : {}),
      spec: {
        label,
        class: type,
        ...(terraformType ? { type: terraformType } : {}),
      },
      ports,
      ...(data.attributes ? { attributes: data.attributes } : {}),
    });
  });

  const nodeById = new Map<string, Node>();
  nodeMap.forEach((node) => {
    nodeById.set(node.id, node);
  });

  edges.forEach((edge) => {
    const sourceNode = nodeById.get(edge.source);
    const targetNode = nodeById.get(edge.target);

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
        port: edge.sourceHandle ?? undefined,
      },
      target: {
        node: targetNode.type,
        id: targetNode.id,
        port: edge.targetHandle ?? undefined,
      },
      type: "data-flow",
    });
  });

  return result;
}
