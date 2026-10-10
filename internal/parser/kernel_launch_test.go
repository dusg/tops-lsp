package parser

import "testing"

func TestParseBuildsKernelLaunchNode(t *testing.T) {
	result := Parse("__host__ void launch(dim3 grid, dim3 block, int* output) { kernel<<<grid, block>>>(output); }", ParseContext{})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v", result.Diagnostics)
	}
	launch := findNode(result.Root, NodeTopsLaunch)
	if launch == nil || launch.Range.Start >= launch.Range.End {
		t.Fatalf("launch = %+v", launch)
	}
	if launch.Callee != "kernel" || launch.Name != "kernel" || len(launch.ConfigTokens) != 3 || len(launch.ArgumentTokens) != 1 {
		t.Fatalf("launch fields = %+v", launch)
	}
}

func TestParseBuildsTemplatedKernelLaunchCallee(t *testing.T) {
	result := Parse("kernel<T><<<grid, block>>>(value);", ParseContext{LanguageStandard: "c++17"})
	launch := findNode(result.Root, NodeTopsLaunch)
	if launch == nil || launch.Callee != "kernel<T>" || len(launch.ConfigTokens) != 3 || len(launch.ArgumentTokens) != 1 {
		t.Fatalf("templated launch = %+v diagnostics=%+v", launch, result.Diagnostics)
	}
}

func TestParseIncompleteLaunchReportsOpeningDelimiter(t *testing.T) {
	result := Parse("kernel<<<grid,", ParseContext{})
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code != DiagnosticMissingToken || diagnostic.Message != "expected '>>>'" {
			continue
		}
		if len(diagnostic.Related) != 1 || diagnostic.Related[0].Range != (SourceRange{Start: 6, End: 9}) {
			t.Fatalf("launch related location = %+v", diagnostic.Related)
		}
		return
	}
	t.Fatalf("incomplete launch diagnostic = %+v", result.Diagnostics)
}

func TestParseLaunchRequiresCallParentheses(t *testing.T) {
	result := Parse("kernel<<<grid>>>;", ParseContext{LanguageStandard: "c++17"})
	launch := findNode(result.Root, NodeTopsLaunch)
	if launch == nil || !launch.Incomplete || findNode(launch, NodeMissingToken) == nil {
		t.Fatalf("launch recovery = %+v", launch)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == DiagnosticMissingToken && diagnostic.Message == "expected '(' after kernel launch configuration" {
			return
		}
	}
	t.Fatalf("missing call-parenthesis diagnostic = %+v", result.Diagnostics)
}

func TestParseKernelLaunchRecoveryStatesAreExclusive(t *testing.T) {
	cases := []struct {
		name           string
		source         string
		wantMessage    string
		forbidMessages []string
	}{
		{name: "complete", source: "kernel<<<grid>>>(arg);"},
		{name: "missing configuration close", source: "kernel<<<grid,", wantMessage: "expected '>>>'"},
		{name: "missing call open", source: "kernel<<<grid>>>;", wantMessage: "expected '(' after kernel launch configuration", forbidMessages: []string{"expected '>>>'", "expected ')'"}},
		{name: "missing call close", source: "kernel<<<grid>>>(arg;", wantMessage: "expected ')' after kernel launch arguments", forbidMessages: []string{"expected '>>>'", "expected ')'"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := Parse(testCase.source, ParseContext{LanguageStandard: "c++17"})
			counts := make(map[string]int)
			for _, diagnostic := range result.Diagnostics {
				if diagnostic.Code == DiagnosticMissingToken {
					counts[diagnostic.Message]++
				}
			}
			if testCase.wantMessage != "" && counts[testCase.wantMessage] != 1 {
				t.Fatalf("diagnostics = %+v, want one %q", result.Diagnostics, testCase.wantMessage)
			}
			for _, forbidMessage := range testCase.forbidMessages {
				if counts[forbidMessage] != 0 {
					t.Fatalf("diagnostics = %+v, must not contain %q", result.Diagnostics, forbidMessage)
				}
			}
			if testCase.wantMessage == "" && len(counts) != 0 {
				t.Fatalf("complete launch diagnostics = %+v", result.Diagnostics)
			}
		})
	}
}
