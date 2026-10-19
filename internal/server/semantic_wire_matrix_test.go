package server

import (
	"context"
	"testing"

	"tops-lsp/internal/protocol"
)

func TestSemanticWireMatrixCoversAllV1Codes(t *testing.T) {
	resolved := &CompilationContext{LanguageStandard: "c++17", DriverKind: "topscc", CompilerContextStatus: "resolved", ArgumentProvenance: "test", TargetProfile: "gcu400", PassKind: "host+device"}
	device := &CompilationContext{LanguageStandard: "c++17", DriverKind: "topscc", CompilerContextStatus: "resolved", ArgumentProvenance: "test", TargetProfile: "gcu400", PassKind: "device"}
	cases := []struct {
		name     string
		language string
		text     string
		context  *CompilationContext
		code     string
		pass     string
	}{
		{name: "unknown-name", language: "tops", text: "int use() { return missing_name; }", context: resolved, code: "tops-semantic-unknown-name", pass: "not-applicable"},
		{name: "ambiguous-name", language: "tops", text: "int overload(int value); float overload(float value); int use() { return overload(1); }", context: resolved, code: "tops-semantic-ambiguous-name", pass: "not-applicable"},
		{name: "type-mismatch", language: "tops", text: "int invalid_value = \"text\";", context: resolved, code: "tops-semantic-type-mismatch", pass: "not-applicable"},
		{name: "invalid-call", language: "tops", text: "int value; int use() { return value(); }", context: resolved, code: "tops-semantic-invalid-call", pass: "device"},
		{name: "host-device-boundary", language: "tops", text: "__device__ int device_value() { return 1; } int host_call() { return device_value(); }", context: resolved, code: "tops-semantic-host-device-boundary", pass: "host"},
		{name: "address-space", language: "tops", text: "__device__ int *invalid_address(__global int *value) { return value; }", context: device, code: "tops-semantic-address-space", pass: "device"},
		{name: "vector-constraint", language: "tops", text: "__device__ int vector_value[3];", context: device, code: "tops-semantic-vector-constraint", pass: "device"},
		{name: "context-missing", language: "unknown", text: "int value;", context: nil, code: "tops-semantic-context-missing", pass: "not-applicable"},
		{name: "context-degraded", language: "tops", text: "int value;", context: nil, code: "tops-semantic-context-degraded", pass: "not-applicable"},
		{name: "incomplete-input", language: "tops", text: "int broken(", context: resolved, code: "tops-semantic-incomplete-input", pass: "not-applicable"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			server := NewWithOptions(nil, ServerOptions{CompilationContext: test.context})
			suffix := ".tops"
			if test.name == "context-missing" {
				suffix = ".txt"
			}
			state, err := server.openDocument("file:///workspace/wire-"+test.name+suffix, test.language, 1, test.text)
			if err != nil {
				t.Fatal(err)
			}
			publisher := &recordingDiagnosticPublisher{}
			server.parseAndPublish(context.Background(), publisher, state)
			var found *semanticDiagnosticView
			for _, params := range publisher.params {
				for _, diagnostic := range params.Diagnostics {
					if diagnostic.Code == test.code {
						value := semanticDiagnosticView{Code: diagnostic.Code, Severity: diagnostic.Severity, PassKind: "", DataPresent: diagnostic.Data != nil, RangeStart: diagnostic.Range.Start.Character, RangeEnd: diagnostic.Range.End.Character}
						if diagnostic.Data != nil {
							value.PassKind = diagnostic.Data.PassKind
						}
						found = &value
					}
				}
			}
			if found == nil {
				t.Fatalf("code %q not published; publications=%+v", test.code, publisher.params)
			}
			if !found.DataPresent || found.PassKind != test.pass || found.RangeEnd <= found.RangeStart {
				t.Fatalf("wire diagnostic = %+v", found)
			}
			if found.Severity == 0 {
				t.Fatalf("diagnostic severity is empty: %+v", found)
			}
		})
	}
}

type semanticDiagnosticView struct {
	Code        string
	Severity    protocol.DiagnosticSeverity
	PassKind    string
	DataPresent bool
	RangeStart  int
	RangeEnd    int
}
