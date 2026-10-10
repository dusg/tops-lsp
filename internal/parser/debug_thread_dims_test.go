package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDebugThreadDimsDiagnostics(t *testing.T) {
	path := filepath.Join("/home/carl.du/work/topsop/topsop/lib/kernel/cc_kernel", "topp_renorm_probs/topp_renorm_probs_kernel_gcu400.tops")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result := Parse(string(source), ParseContext{LanguageID: "tops", LanguageStandard: "c++17", DriverKind: "topscc", CompilerContextStatus: "partial", ArgumentProvenance: "cc-kernel-manifest", TargetProfile: "gcu400", PassKind: "device", DocumentVersion: 1, ContextVersion: 1})
	for _, diagnostic := range result.Diagnostics {
		t.Logf("code=%s message=%q range=%+v", diagnostic.Code, diagnostic.Message, diagnostic.Range)
	}
}
