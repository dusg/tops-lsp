package semantic

func CheckCallBoundary(caller, callee CallableFact, pass PassKind, context SemanticContext) (SemanticDiagnostic, bool) {
	view, ok := context.PassViews[pass]
	if !ok || view.Availability != PassAvailabilityAvailable || !passViewHasProvenance(context, view) || context.Status != ContextStatusResolved || !context.Profile.TargetRules {
		return SemanticDiagnostic{}, false
	}
	if !callableSupportsPass(caller, pass) {
		return SemanticDiagnostic{}, false
	}
	invalid := !callableSupportsPass(callee, pass)
	if pass == PassKindHost {
		invalid = invalid || (isHostCallable(caller.ExecutionKind) && isDeviceOnlyCallable(callee.ExecutionKind))
	}
	if pass == PassKindDevice {
		invalid = invalid || (isDeviceCallable(caller.ExecutionKind) && isHostOnlyCallable(callee.ExecutionKind))
	}
	if !invalid {
		return SemanticDiagnostic{}, false
	}
	rangeValue := caller.Range
	if rangeValue.Start == rangeValue.End {
		rangeValue = callee.Range
	}
	return SemanticDiagnostic{
		Code:            "tops-semantic-host-device-boundary",
		Severity:        SeverityError,
		SemanticKind:    "host-device-boundary",
		Message:         "call crosses an unavailable host/device boundary",
		Range:           rangeValue,
		PassKind:        string(pass),
		ProfileID:       context.Profile.ProfileID,
		ContextStatus:   context.Status,
		DocumentVersion: context.DocumentVersion,
		ContextVersion:  context.SemanticContextVersion,
		SourceVersion:   context.SourceVersionFingerprint,
	}, true
}

func passViewHasProvenance(context SemanticContext, view PassView) bool {
	return view.Provenance["source"] != "" || context.Command.Source == "default"
}

func callableSupportsPass(callable CallableFact, pass PassKind) bool {
	if len(callable.PassKinds) == 0 {
		return true
	}
	for _, available := range callable.PassKinds {
		if available == pass {
			return true
		}
	}
	return false
}

func isHostCallable(kind string) bool {
	return kind == "host" || kind == "host-device"
}

func isDeviceCallable(kind string) bool {
	return kind == "device" || kind == "host-device" || kind == "kernel-entry"
}

func isDeviceOnlyCallable(kind string) bool {
	return kind == "device" || kind == "kernel-entry"
}

func isHostOnlyCallable(kind string) bool {
	return kind == "host"
}
