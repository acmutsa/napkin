package analysis

import (
	"napkin-backend/graph"
)

type AnalysisError struct {
	NodeID  graph.NodeID `json:"nodeId"`
	Message string       `json:"message"`
}

type Annotation struct {
	NodeID graph.NodeID `json:"nodeId"`
	Label  string       `json:"label"`
	Value  string       `json:"value"`
}

type Analyzer interface {
	Analyze(IntentGraph *graph.DirectedGraph) ([]AnalysisError, []Annotation)
}
