package semantic

import (
	"fmt"

	"tops-lsp/internal/parser"
)

type ContextStatus string

const (
	ContextStatusResolved ContextStatus = "resolved"
	ContextStatusPartial  ContextStatus = "partial"
	ContextStatusMissing  ContextStatus = "missing"
	ContextStatusInvalid  ContextStatus = "invalid"
)

type PassAvailability string

const (
	PassAvailabilityAvailable     PassAvailability = "available"
	PassAvailabilityNotApplicable PassAvailability = "not-applicable"
	PassAvailabilityMissing       PassAvailability = "missing"
)

type PassView struct {
	PassKind       PassKind
	Availability   PassAvailability
	Status         ContextStatus
	RulesAvailable []string
	Provenance     map[string]string
}

type SemanticContext struct {
	DocumentURI              string
	Command                  CompilationCommand
	Profile                  SemanticProfile
	Status                   ContextStatus
	DocumentVersion          int
	SemanticContextVersion   int
	SourceVersions           []SourceVersionEntry
	HeaderVersions           []SourceVersionEntry
	AnalysisGeneration       uint64
	Epoch                    uint64
	PassViews                map[PassKind]PassView
	ConditionalMacros        map[string]string
	MissingFields            []string
	Conflicts                []string
	Provenance               map[string]string
	SourceVersionFingerprint string
	ParseStatus              parser.ParseStatus
	IndexStatus              string
	IncompleteRange          parser.SourceRange
}

func (context SemanticContext) Validate() error {
	if context.DocumentURI == "" {
		return fmt.Errorf("semantic context document URI is empty")
	}
	if context.Profile.ProfileID == "" {
		return fmt.Errorf("semantic context profile ID is empty")
	}
	switch context.Status {
	case ContextStatusResolved, ContextStatusPartial, ContextStatusMissing, ContextStatusInvalid:
	default:
		return fmt.Errorf("invalid semantic context status %q", context.Status)
	}
	if context.Command.PassKind != "" && context.Command.PassKind != PassKindHost && context.Command.PassKind != PassKindDevice && context.Command.PassKind != PassKindHostDevice {
		return fmt.Errorf("invalid semantic command pass kind %q", context.Command.PassKind)
	}
	for passKind, view := range context.PassViews {
		if passKind != PassKindHost && passKind != PassKindDevice {
			return fmt.Errorf("invalid semantic pass view %q", passKind)
		}
		if view.PassKind != passKind {
			return fmt.Errorf("pass view %q has mismatched pass kind %q", passKind, view.PassKind)
		}
		switch view.Availability {
		case PassAvailabilityAvailable, PassAvailabilityNotApplicable, PassAvailabilityMissing:
		default:
			return fmt.Errorf("invalid pass availability %q", view.Availability)
		}
	}
	return nil
}

func NewPassViews(passKind PassKind, status ContextStatus) map[PassKind]PassView {
	views := make(map[PassKind]PassView)
	switch passKind {
	case PassKindHost:
		views[PassKindHost] = PassView{PassKind: PassKindHost, Availability: PassAvailabilityAvailable, Status: status}
		views[PassKindDevice] = PassView{PassKind: PassKindDevice, Availability: PassAvailabilityNotApplicable, Status: status}
	case PassKindDevice:
		views[PassKindHost] = PassView{PassKind: PassKindHost, Availability: PassAvailabilityNotApplicable, Status: status}
		views[PassKindDevice] = PassView{PassKind: PassKindDevice, Availability: PassAvailabilityAvailable, Status: status}
	case PassKindHostDevice:
		views[PassKindHost] = PassView{PassKind: PassKindHost, Availability: PassAvailabilityAvailable, Status: status}
		views[PassKindDevice] = PassView{PassKind: PassKindDevice, Availability: PassAvailabilityAvailable, Status: status}
	}
	return views
}
