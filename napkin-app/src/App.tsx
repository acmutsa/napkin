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

  const onAdd = useCallback(
    (kind: string) => {
      let spec;

      switch (kind) {
        case "compute":
          spec = {
            label: "EC2 Instance",
            color: "bg-blue-50 border-blue-200",
            borderColor: "border-blue-200",
            iconColor: "text-blue-400",
            inputs: [
              { id: "db-in", type: "data", label: "DB Connection" },
              { id: "traffic-in", type: "network", label: "Incoming Traffic" },
            ],
            outputs: [
              { id: "network-out", type: "network", label: "Network" },
              { id: "data-out", type: "data", label: "Storage Output" },
            ],
          };
          break;

        case "database":
          spec = {
            label: "Database",
            color: "bg-green-50 border-green-200",
            borderColor: "border-green-200",
            iconColor: "text-green-400",
            inputs: [],
            outputs: [{ id: "db-out", type: "data", label: "DB Output" }],
          };
          break;

        case "loadBalancer":
          spec = {
            label: "Load Balancer",
            color: "bg-purple-50 border-purple-200",
            borderColor: "border-purple-200",
            iconColor: "text-purple-400",
            inputs: [{ id: "traffic-in", type: "network", label: "Incoming Traffic" }],
            outputs: [{ id: "traffic-out", type: "network", label: "Forward Traffic" }],
          };
          break;

        case "securityGroup":
          spec = {
            label: "Security Group",
            color: "bg-yellow-50 border-yellow-200",
            borderColor: "border-yellow-200",
            iconColor: "text-yellow-400",
            inputs: [{ id: "inbound", type: "network", label: "Inbound" }],
            outputs: [{ id: "outbound", type: "network", label: "Outbound" }],
          };
          break;

        case "storageBucket":
          spec = {
            label: "Storage Bucket",
            color: "bg-orange-50 border-orange-200",
            borderColor: "border-orange-200",
            iconColor: "text-orange-400",
            inputs: [{ id: "data-in", type: "data", label: "Objects" }],
            outputs: [],
          };
          break;

        default:
          return;
      }

      const newNode: Node = {
        id: `${Date.now()}`,
        type: "resource",
        position: { x: 100, y: 100 },
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
      <FloatingMenu onAnalyzeError={handleAnalyzeError} />

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
