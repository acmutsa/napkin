import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface FloatingMenuProps {
  onAnalyzeError: (nodeId: string | null, message: string) => void;
}

type CompileError = {
  severity: "error" | "warning";
  message: string;
};

export default function FloatingMenu({ onAnalyzeError }: FloatingMenuProps) {
  const [analysisType, setAnalysisType] = useState("Network");
  const [compileTarget, setCompileTarget] = useState("Terraform");
  const [compileErrors, setCompileErrors] = useState<CompileError[]>([]);
  const [errorsVisible, setErrorsVisible] = useState(false);

  async function handleAnalyze() {
    try {
      const res = await fetch("/analyze", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ type: analysisType }),
      });
      const data = await res.json();
      data.errors?.forEach(({ nodeId, message }: { nodeId: string | null; message: string }) => {
        onAnalyzeError(nodeId, message);
      });
    } catch (err) {
      onAnalyzeError(null, (err as Error).message);
    }
  }

  async function handleCompile() {
    try {
      const res = await fetch("/compile", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target: compileTarget }),
      });
      const data = await res.json();
      if (!res.ok || data.errors?.length) {
        setCompileErrors(data.errors ?? [{ severity: "error", message: "Unknown error." }]);
        setErrorsVisible(true);
      }
    } catch (err) {
      setCompileErrors([{ severity: "error", message: (err as Error).message }]);
      setErrorsVisible(true);
    }
  }

  return (
    <div className="absolute top-4 right-4 z-10 flex flex-col gap-2">
      <div className="flex gap-2 bg-white border border-gray-200 rounded-xl shadow-md p-3">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="lg" variant="outline" className="border-gray-200! bg-white! text-gray-700! hover:border-gray-400! hover:bg-white!">
              {analysisType}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="translate-y-4">
            <DropdownMenuGroup>
              <DropdownMenuLabel>Analyze</DropdownMenuLabel>
              <DropdownMenuItem onSelect={() => setAnalysisType("Network")}>Network</DropdownMenuItem>
              <DropdownMenuItem onSelect={() => setAnalysisType("Security")}>Security</DropdownMenuItem>
              <DropdownMenuItem onSelect={() => setAnalysisType("Performance")}>Performance</DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button size="lg" variant="outline" className="border-gray-200! bg-white! text-gray-700! hover:border-gray-400! hover:bg-white!" onClick={handleAnalyze}>
          Analyze
        </Button>

        <div className="w-px bg-gray-200 mx-1" />

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="lg" variant="outline" className="border-gray-200! bg-white! text-gray-700! hover:border-gray-400! hover:bg-white!">
              {compileTarget}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="translate-y-4">
            <DropdownMenuGroup>
              <DropdownMenuLabel>Compile</DropdownMenuLabel>
              <DropdownMenuItem onSelect={() => setCompileTarget("Terraform")}>Terraform</DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button size="lg" variant="outline" className="border-gray-200! bg-white! text-gray-700! hover:border-gray-400! hover:bg-white!" onClick={handleCompile}>
          Compile
        </Button>
      </div>

      {errorsVisible && compileErrors.length > 0 && (
        <div className="bg-white border border-gray-200 rounded-xl shadow-md p-3 flex flex-col gap-2 max-h-64 overflow-y-auto">
          <div className="flex items-center justify-between mb-1">
            <span className="text-xs font-medium text-gray-500 uppercase tracking-widest">
              {compileTarget} · {compileErrors.length} issue{compileErrors.length !== 1 ? "s" : ""}
            </span>
            <button
              onClick={() => setErrorsVisible(false)}
              className="text-gray-400 hover:text-gray-600 bg-white! border-gray-400! text-sm leading-none"
            >
              ✕
            </button>
          </div>
          {compileErrors.map((err, i) => (
            <div
              key={i}
              className={`rounded-lg border p-3 text-sm ${
                err.severity === "warning"
                  ? "border-yellow-200 bg-yellow-50 text-yellow-800"
                  : "border-red-200 bg-red-50 text-red-800"
              }`}
            >
              <span className="font-medium capitalize">{err.severity}: </span>
              {err.message}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}