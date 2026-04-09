package handlers

import (
	"encoding/json"
	"napkin-backend/analysis"
	"napkin-backend/graph"
	"net/http"
)

type AnalyzeRequest struct {
	IntentGraph json.RawMessage `json:"intentGraph"`
}

func AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AnalyzeRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.IntentGraph) == 0 {
		http.Error(w, "Missing intentGraph", http.StatusBadRequest)
		return
	}

	ig := graph.NewIntentGraph()
	err = ig.FromJSON(req.IntentGraph)
	if err != nil {
		http.Error(w, "Invalid graph format", http.StatusBadRequest)
		return
	}

	dg, err := ig.ToDirectedGraph()
	if err != nil {
		http.Error(w, "Invalid graph format", http.StatusBadRequest)
		return
	}

	networkAnalyzer := analysis.NetworkSecurityAnalyzer{}
	networkErrors, networkAnnotations := networkAnalyzer.Analyze(dg)

	performanceAnalyzer := analysis.PerformanceAnalyzer{}
	performanceErrors, performanceAnnotations := performanceAnalyzer.Analyze(dg)

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"success":                    true,
		"networkSecurityErrors":      networkErrors,
		"networkSecurityAnnotations": networkAnnotations,
		"performanceErrors":          performanceErrors,
		"performanceAnnotations":     performanceAnnotations,
	})
}
