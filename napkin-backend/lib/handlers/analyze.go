package handlers

import (
	"encoding/json"
	"napkin-backend/analysis"
	"napkin-backend/graph"
	"net/http"
)

type AnalyzeRequest struct {
	Type  string          `json:"type"`
	Graph json.RawMessage `json:"graph"`
}

type ResponseError struct {
	NodeId  string `json:"nodeId"`
	Message string `json:"message"`
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

	ig := graph.NewIntentGraph()

	err = ig.FromJSON(req.Graph)
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

	allErrors := make([]analysis.AnalysisError, 0, len(networkErrors)+len(performanceErrors))
	allErrors = append(allErrors, networkErrors...)
	allErrors = append(allErrors, performanceErrors...)

	allAnnotations := make([]analysis.Annotation, 0, len(networkAnnotations)+len(performanceAnnotations))
	allAnnotations = append(allAnnotations, networkAnnotations...)
	allAnnotations = append(allAnnotations, performanceAnnotations...)

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"success":     true,
		"errors":      allErrors,
		"annotations": allAnnotations,
	})
}

func TestAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
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

	ig := graph.NewIntentGraph()

	err = ig.FromJSON(req.Graph)
	if err != nil {
		http.Error(w, "Invalid graph format", http.StatusBadRequest)
		return
	}

	var allErrors []ResponseError

	for _, node := range ig.Nodes {
		spec, ok := node.Spec.(map[string]any)
		if ok && spec["label"] == "EC2 Instance" {
			allErrors = append(allErrors, ResponseError{
				NodeId:  string(node.ID),
				Message: "Security Risk: Instance is publicly accessible!",
			})
		}

		if ok && spec["label"] == "Database" {
			allErrors = append(allErrors, ResponseError{
				NodeId:  string(node.ID),
				Message: "Performance Warning: High latency detected.",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"errors":  allErrors,
	})
}
