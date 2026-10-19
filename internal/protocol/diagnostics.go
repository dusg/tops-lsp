package protocol

import "encoding/json"

type DiagnosticSeverity uint8

const (
	DiagnosticSeverityError   DiagnosticSeverity = 1
	DiagnosticSeverityWarning DiagnosticSeverity = 2
	DiagnosticSeverityInfo    DiagnosticSeverity = 3
	DiagnosticSeverityHint    DiagnosticSeverity = 4
)

type Diagnostic struct {
	Range              Range                          `json:"range"`
	Severity           DiagnosticSeverity             `json:"severity,omitempty"`
	Code               string                         `json:"code,omitempty"`
	Source             string                         `json:"source,omitempty"`
	Message            string                         `json:"message"`
	Data               *DiagnosticData                `json:"data,omitempty"`
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`
}

type DiagnosticData struct {
	ConditionalState      string                 `json:"conditionalState,omitempty"`
	ConditionalProvenance *ConditionalProvenance `json:"conditionalProvenance,omitempty"`
	Recoverable           bool                   `json:"recoverable"`
	Incomplete            bool                   `json:"incomplete"`
	DocumentVersion       int                    `json:"documentVersion,omitempty"`
	ContextVersion        int                    `json:"contextVersion,omitempty"`
	SourceVersion         string                 `json:"sourceVersion,omitempty"`
	ContextStatus         string                 `json:"contextStatus,omitempty"`
	ProfileID             string                 `json:"profileId,omitempty"`
	PassKind              string                 `json:"passKind,omitempty"`
	SemanticKind          string                 `json:"semanticKind,omitempty"`
	Reason                string                 `json:"reason,omitempty"`
	ErrorKind             string                 `json:"errorKind,omitempty"`
}

type ConditionalProvenance struct {
	SourceURI        string                  `json:"sourceURI,omitempty"`
	RegionID         int                     `json:"regionId"`
	BranchIndex      int                     `json:"branchIndex"`
	ExpressionRange  *Range                  `json:"expressionRange,omitempty"`
	State            string                  `json:"state,omitempty"`
	ParentRegionID   int                     `json:"parentRegionId"`
	ConditionTokens  []string                `json:"conditionTokens,omitempty"`
	ParentConditions []ConditionalProvenance `json:"parentConditions,omitempty"`
}

type DiagnosticRelatedInformation struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type PublishDiagnosticsParams struct {
	URI            string       `json:"uri"`
	Version        *int         `json:"version,omitempty"`
	ContextVersion *int         `json:"contextVersion,omitempty"`
	Diagnostics    []Diagnostic `json:"diagnostics"`
}

func MarshalNotification(method string, params any) ([]byte, error) {
	return json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
}
