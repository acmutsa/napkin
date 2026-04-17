import { transformNodes } from "./transformer";
import { useReactFlow } from "@xyflow/react";

function ExportButton() {
  const { getNodes, getEdges } = useReactFlow();
  const nodes = getNodes();
  const edges = getEdges();

  const result = transformNodes(nodes, edges);
  console.log(JSON.stringify(result, null, 2));

  return <button onClick={() => console.log(result)}>Export Graph</button>;
}

export default ExportButton;