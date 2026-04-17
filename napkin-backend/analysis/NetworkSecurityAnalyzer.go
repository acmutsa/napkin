package analysis

import "napkin-backend/graph"

type NetworkSecurityAnalyzer struct{}

func (a NetworkSecurityAnalyzer) Analyze(g *graph.DirectedGraph) ([]AnalysisError, []Annotation) {
	return []AnalysisError{}, []Annotation{}
}
