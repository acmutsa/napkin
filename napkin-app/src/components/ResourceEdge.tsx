import { type EdgeProps } from '@xyflow/react';
import { useReactFlow } from '@xyflow/react';
import { ButtonEdge } from '@/components/button-edge';
import { Button } from '@/components/ui/button';
import { X } from 'lucide-react';

export default function ResourceEdge(props: EdgeProps) {
  const { deleteElements } = useReactFlow();

  function handleDelete() {
    deleteElements({ edges: [{ id: props.id }] });
  }

  return (
    <ButtonEdge {...props}>
        <Button
        onClick={handleDelete}
        variant="outline"
        className="!h-5 !w-5 !p-0 !rounded-full !bg-transparent !bg-white !border !border-gray-200 !text-gray-600 !hover:bg-gray-100"
        >
            <X size={10} />
        </Button>
    </ButtonEdge>
  );
}