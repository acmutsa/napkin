package analysis

import (
	"napkin-backend/graph"
)

type AnalysisError struct {
	NodeID  graph.NodeID
	Message string
}

type Annotation struct {
	NodeID graph.NodeID
	Label  string
	Value  string
}

type Analyzer interface {
	Analyze(IntentGraph *graph.DirectedGraph) ([]AnalysisError, []Annotation)
}
