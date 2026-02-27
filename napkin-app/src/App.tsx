import { useState, useCallback } from "react";
import Toolbox from "./ToolBox";
import ResourceNode from "@/components/ResourceNode";
import CustomNode from "./CustomNode";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  addEdge,
  applyNodeChanges,
  applyEdgeChanges,
  useReactFlow,
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
  custom: CustomNode,
  resource: ResourceNode,
};

const initialNodes: Node[] = [];
const initialEdges: Edge[] = [];

const fitViewOptions: FitViewOptions = { padding: 0.2 };
const defaultEdgeOptions: DefaultEdgeOptions = {
  animated: true,
  style: { strokeDasharray: "5 5", stroke: "#888" },
};

function Flow() {
  const [nodes, setNodes] = useState(initialNodes);
  const [edges, setEdges] = useState<Edge[]>(initialEdges);
  const { screenToFlowPosition } = useReactFlow();

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

  const onAdd = useCallback(
    (kind: string) => {
      const centerX = window.innerWidth / 2;
      const centerY = window.innerHeight / 2;
      const position = screenToFlowPosition({ x: centerX, y: centerY });

      const newNode: Node = {
        id: `${Date.now()}`,
        type: "resource",
        position,
        data: {
          spec: {
            label: kind === "compute" ? "EC2 Instance" : "Database",
            color:
              kind === "compute"
                ? "bg-blue-50 border-blue-200"
                : "bg-green-50 border-green-200",
            inputs:
              kind === "compute"
                ? [{ id: "db-conn", type: "data", label: "DB Connection" }]
                : [],
            outputs:
              kind === "compute"
                ? [{ id: "vpc", type: "network", label: "Network" }]
                : [{ id: "db-out", type: "data", label: "DB Output" }],
          },
        },
      };

      setNodes((nds) => [...nds, newNode]);
    },
    [screenToFlowPosition]
  );

  return (
    <div className="w-screen h-screen relative">
      <Toolbox onAdd={onAdd} />

      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
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