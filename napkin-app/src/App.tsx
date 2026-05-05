import { useState, useCallback, useMemo } from "react";
import Toolbox from "./ToolBox";
import ResourceNode from "@/components/ResourceNode";
import ResourceEdge from "@/components/ResourceEdge";
import FloatingMenu, { type InheritanceMap } from "@/components/FloatingMenu";
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
import { getKind, validateConnection } from "@/lib/kinds";
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
  const [nodeInheritance, setNodeInheritance] = useState<InheritanceMap>({});
  const [connectError, setConnectError] = useState<string | null>(null);

  const onNodesChange: OnNodesChange = useCallback(
    (changes) => {
      setNodes((nds) => applyNodeChanges(changes, nds));
      setNodeInheritance({});
    },
    [],
  );
  const onEdgesChange: OnEdgesChange = useCallback(
    (changes) => {
      setEdges((eds) => applyEdgeChanges(changes, eds));
      setNodeInheritance({});
    },
    [],
  );
  const onConnect: OnConnect = useCallback(
    (connection) => {
      const sourceNode = nodes.find((n) => n.id === connection.source);
      const targetNode = nodes.find((n) => n.id === connection.target);
      const sourceKind =
        (sourceNode?.data as { kind?: string } | undefined)?.kind;
      const targetKind =
        (targetNode?.data as { kind?: string } | undefined)?.kind;

      const reason = validateConnection({
        sourceKind,
        sourcePortId: connection.sourceHandle,
        targetKind,
        targetPortId: connection.targetHandle,
      });
      if (reason) {
        setConnectError(reason);
        return;
      }
      setConnectError(null);
      setEdges((eds) => addEdge(connection, eds));
    },
    [nodes],
  );

  const isValidConnection = useCallback(
    (connection: { source: string | null; target: string | null; sourceHandle?: string | null; targetHandle?: string | null }) => {
      const sourceNode = nodes.find((n) => n.id === connection.source);
      const targetNode = nodes.find((n) => n.id === connection.target);
      const sourceKind =
        (sourceNode?.data as { kind?: string } | undefined)?.kind;
      const targetKind =
        (targetNode?.data as { kind?: string } | undefined)?.kind;
      return (
        validateConnection({
          sourceKind,
          sourcePortId: connection.sourceHandle,
          targetKind,
          targetPortId: connection.targetHandle,
        }) === null
      );
    },
    [nodes],
  );

  const handleAnalyzeError = useCallback(
    (nodeId: string | null, message: string) => {
      if (nodeId) {
        setNodeErrors((prev) => ({ ...prev, [nodeId]: message }));
      }
    },
    [],
  );

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

        setNodeErrors(newNodeErrors);

        return menuError;
      } catch (error) {
        return [
          {
            severity: "error",
            message: (error as Error).message,
          },
        ];
      }
    },
    [nodes, edges, handleAnalyzeError],
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

  const nodesWithMeta = nodes.map((node) => ({
    ...node,
    data: {
      ...node.data,
      error: nodeErrors[node.id] ?? null,
      inherited: nodeInheritance[node.id] ?? null,
    },
  }));

  return (
    <div className="w-screen h-screen relative">
      <Toolbox onAdd={onAdd} />
      <FloatingMenu
        onAnalyze={handleAnalyze}
        intentGraph={intentGraph}
        onCompileResult={setNodeInheritance}
      />
      {connectError && (
        <div
          role="alert"
          className="absolute top-4 left-1/2 -translate-x-1/2 z-20 max-w-lg rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-700 shadow"
        >
          <div className="flex items-start gap-2">
            <span className="flex-1">{connectError}</span>
            <button
              type="button"
              onClick={() => setConnectError(null)}
              className="text-red-500 hover:text-red-700"
              aria-label="Dismiss"
            >
              x
            </button>
          </div>
        </div>
      )}
      <ReactFlow
        nodes={nodesWithMeta}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        isValidConnection={isValidConnection}
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
