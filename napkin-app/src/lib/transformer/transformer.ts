import type { Node, Edge } from "@xyflow/react";

type TransformedEdge = {
  source: { node: string; id: string; port?: string };
  target: { node: string; id: string; port?: string };
  type: string;
};

type TransformResult = {
  nodes: Record<string, unknown[]>;
  edges: TransformedEdge[];
};

/** Maps canvas labels to Terraform resource types for compile IR. */
function terraformTypeFromSpecLabel(label: string): string {
  const map: Record<string, string> = {
    "EC2 Instance": "aws_instance",
    Database: "aws_db_instance",
    "Load Balancer": "aws_lb",
    "Security Group": "aws_security_group",
    "Storage Bucket": "aws_s3_bucket",
  };
  return map[label] ?? "aws_instance";
}

export function transformNodes(nodeMap: Node[], edges: Edge[]) {
  const result: TransformResult = {
    nodes: {},
    edges: [],
  };

  // process the nodes
  nodeMap.forEach((node) => {
    const type = node.type || "Unknown";

    if (!result.nodes[type]) {
      result.nodes[type] = [];
    }

    const data = node.data as {
      spec?: Record<string, unknown> & { label?: string };
      attributes?: Record<string, string>;
    } | null;
    const rawSpec =
      data?.spec && typeof data.spec === "object" ? data.spec : {};
    const label =
      typeof rawSpec.label === "string" ? rawSpec.label : "";

    // Compile/analyze payload only: omit canvas-only fields (colors, ports).
    result.nodes[type].push({
      id: node.id,
      spec: {
        label,
        class: type,
        type: terraformTypeFromSpecLabel(label),
      },
      ...(data?.attributes ? { attributes: data.attributes } : {}),
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
        port: edge.sourceHandle ?? undefined,
      },
      target: {
        node: targetNode.type,
        id: targetNode.id,
        port: edge.targetHandle ?? undefined,
      },
      type: "data-flow", // could be dynamic later, hard coding for right now
    });
  });

  return result;
}
