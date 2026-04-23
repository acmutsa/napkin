import { useState, useCallback } from "react";
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
        const res = await fetch("http://localhost:8080/api/analyze", {
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

  const onAdd = useCallback(
  (kind: string) => {
    let spec;

    switch (kind) {
      case "compute": // EC2 Instance
        spec = {
          label: "EC2 Instance",
          color: "bg-blue-50 border-blue-200",
          borderColor: "border-blue-200",
          iconColor: "text-blue-400",
          inputs: [
            { id: "db-in", type: "data", label: "DB Connection" }, // from Database
            { id: "traffic-in", type: "network", label: "Incoming Traffic" }, // from Load Balancer
          ],
          outputs: [
            { id: "network-out", type: "network", label: "Network" }, // to Security Group / downstream
            { id: "data-out", type: "data", label: "Storage Output" }, // to Storage Bucket
          ],
        };
        break;

      case "database":
        spec = {
          label: "Database",
          color: "bg-green-50 border-green-200",
          borderColor: "border-green-200",
          iconColor: "text-green-400",
          inputs: [], // no inputs
          outputs: [{ id: "db-out", type: "data", label: "DB Output" }], // to EC2
        };
        break;

      case "loadBalancer":
        spec = {
          label: "Load Balancer",
          color: "bg-purple-50 border-purple-200",
          borderColor: "border-purple-200",
          iconColor: "text-purple-400",
          inputs: [{ id: "traffic-in", type: "network", label: "Incoming Traffic" }], // from users / upstream
          outputs: [{ id: "traffic-out", type: "network", label: "Forward Traffic" }], // to EC2 nodes
        };
        break;

      case "securityGroup":
        spec = {
          label: "Security Group",
          color: "bg-yellow-50 border-yellow-200",
          borderColor: "border-yellow-200",
          iconColor: "text-yellow-400",
          inputs: [{ id: "inbound", type: "network", label: "Inbound" }], // from EC2 or Load Balancer
          outputs: [{ id: "outbound", type: "network", label: "Outbound" }], // to EC2 / network nodes
        };
        break;

      case "storageBucket":
        spec = {
          label: "Storage Bucket",
          color: "bg-orange-50 border-orange-200",
          borderColor: "border-orange-200",
          iconColor: "text-orange-400",
          inputs: [{ id: "data-in", type: "data", label: "Objects" }], // from EC2
          outputs: [], // sink
        };
        break;

      default:
        return; // unknown kind
    }

    const newNode: Node = {
      id: `${Date.now()}`,
      type: "resource",
      position: { x: 100, y: 100 }, // fixed starting position
      data: { spec },
    };

    setNodes((nds) => [...nds, newNode]);
  },
  []
);

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
      <FloatingMenu 
        onAnalyze={handleAnalyze}
      />
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
