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
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { AlertCircle, Check, Copy, X } from "lucide-react";
import { type AnalyzeError } from "@/lib/types/errors";
import { transformNodes } from "@/lib/transformer/transformer";
import { apiBase } from "@/lib/apiBase";

type IntentGraphPayload = ReturnType<typeof transformNodes>;

/** Per-node, per-field source string from the compiler's inheritance pass. */
export type InheritanceMap = Record<string, Record<string, string>>;

interface FloatingMenuProps {
  onAnalyze: (analysisType: string) => Promise<AnalyzeError[]>;
  intentGraph: IntentGraphPayload;
  /**
   * Receives the inheritance map after a successful compile. The parent owns
   * the state so node badges can be cleared when the user keeps editing.
   */
  onCompileResult?: (inherited: InheritanceMap) => void;
}

type CompileError = {
  severity: "error" | "warning";
  message: string;
};

export default function FloatingMenu({
  onAnalyze,
  intentGraph,
  onCompileResult,
}: FloatingMenuProps) {
  const [analysisType, setAnalysisType] = useState("Network");
  const [compileTarget, setCompileTarget] = useState("Terraform");
  const [compileErrors, setCompileErrors] = useState<CompileError[]>([]);
  const [analyzeErrors, setAnalyzeErrors] = useState<AnalyzeError[]>([]);
  const [analyzeVisible, setAnalyzeVisible] = useState(false);
  const [errorsVisible, setErrorsVisible] = useState(false);
  const [compiledOutput, setCompiledOutput] = useState<{
    output: string;
    target: string;
  } | null>(null);
  const [copyState, setCopyState] = useState<"idle" | "copied" | "failed">(
    "idle",
  );

  async function handleAnalyze() {
    setAnalyzeVisible(false);
    setAnalyzeErrors([]);

    const errors = await onAnalyze(analysisType);

    if (errors.length > 0) {
      setAnalyzeErrors(errors);
      setAnalyzeVisible(true);
    }
  }

  async function handleCompile() {
    try {
      const compileBody = {
        intentGraph,
        target: compileTarget.toLowerCase(),
      };
      const res = await fetch(`${apiBase()}/api/compile`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(compileBody),
      });
      const text = await res.text();
      let data: {
        errors?: CompileError[];
        success?: boolean;
        target?: string;
        output?: string;
        inherited?: InheritanceMap;
      } = {};
      if (text) {
        try {
          data = JSON.parse(text) as typeof data;
        } catch {
          /* Go http.Error bodies are plain text, not JSON */
        }
      }

      if (!res.ok || data.errors?.length) {
        const fallbackMsg = text.trim() || `Request failed (${res.status})`;
        setCompileErrors(
          data.errors?.length
            ? data.errors
            : [{ severity: "error", message: fallbackMsg }],
        );
        setErrorsVisible(true);
        setCompiledOutput(null);
        onCompileResult?.({});
      } else if (
        data.success &&
        typeof data.output === "string" &&
        data.output.length > 0
      ) {
        setCompiledOutput({
          output: data.output,
          target: data.target ?? compileTarget,
        });
        setCopyState("idle");
        onCompileResult?.(data.inherited ?? {});
      } else {
        setCompileErrors([
          {
            severity: "error",
            message: "Compile succeeded but returned no output.",
          },
        ]);
        setErrorsVisible(true);
        setCompiledOutput(null);
      }
    } catch (err) {
      setCompileErrors([
        { severity: "error", message: (err as Error).message },
      ]);
      setErrorsVisible(true);
      setCompiledOutput(null);
      onCompileResult?.({});
    }
  }

  async function handleCopyCompiled() {
    if (!compiledOutput) return;
    try {
      await navigator.clipboard.writeText(compiledOutput.output);
      setCopyState("copied");
      window.setTimeout(() => setCopyState("idle"), 2000);
    } catch {
      setCopyState("failed");
      window.setTimeout(() => setCopyState("idle"), 2000);
    }
  }

  return (
    <div className="absolute top-4 right-4 z-10 flex flex-col gap-2">
      <div className="flex gap-2 bg-white border border-gray-200 rounded-xl shadow-md p-3">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              size="lg"
              variant="outline"
              className="border-gray-200! bg-white! text-gray-700! hover:border-gray-400! hover:bg-white!"
            >
              {analysisType}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="translate-y-4">
            <DropdownMenuGroup>
              <DropdownMenuLabel>Analyze</DropdownMenuLabel>
              <DropdownMenuItem onSelect={() => setAnalysisType("Network")}>
                Network
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => setAnalysisType("Security")}>
                Security
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => setAnalysisType("Performance")}>
                Performance
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          size="lg"
          variant="outline"
          className="border-gray-200! bg-white! text-gray-700! hover:border-gray-400! hover:bg-white!"
          onClick={handleAnalyze}
        >
          Analyze
        </Button>

        <div className="w-px bg-gray-200 mx-1" />

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              size="lg"
              variant="outline"
              className="border-gray-200! bg-white! text-gray-700! hover:border-gray-400! hover:bg-white!"
            >
              {compileTarget}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="translate-y-4">
            <DropdownMenuGroup>
              <DropdownMenuLabel>Compile</DropdownMenuLabel>
              <DropdownMenuItem onSelect={() => setCompileTarget("Terraform")}>
                Terraform
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          size="lg"
          variant="outline"
          className="border-gray-200! bg-white! text-gray-700! hover:border-gray-400! hover:bg-white!"
          onClick={handleCompile}
        >
          Compile
        </Button>
      </div>

      {errorsVisible && compileErrors.length > 0 && (
        <div className="bg-white border border-gray-200 rounded-xl shadow-md p-3 flex flex-col gap-2 max-h-64 overflow-y-auto">
          <div className="flex items-center justify-between mb-1">
            <span className="text-xs font-medium text-gray-500 uppercase tracking-widest">
              {compileTarget} · {compileErrors.length} issue
              {compileErrors.length !== 1 ? "s" : ""}
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
      <Dialog
        open={compiledOutput !== null}
        onOpenChange={(open) => {
          if (!open) {
            setCompiledOutput(null);
            setCopyState("idle");
          }
        }}
      >
        <DialogContent
          showCloseButton={false}
          className="flex max-h-[min(85vh,720px)] max-w-[calc(100%-2rem)] flex-col gap-0 overflow-hidden p-0 sm:max-w-3xl"
        >
          {compiledOutput ? (
            <>
              <DialogDescription className="sr-only">
                Generated {compiledOutput.target} output from compile. Use the
                copy control to place it on the clipboard.
              </DialogDescription>
              <DialogHeader className="shrink-0 flex-row items-center gap-3 space-y-0 border-b px-4 py-3 text-left sm:text-left">
                <DialogTitle className="min-w-0 flex-1 text-sm font-semibold">
                  Compiled {compiledOutput.target}
                </DialogTitle>
                <div className="flex shrink-0 items-center gap-0.5">
                  <Button
                    type="button"
                    size="icon-sm"
                    variant="ghost"
                    className="text-muted-foreground hover:text-foreground"
                    onClick={handleCopyCompiled}
                    title={
                      copyState === "copied"
                        ? "Copied"
                        : copyState === "failed"
                          ? "Copy failed — try again"
                          : "Copy to clipboard"
                    }
                    aria-label={
                      copyState === "copied"
                        ? "Copied to clipboard"
                        : "Copy to clipboard"
                    }
                  >
                    {copyState === "copied" ? (
                      <Check className="size-4 text-green-600" aria-hidden />
                    ) : copyState === "failed" ? (
                      <AlertCircle
                        className="size-4 text-destructive"
                        aria-hidden
                      />
                    ) : (
                      <Copy className="size-4" aria-hidden />
                    )}
                  </Button>
                  <DialogClose asChild>
                    <Button
                      type="button"
                      size="icon-sm"
                      variant="ghost"
                      className="text-muted-foreground hover:text-foreground"
                      aria-label="Close"
                    >
                      <X className="size-4" aria-hidden />
                    </Button>
                  </DialogClose>
                </div>
              </DialogHeader>
              <pre className="m-0 min-h-0 flex-1 overflow-auto p-4 text-left font-mono text-xs leading-relaxed whitespace-pre text-gray-800">
                {compiledOutput.output}
              </pre>
            </>
          ) : null}
        </DialogContent>
      </Dialog>
      {analyzeVisible && analyzeErrors.length > 0 && (
        <div className="bg-white border border-gray-200 rounded-xl shadow-md p-3 flex flex-col gap-2 max-h-64 overflow-y-auto">
          <div className="flex items-center justify-between mb-1">
            <span className="text-xs font-medium text-gray-500 uppercase tracking-widest">
              Analyze · {analyzeErrors.length} issue
              {analyzeErrors.length !== 1 ? "s" : ""}
            </span>

            <button
              onClick={() => setAnalyzeVisible(false)}
              className="text-gray-400 hover:text-gray-600 text-sm"
            >
              ✕
            </button>
          </div>

          {analyzeErrors.map((err, i) => (
            <div
              key={i}
              className="rounded-lg border p-3 text-sm border-red-200 bg-red-50 text-red-800"
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
