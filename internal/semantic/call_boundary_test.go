package semantic

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tops-lsp/internal/parser"
)

func TestCheckCallBoundarySeparatesHostAndDevicePasses(t *testing.T) {
	context := semanticTestContext(ContextStatusResolved)
	host := CallableFact{SymbolID: "host", ExecutionKind: "host", PassKinds: []PassKind{PassKindHost}}
	device := CallableFact{SymbolID: "device", ExecutionKind: "device", PassKinds: []PassKind{PassKindDevice}}
	if diagnostic, ok := CheckCallBoundary(host, device, PassKindHost, context); !ok || diagnostic.Code != "tops-semantic-host-device-boundary" || diagnostic.PassKind != string(PassKindHost) {
		t.Fatalf("host pass boundary = %+v, ok = %v", diagnostic, ok)
	}
	if _, ok := CheckCallBoundary(host, device, PassKindDevice, context); ok {
		t.Fatal("device pass incorrectly reported host caller boundary")
	}
	shared := CallableFact{SymbolID: "shared", ExecutionKind: "host-device", PassKinds: []PassKind{PassKindHost, PassKindDevice}}
	if _, ok := CheckCallBoundary(host, shared, PassKindHost, context); ok {
		t.Fatal("host-device callee was rejected on host pass")
	}
}

func TestCheckCallBoundaryHandlesHostDeviceAndKernelEntryCallers(t *testing.T) {
	context := semanticTestContext(ContextStatusResolved)
	shared := CallableFact{SymbolID: "shared", ExecutionKind: "host-device", PassKinds: []PassKind{PassKindHost, PassKindDevice}}
	device := CallableFact{SymbolID: "device", ExecutionKind: "device", PassKinds: []PassKind{PassKindDevice}}
	if _, ok := CheckCallBoundary(shared, device, PassKindHost, context); !ok {
		t.Fatal("host-device caller did not reject device callee on host pass")
	}
	if _, ok := CheckCallBoundary(shared, device, PassKindDevice, context); ok {
		t.Fatal("host-device caller rejected device callee on device pass")
	}
	kernel := CallableFact{SymbolID: "kernel", ExecutionKind: "kernel-entry", PassKinds: []PassKind{PassKindDevice}}
	host := CallableFact{SymbolID: "host", ExecutionKind: "host", PassKinds: []PassKind{PassKindHost}}
	if _, ok := CheckCallBoundary(kernel, host, PassKindDevice, context); !ok {
		t.Fatal("kernel-entry caller did not reject host callee on device pass")
	}
}

func TestCheckCallBoundarySuppressesTargetErrorWhenContextIsIncomplete(t *testing.T) {
	context := semanticTestContext(ContextStatusPartial)
	host := CallableFact{SymbolID: "host", ExecutionKind: "host"}
	device := CallableFact{SymbolID: "device", ExecutionKind: "device"}
	if _, ok := CheckCallBoundary(host, device, PassKindHost, context); ok {
		t.Fatal("partial context produced a deterministic boundary error")
	}
	hostView := context.PassViews[PassKindHost]
	hostView.Availability = PassAvailabilityMissing
	context.PassViews[PassKindHost] = hostView
	context.Status = ContextStatusResolved
	if _, ok := CheckCallBoundary(host, device, PassKindHost, context); ok {
		t.Fatal("missing pass view produced a deterministic boundary error")
	}
}

func TestPassViewAvailabilityCombinationsRemainIndependent(t *testing.T) {
	host := CallableFact{SymbolID: "host", ExecutionKind: "host", Range: parser.SourceRange{Start: 0, End: 1}}
	device := CallableFact{SymbolID: "device", ExecutionKind: "device", Range: parser.SourceRange{Start: 0, End: 1}}
	for _, passKind := range []PassKind{PassKindHost, PassKindDevice} {
		context := semanticTestContext(ContextStatusResolved)
		if passKind == PassKindHost {
			context.Command.PassKind = PassKindHost
			context.PassViews = NewPassViews(PassKindHost, ContextStatusResolved)
		} else {
			context.Command.PassKind = PassKindDevice
			context.PassViews = NewPassViews(PassKindDevice, ContextStatusResolved)
		}
		if passKind == PassKindHost {
			if _, ok := CheckCallBoundary(host, device, passKind, context); !ok {
				t.Fatal("host-only view lost host boundary")
			}
		} else if _, ok := CheckCallBoundary(host, device, passKind, context); ok {
			t.Fatal("device-only view reported host boundary")
		}
	}
	context := semanticTestContext(ContextStatusPartial)
	context.PassViews[PassKindHost] = PassView{PassKind: PassKindHost, Availability: PassAvailabilityMissing, Status: ContextStatusPartial}
	context.PassViews[PassKindDevice] = PassView{PassKind: PassKindDevice, Availability: PassAvailabilityAvailable, Status: ContextStatusPartial}
	if _, ok := CheckCallBoundary(host, device, PassKindHost, context); ok {
		t.Fatal("single missing view produced a boundary error")
	}
}

func TestCallableFixtureGridReadsAllCanonicalKindsAndScenarios(t *testing.T) {
	manifest := loadRepositoryFixtureManifest(t)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..")
	seen := make(map[string]struct{})
	for _, fixture := range manifest.Cases {
		if fixture.Capability != "callables" {
			continue
		}
		key := fixture.CallableKind + "/" + fixture.Scenario
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("duplicate callable fixture %q", key)
		}
		seen[key] = struct{}{}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(fixture.SourceFiles[0]))); err != nil {
			t.Fatalf("callable fixture %q: %v", key, err)
		}
	}
	if len(seen) != 16 {
		t.Fatalf("callable fixture grid size = %d", len(seen))
	}
}
