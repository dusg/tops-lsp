package semantic

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func comparisonClangInput(input ComparisonCase, pass PassKind, rootURI string) (string, []string) {
	clangPath := strings.TrimSpace(os.Getenv("TOPS_LSP_CLANG"))
	if clangPath == "" {
		return "skipped", nil
	}
	parsed, err := url.Parse(rootURI)
	if err != nil || parsed.Scheme != "file" || parsed.Path == "" {
		return "not-comparable", nil
	}
	arguments := []string{"-fsyntax-only", "-fdiagnostics-color=never"}
	arguments = append(arguments, comparisonClangArguments(input, pass, rootURI)...)
	arguments = append(arguments, filepath.FromSlash(parsed.Path))
	commandContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(commandContext, clangPath, arguments...)
	if input.WorkingDirectory != "" {
		command.Dir = input.WorkingDirectory
	}
	output, commandErr := command.CombinedOutput()
	categories := clangDiagnosticCategories(output)
	if commandErr != nil {
		return "failed", categories
	}
	return "available", categories
}

func comparisonClangArguments(input ComparisonCase, pass PassKind, rootURI string) []string {
	arguments := input.NormalizedArguments
	if len(arguments) == 0 {
		arguments = input.RawArguments
	}
	result := make([]string, 0, len(arguments)+8)
	sourcePath := ""
	if parsed, err := url.Parse(rootURI); err == nil {
		sourcePath = filepath.FromSlash(parsed.Path)
	}
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		if index == 0 && !strings.HasPrefix(argument, "-") {
			continue
		}
		if argument == rootURI || argument == sourcePath || filepath.Clean(argument) == filepath.Clean(sourcePath) {
			continue
		}
		if argument == "-o" || argument == "--output" {
			index++
			continue
		}
		result = append(result, argument)
	}
	result = append(result, comparisonProvenanceArguments(input.IncludeProvenance, "-I")...)
	result = append(result, comparisonProvenanceArguments(input.MacroProvenance, "-D")...)
	if input.ProfileID != "" {
		result = append(result, "-D__TOPS_LSP_PROFILE_ID="+input.ProfileID)
	}
	if input.PassKind != "" {
		result = append(result, "-D__TOPS_LSP_PASS_KIND="+string(pass))
	}
	if input.CompilerToolVersion != "" {
		result = append(result, "-D__TOPS_LSP_COMPILER_TOOL_VERSION="+input.CompilerToolVersion)
	}
	return result
}

func comparisonProvenanceArguments(provenance map[string]any, flag string) []string {
	if provenance == nil {
		return nil
	}
	result := make([]string, 0)
	for _, key := range []string{"include_roots", "roots", "macros"} {
		value, ok := provenance[key]
		if !ok {
			continue
		}
		if key == "macros" {
			macros, ok := value.(map[string]any)
			if !ok {
				continue
			}
			names := make([]string, 0, len(macros))
			for name := range macros {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				macroValue := fmt.Sprint(macros[name])
				if macroValue == "" || macroValue == "1" {
					result = append(result, flag+name)
				} else {
					result = append(result, flag+name+"="+macroValue)
				}
			}
			continue
		}
		values, ok := value.([]any)
		if !ok {
			if single, ok := value.(string); ok && single != "" {
				result = append(result, flag, single)
			}
			continue
		}
		for _, item := range values {
			if root, ok := item.(string); ok && root != "" {
				result = append(result, flag, root)
			}
		}
	}
	return result
}

func clangDiagnosticCategories(output []byte) []string {
	seen := make(map[string]struct{})
	for _, line := range strings.Split(string(output), "\n") {
		for _, category := range []string{"error", "warning", "note", "remark"} {
			if strings.Contains(line, ": "+category+":") {
				seen[category] = struct{}{}
			}
		}
	}
	categories := make([]string, 0, len(seen))
	for category := range seen {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	return categories
}
