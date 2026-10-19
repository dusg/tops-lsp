package semantic

import (
	"fmt"

	"tops-lsp/internal/index"
	"tops-lsp/internal/parser"
)

type SnapshotStatus string

const (
	SnapshotStatusComplete SnapshotStatus = "complete"
	SnapshotStatusDegraded SnapshotStatus = "degraded"
	SnapshotStatusStale    SnapshotStatus = "stale"
)

type SemanticSnapshot struct {
	Context                  SemanticContext
	ProfileID                string
	ContextStatus            ContextStatus
	SourceVersionFingerprint string
	DocumentVersion          int
	SemanticContextVersion   int
	SourceVersions           []SourceVersionEntry
	HeaderVersions           []SourceVersionEntry
	Generation               uint64
	Epoch                    uint64
	Status                   SnapshotStatus
	UnsupportedReason        string
	InvalidReason            string
	StaleReason              string
	Declarations             []DeclarationFact
	Scopes                   []ScopeFact
	Types                    map[string]SemanticType
	Callables                []CallableFact
	References               []ReferenceFact
	Calls                    []CallFact
	Diagnostics              []SemanticDiagnostic
}

func (snapshot SemanticSnapshot) Validate() error {
	if err := snapshot.Context.Validate(); err != nil {
		return err
	}
	if snapshot.ProfileID == "" {
		return fmt.Errorf("semantic snapshot profile ID is empty")
	}
	if snapshot.ProfileID != snapshot.Context.Profile.ProfileID {
		return fmt.Errorf("semantic snapshot profile ID does not match context")
	}
	if snapshot.ContextStatus != snapshot.Context.Status {
		return fmt.Errorf("semantic snapshot context status does not match context")
	}
	if snapshot.SourceVersionFingerprint == "" {
		return fmt.Errorf("semantic snapshot source version fingerprint is empty")
	}
	if err := snapshot.ValidateSourceVersion(); err != nil {
		return err
	}
	switch snapshot.Status {
	case SnapshotStatusComplete, SnapshotStatusDegraded:
		return nil
	case SnapshotStatusStale:
		return fmt.Errorf("stale semantic snapshot cannot be committed")
	default:
		return fmt.Errorf("invalid semantic snapshot status %q", snapshot.Status)
	}
}

func (snapshot SemanticSnapshot) ValidateSourceVersion() error {
	rootURI := index.NormalizeSourceURI(snapshot.Context.DocumentURI)
	if rootURI == "" {
		return fmt.Errorf("semantic snapshot document URI is not normalizable")
	}
	entries := append([]SourceVersionEntry(nil), snapshot.SourceVersions...)
	entries = append(entries, snapshot.HeaderVersions...)
	fingerprint, err := AggregateSourceVersion(snapshot.Context.DocumentURI, entries)
	if err != nil {
		return err
	}
	if fingerprint != snapshot.SourceVersionFingerprint {
		return fmt.Errorf("semantic snapshot source version fingerprint does not match source versions")
	}
	return nil
}

type SemanticSeverity string

const (
	SeverityError       SemanticSeverity = "Error"
	SeverityWarning     SemanticSeverity = "Warning"
	SeverityInformation SemanticSeverity = "Information"
	SeverityHint        SemanticSeverity = "Hint"
)

type DeclarationFact struct {
	SymbolID      string
	Name          string
	QualifiedName string
	Kind          index.SymbolKind
	Type          SemanticType
	ScopeID       string
	Range         parser.SourceRange
	Status        index.SymbolStatus
	Condition     index.ConditionalProvenance
	SourceVersion index.SourceVersion
}

type ScopeFact struct {
	ScopeID       string
	SourceURI     string
	Kind          index.ScopeKind
	Name          string
	QualifiedName string
	ParentID      string
	Range         parser.SourceRange
	Symbols       []string
	Children      []string
}

type ReferenceFact struct {
	ReferenceID   string
	URI           string
	Range         parser.SourceRange
	Name          string
	QualifiedName string
	ScopeID       string
	Candidates    []string
	Status        index.ReferenceStatus
	Kind          string
	Condition     index.ConditionalProvenance
	SourceVersion index.SourceVersion
}

type CandidateStatus string

const (
	CandidateStatusResolved   CandidateStatus = "resolved"
	CandidateStatusAmbiguous  CandidateStatus = "ambiguous"
	CandidateStatusUnresolved CandidateStatus = "unresolved"
)

type CallableParameter struct {
	Name string
	Type SemanticType
}

type CallableFact struct {
	SymbolID              string
	Name                  string
	QualifiedName         string
	ScopeID               string
	Parameters            []CallableParameter
	ReturnType            SemanticType
	DeclarationOnly       bool
	ExecutionKind         string
	DefaultExecutionSpace bool
	PassKinds             []PassKind
	Provenance            map[string]string
	Range                 parser.SourceRange
}

type CallConversion struct {
	SourceType SemanticType
	TargetType SemanticType
	Status     ConversionStatus
	Kind       ConversionKind
}

type CallPassFact struct {
	PassKind       PassKind
	Availability   PassAvailability
	CallerSymbolID string
	CalleeSymbolID string
	CallerKind     string
	CalleeKind     string
}

type CallFact struct {
	CallID          string
	URI             string
	Range           parser.SourceRange
	CalleeName      string
	Candidates      []string
	CandidateStatus CandidateStatus
	ArgumentCount   int
	ArgumentsKnown  bool
	CallerSymbolID  string
	CalleeSymbolID  string
	CallerKind      string
	CalleeKind      string
	PassKind        PassKind
	PassFacts       []CallPassFact
	Conversions     []CallConversion
	Condition       index.ConditionalProvenance
}

type SemanticRelatedInformation struct {
	URI     string
	Range   parser.SourceRange
	Message string
}

type SemanticDiagnostic struct {
	Code                  string
	Severity              SemanticSeverity
	SemanticKind          string
	Message               string
	Range                 parser.SourceRange
	PassKind              string
	ProfileID             string
	ContextStatus         ContextStatus
	DocumentVersion       int
	ContextVersion        int
	SourceVersion         string
	ConditionalProvenance index.ConditionalProvenance
	Candidates            []string
	Related               []SemanticRelatedInformation
	Recoverable           bool
	Incomplete            bool
}
