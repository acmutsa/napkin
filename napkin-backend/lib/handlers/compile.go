package handlers

import (
	"encoding/json"
	"napkin-backend/compiler"
	"napkin-backend/graph"
	"net/http"
)

type CompileRequest struct {
	IntentGraph json.RawMessage `json:"intentGraph"`
	Target      string          `json:"target"`
}

func CompileHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CompileRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.IntentGraph) == 0 {
		http.Error(w, "Missing intentGraph", http.StatusBadRequest)
		return
	}

	if req.Target == "" {
		http.Error(w, "Missing target", http.StatusBadRequest)
		return
	}

	if req.Target != "terraform" {
		http.Error(w, "Unsupported target", http.StatusBadRequest)
		return
	}

	ig := graph.NewIntentGraph()
	err = ig.FromJSON(req.IntentGraph)
	if err != nil {
		http.Error(w, "Invalid graph format", http.StatusBadRequest)
		return
	}

	ir, err := IntentGraphToIR(ig)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	target := &compiler.TerraformTarget{}
	tfFile, err := target.Compile(ir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	output := tfFile.String()

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"target":  req.Target,
		"output":  output,
	})

}
