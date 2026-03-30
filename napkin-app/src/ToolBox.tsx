const nodeItems = [
  { kind: 'number', label: 'Number', description: 'Holds a numeric value' },
  { kind: 'text', label: 'Text', description: 'Holds a text value' },
];
type ToolboxProps = {
  onAdd: (kind: string) => void;
};

export default function Toolbox({ onAdd }: ToolboxProps) {
  return (
    <div className="absolute left-4 top-1/2 -translate-y-1/2 z-10 flex flex-col gap-2 bg-white border border-gray-200 rounded-xl shadow-md p-3 min-w-[130px]">
      <p className="text-[10px] uppercase tracking-widest text-gray-400 mb-1 px-1">
        Nodes
      </p>

      {nodeItems.map((node) => (
        <div
          key={node.kind}
          onClick={() => onAdd(node.kind)}
          className="flex flex-col px-3 py-2 rounded-lg border border-gray-200 bg-gray-50 cursor-pointer hover:border-gray-400 hover:bg-white transition-all duration-150 select-none"
        >
          <span className="text-sm font-medium text-gray-700">{node.label}</span>
          <span className="text-[11px] text-gray-400">{node.description}</span>
        </div>
      ))}
    </div>
  );
}