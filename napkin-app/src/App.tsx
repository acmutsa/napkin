import { useState, useCallback, useMemo } from "react";
import Toolbox from "./ToolBox";
import ResourceNode from "@/components/ResourceNode";
import ResourceEdge from "@/components/ResourceEdge";
import FloatingMenu from "@/components/FloatingMenu";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  addEdge,
  applyNodeChanges,
  applyEdgeChanges,
  ReactFlowProvider,
  type Node,
  type Edge,
  type FitViewOptions,
  type OnConnect,
  type OnNodesChange,
  type OnEdgesChange,
  type DefaultEdgeOptions,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { transformNodes } from "@/lib/transformer/transformer";
import { apiBase } from "@/lib/apiBase";
import { getKind } from "@/lib/kinds";
import { type AnalyzeError } from "./lib/types/errors";

const nodeTypes = {
  resource: ResourceNode,
};
const edgeTypes = {
  resource: ResourceEdge,
};

const initialNodes: Node[] = [];
const initialEdges: Edge[] = [];

const fitViewOptions: FitViewOptions = { padding: 0.2 };
const defaultEdgeOptions: DefaultEdgeOptions = {
  type: "resource",
  animated: true,
  style: { strokeDasharray: "5 5", stroke: "#888" },
};

type BackendError = {
  nodeId?: string | null;
  message: string;
};

function Flow() {
  const [nodes, setNodes] = useState(initialNodes);
  const [edges, setEdges] = useState<Edge[]>(initialEdges);
  const [nodeErrors, setNodeErrors] = useState<Record<string, string>>({});

  const onNodesChange: OnNodesChange = useCallback(
    (changes) => setNodes((nds) => applyNodeChanges(changes, nds)),
    []
  );
  const onEdgesChange: OnEdgesChange = useCallback(
    (changes) => setEdges((eds) => applyEdgeChanges(changes, eds)),
    []
  );
  const onConnect: OnConnect = useCallback(
    (connection) => setEdges((eds) => addEdge(connection, eds)),
    []
  );

  const handleAnalyzeError = useCallback((nodeId: string | null, message: string) => {
    if (nodeId) {
      setNodeErrors((prev) => ({ ...prev, [nodeId]: message }));
    }
  }, []);

  const handleAnalyze = useCallback(
    async (analysisType: string): Promise<AnalyzeError[]> => {
      const graph = transformNodes(nodes, edges);
      setNodeErrors({});

      try {
        const res = await fetch(`${apiBase()}/api/analyze`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            type: analysisType,
            graph,
          }),
        });

        const data = await res.json();
        const menuError: AnalyzeError[] = [];

        const newNodeErrors: Record<string, string> = {};

        data.errors?.forEach((err: BackendError) => {
          if (err.nodeId) {
            newNodeErrors[err.nodeId] = err.message;
          } else {
            menuError.push({
              severity: "error",
              message: err.message,
            });
          }
        });

        setNodeErrors(newNodeErrors)

        return menuError;
      } catch (error) {
        return [
          {
            severity: "error",
            message: (error as Error).message,
          }
        ]
      }
    },
    [nodes, edges, handleAnalyzeError]
  );

  const intentGraph = useMemo(
    () => transformNodes(nodes, edges),
    [nodes, edges],
  );

  const onAdd = useCallback((kind: string) => {
    const def = getKind(kind);
    if (!def) return;

    const newNode: Node = {
      id: `${Date.now()}`,
      type: "resource",
      position: { x: 100, y: 100 },
      data: {
        kind: def.kind,
        spec: def,
        attributes: {},
      },
    };

    setNodes((nds) => [...nds, newNode]);
  }, []);

  const nodesWithErrors = nodes.map((node) => ({
    ...node,
    data: {
      ...node.data,
      error: nodeErrors[node.id] ?? null,
    },
  }));

  return (
    <div className="w-screen h-screen relative">
      <Toolbox onAdd={onAdd} />
      <FloatingMenu onAnalyze={handleAnalyze} intentGraph={intentGraph} />
      <ReactFlow
        nodes={nodesWithErrors}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        fitView
        fitViewOptions={fitViewOptions}
        defaultEdgeOptions={defaultEdgeOptions}
      >
        <Background />
        <Controls />
        <MiniMap />
      </ReactFlow>
    </div>
  );
}

export default function App() {
  return (
    <ReactFlowProvider>
      <Flow />
    </ReactFlowProvider>
  );
}
