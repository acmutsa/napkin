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
    let spec;

    switch (kind) {
      case "compute": // EC2 Instance
        spec = {
          label: "EC2 Instance",
          color: "bg-blue-50 border-blue-200",
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
          inputs: [], // no inputs
          outputs: [{ id: "db-out", type: "data", label: "DB Output" }], // to EC2
        };
        break;

      case "loadBalancer":
        spec = {
          label: "Load Balancer",
          color: "bg-purple-50 border-purple-200",
          inputs: [{ id: "traffic-in", type: "network", label: "Incoming Traffic" }], // from users / upstream
          outputs: [{ id: "traffic-out", type: "network", label: "Forward Traffic" }], // to EC2 nodes
        };
        break;

      case "securityGroup":
        spec = {
          label: "Security Group",
          color: "bg-yellow-50 border-yellow-200",
          inputs: [{ id: "inbound", type: "network", label: "Inbound" }], // from EC2 or Load Balancer
          outputs: [{ id: "outbound", type: "network", label: "Outbound" }], // to EC2 / network nodes
        };
        break;

      case "storageBucket":
        spec = {
          label: "Storage Bucket",
          color: "bg-orange-50 border-orange-200",
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