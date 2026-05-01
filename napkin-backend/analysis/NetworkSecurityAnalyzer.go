package analysis

import "napkin-backend/graph"

type NetworkSecurityAnalyzer struct{}

func (a NetworkSecurityAnalyzer) Analyze(ig *graph.IntentGraph) ([]AnalysisError, []Annotation) {
	errorsList := []AnalysisError{}
	annotations := []Annotation{}

	nodes := ig.GetNodes()

	for _, node := range nodes {
		if _, exists := node.Data["vpcId"]; !exists {
			errorsList = append(errorsList, AnalysisError{
				NodeID:  node.ID,
				Message: "missing vpcId",
			})
		}
		if _, exists := node.Data["subnetId"]; !exists {
			errorsList = append(errorsList, AnalysisError{
				NodeID:  node.ID,
				Message: "missing subnetId",
			})
		}
		if _, exists := node.Data["securityGroupIds"]; !exists {
			errorsList = append(errorsList, AnalysisError{
				NodeID:  node.ID,
				Message: "missing securityGroupIds",
			})
		}
	}

	return errorsList, annotations
}
