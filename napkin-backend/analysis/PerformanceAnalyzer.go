package analysis

import "napkin-backend/graph"

type PerformanceAnalyzer struct{}

func (a PerformanceAnalyzer) Analyze(g *graph.DirectedGraph) ([]AnalysisError, []Annotation) {
	return []AnalysisError{}, []Annotation{}
}
