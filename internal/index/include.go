package index

import (
	"sort"
	"strings"
	"tops-lsp/internal/parser"
)

func buildWithProvider(text string, result parser.ParseResult, context parser.ParseContext, provider SourceProvider, cache map[string]IndexSnapshot, stack map[string]bool) IndexSnapshot {
	snapshot := Build(text, result, context)
	if provider == nil {
		return snapshot
	}
	for _, include := range includeDirectives(result.Tokens) {
		condition := provenanceAtSnapshot(&snapshot, include.Range.Start)
		if condition.State == "" {
			condition = ConditionalProvenance{SourceURI: snapshot.RootURI, State: include.State}
		}
		if condition.State == parser.ConditionalInactive {
			continue
		}
		roots, _ := provider.IncludeRoots()
		source, status := resolveInclude(provider, context.DocumentURI, include.Spelling, include.Form)
		dependency := HeaderDependency{
			RequesterURI:    context.DocumentURI,
			IncludeSpelling: include.Spelling,
			IncludeForm:     include.Form,
			IncludeRange:    include.Range,
			SearchRoots:     append([]string(nil), roots...),
			Condition:       condition,
			Dependents:      []string{context.DocumentURI},
			SourceVersion: SourceVersion{
				URI:             source.URI,
				DocumentVersion: source.Version,
				ContextVersion:  context.ContextVersion,
				SourceVersion:   source.SourceVersion,
			},
			Status: status,
		}
		if status != HeaderDependencyResolved || source.URI == "" {
			snapshot.Dependencies = append(snapshot.Dependencies, dependency)
			if status != HeaderDependencyResolved {
				snapshot.IndexStatus = IndexStatusDegraded
				snapshot.Diagnostics = append(snapshot.Diagnostics, IndexDiagnostic{SourceURI: context.DocumentURI, SourceVersion: snapshot.SourceVersion, Code: "analysis-degraded", Message: "header dependency is not fully available", Range: include.Range, Reason: string(status), Condition: condition, ConditionalState: condition.State})
			}
			continue
		}
		if condition.State == parser.ConditionalUnknown {
			snapshot.IndexStatus = IndexStatusDegraded
			appendIndexDiagnostic(&snapshot, IndexDiagnostic{SourceURI: context.DocumentURI, SourceVersion: snapshot.SourceVersion, Code: "analysis-degraded", Message: "header include condition is unknown", Range: include.Range, Reason: "unknown-condition", Condition: condition, ConditionalState: condition.State})
		}
		dependency.TargetURI = source.URI
		if stack[source.URI] {
			dependency.Status = HeaderDependencyCycle
			snapshot.Dependencies = append(snapshot.Dependencies, dependency)
			snapshot.IndexStatus = IndexStatusDegraded
			snapshot.Diagnostics = append(snapshot.Diagnostics, IndexDiagnostic{SourceURI: context.DocumentURI, SourceVersion: snapshot.SourceVersion, Code: "analysis-degraded", Message: "header include cycle detected", Range: include.Range, Reason: "include-cycle", Condition: condition, ConditionalState: condition.State})
			continue
		}
		headerSnapshot, ok := cache[source.URI]
		if !ok {
			if headerCache, cacheable := provider.(HeaderSnapshotCache); cacheable {
				headerSnapshot, ok = headerCache.GetHeaderSnapshot(source.URI, sourceVersionForDocument(source), context.ContextVersion)
			}
		}
		if !ok {
			headerContext := context
			headerContext.DocumentURI = source.URI
			headerContext.DocumentVersion = source.Version
			headerContext.LanguageID = "tops-header"
			headerResult := parser.Parse(source.Text, headerContext)
			stack[source.URI] = true
			headerSnapshot = buildWithProvider(source.Text, headerResult, headerContext, provider, cache, stack)
			delete(stack, source.URI)
			cache[source.URI] = headerSnapshot
			if headerCache, cacheable := provider.(HeaderSnapshotCache); cacheable {
				headerCache.PutHeaderSnapshot(headerSnapshot)
			}
		}
		snapshot.Dependencies = append(snapshot.Dependencies, dependency)
		mergeSnapshotWithCondition(&snapshot, headerSnapshot, condition)
		if headerSnapshot.IndexStatus != IndexStatusComplete {
			snapshot.IndexStatus = IndexStatusDegraded
		}
	}
	snapshot.SourceVersion.SourceVersion = dependencyVersionDigest(snapshot.SourceVersion.SourceVersion, snapshot.Dependencies)
	applyFindingStatus(&snapshot)
	return snapshot
}

func applyFindingStatus(snapshot *IndexSnapshot) {
	if snapshot.IndexStatus == IndexStatusStale {
		return
	}
	for _, symbol := range snapshot.Symbols {
		if symbol.Condition.State == parser.ConditionalUnknown {
			snapshot.IndexStatus = IndexStatusDegraded
			appendIndexDiagnostic(snapshot, IndexDiagnostic{SourceURI: symbol.SourceVersion.URI, SourceVersion: symbol.SourceVersion, Code: "analysis-degraded", Message: "condition value is unknown", Reason: "unknown-condition", Range: symbol.NameRange, Condition: symbol.Condition, ConditionalState: symbol.Condition.State})
		}
		if symbol.Status == SymbolStatusAmbiguous {
			snapshot.IndexStatus = IndexStatusDegraded
			appendIndexDiagnostic(snapshot, IndexDiagnostic{SourceURI: symbol.SourceVersion.URI, SourceVersion: symbol.SourceVersion, Code: "analysis-degraded", Message: "symbol reference is ambiguous", Reason: "ambiguous-candidate", Range: symbol.NameRange, Condition: symbol.Condition, ConditionalState: symbol.Condition.State})
		}
	}
	for _, reference := range snapshot.References {
		if reference.Status == ReferenceStatusUnknown || reference.Status == ReferenceStatusAmbiguous {
			snapshot.IndexStatus = IndexStatusDegraded
			appendIndexDiagnostic(snapshot, IndexDiagnostic{SourceURI: reference.URI, SourceVersion: reference.SourceVersion, Code: "analysis-degraded", Message: "reference resolution is not complete", Reason: string(reference.Status), Range: reference.Range, Condition: reference.Condition, ConditionalState: reference.Condition.State})
		}
	}
	for _, dependency := range snapshot.Dependencies {
		if dependency.Status != HeaderDependencyResolved {
			snapshot.IndexStatus = IndexStatusDegraded
		}
	}
	if snapshot.IndexStatus == IndexStatusPartial || snapshot.IndexStatus == IndexStatusDegraded {
		rootDiagnostic := false
		for _, diagnostic := range snapshot.Diagnostics {
			if diagnostic.Code == "analysis-degraded" && (diagnostic.SourceURI == "" || diagnostic.SourceURI == snapshot.RootURI) {
				rootDiagnostic = true
				break
			}
		}
		if rootDiagnostic {
			return
		}
		reason := "degraded-index"
		message := "index is only partially available"
		if snapshot.IndexStatus == IndexStatusPartial {
			reason = "partial-index"
			message = "index is incomplete for the current document"
		}
		appendIndexDiagnostic(snapshot, IndexDiagnostic{
			SourceURI:        snapshot.RootURI,
			SourceVersion:    snapshot.SourceVersion,
			Code:             "analysis-degraded",
			Message:          message,
			Reason:           reason,
			Range:            parser.SourceRange{},
			Condition:        ConditionalProvenance{SourceURI: snapshot.RootURI, State: parser.ConditionalActive},
			ConditionalState: parser.ConditionalActive,
		})
	}
}

func appendIndexDiagnostic(snapshot *IndexSnapshot, diagnostic IndexDiagnostic) {
	for _, existing := range snapshot.Diagnostics {
		if existing.SourceURI == diagnostic.SourceURI && existing.Code == diagnostic.Code && existing.Reason == diagnostic.Reason && existing.Range == diagnostic.Range {
			return
		}
	}
	snapshot.Diagnostics = append(snapshot.Diagnostics, diagnostic)
}

func dependencyVersionDigest(rootDigest string, dependencies []HeaderDependency) string {
	parts := []string{rootDigest}
	for _, dependency := range dependencies {
		parts = append(parts, dependency.TargetURI+":"+dependency.SourceVersion.SourceVersion+":"+string(dependency.Status))
	}
	sort.Strings(parts[1:])
	return SourceDigest(strings.Join(parts, "\x00"))
}

type includeDirective struct {
	Spelling string
	Form     IncludeForm
	Range    parser.SourceRange
	State    parser.ConditionalState
}

func includeDirectives(tokens []parser.Token) []includeDirective {
	directives := make([]includeDirective, 0)
	for index := 0; index < len(tokens); index++ {
		if tokens[index].Spelling != "#" || !tokens[index].LineStart || tokens[index].ConditionalState == parser.ConditionalInactive {
			continue
		}
		lineEnd := index + 1
		for lineEnd < len(tokens) && tokens[lineEnd].Kind != parser.TokenNewline {
			lineEnd++
		}
		nameIndex := index + 1
		for nameIndex < lineEnd && (tokens[nameIndex].Kind == parser.TokenComment || tokens[nameIndex].Kind == parser.TokenNewline) {
			nameIndex++
		}
		if nameIndex >= lineEnd || tokens[nameIndex].Spelling != "include" {
			index = lineEnd
			continue
		}
		valueIndex := nameIndex + 1
		for valueIndex < lineEnd && tokens[valueIndex].Kind == parser.TokenComment {
			valueIndex++
		}
		if valueIndex >= lineEnd {
			index = lineEnd
			continue
		}
		endIndex := valueIndex
		spelling := strings.Trim(tokens[valueIndex].Spelling, "<>\"")
		form := IncludeFormQuoted
		if tokens[valueIndex].Spelling == "<" {
			form = IncludeFormAngle
			var parts strings.Builder
			endIndex = valueIndex + 1
			for endIndex < lineEnd && tokens[endIndex].Spelling != ">" {
				if tokens[endIndex].Kind != parser.TokenComment {
					parts.WriteString(tokens[endIndex].Spelling)
				}
				endIndex++
			}
			spelling = parts.String()
		}
		if spelling == "" {
			index = lineEnd
			continue
		}
		if endIndex >= lineEnd {
			endIndex = lineEnd - 1
		} else if tokens[endIndex].Spelling == ">" {
			// Keep the closing delimiter in the dependency range.
		}
		directives = append(directives, includeDirective{
			Spelling: spelling,
			Form:     form,
			Range: parser.SourceRange{
				Start: tokens[index].Range.Start,
				End:   tokens[endIndex].Range.End,
			},
			State: tokens[index].ConditionalState,
		})
		index = lineEnd
	}
	return directives
}

func mergeSnapshotWithCondition(target *IndexSnapshot, source IndexSnapshot, parentCondition ConditionalProvenance) {
	for id, scope := range source.Scopes {
		target.Scopes[id] = scope
	}
	for id, symbol := range source.Symbols {
		symbol.Condition = combineConditionalProvenance(parentCondition, symbol.Condition)
		for index := range symbol.Declarations {
			symbol.Declarations[index].Condition = combineConditionalProvenance(parentCondition, symbol.Declarations[index].Condition)
		}
		for index := range symbol.Definitions {
			symbol.Definitions[index].Condition = combineConditionalProvenance(parentCondition, symbol.Definitions[index].Condition)
		}
		for index := range symbol.AttributeUsages {
			symbol.AttributeUsages[index].Condition = combineConditionalProvenance(parentCondition, symbol.AttributeUsages[index].Condition)
		}
		for index := range symbol.AttributeSources {
			symbol.AttributeSources[index].Condition = combineConditionalProvenance(parentCondition, symbol.AttributeSources[index].Condition)
		}
		if symbol.Condition.State == parser.ConditionalUnknown && symbol.Status == SymbolStatusActive {
			symbol.Status = SymbolStatusConditional
		}
		target.Symbols[id] = symbol
	}
	for id, reference := range source.References {
		reference.Condition = combineConditionalProvenance(parentCondition, reference.Condition)
		reference.Status = referenceStatus(reference.Condition.State, reference.Candidates)
		target.References[id] = reference
	}
	for uri, text := range source.Sources {
		target.Sources[uri] = text
	}
	if target.ConditionalRegionsByURI == nil {
		target.ConditionalRegionsByURI = make(map[string][]parser.ConditionalRegion)
	}
	for uri, regions := range source.ConditionalRegionsByURI {
		target.ConditionalRegionsByURI[uri] = append([]parser.ConditionalRegion(nil), regions...)
	}
	for _, dependency := range source.Dependencies {
		dependency.Condition = combineConditionalProvenance(parentCondition, dependency.Condition)
		target.Dependencies = append(target.Dependencies, dependency)
	}
	for _, diagnostic := range source.Diagnostics {
		condition := diagnostic.Condition
		if condition.State == "" {
			condition = ConditionalProvenance{SourceURI: diagnostic.SourceURI, State: diagnostic.ConditionalState}
		}
		diagnostic.Condition = combineConditionalProvenance(parentCondition, condition)
		diagnostic.ConditionalState = diagnostic.Condition.State
		target.Diagnostics = append(target.Diagnostics, diagnostic)
	}
}

func combineConditionalProvenance(parent, child ConditionalProvenance) ConditionalProvenance {
	if parent.State == "" {
		return child
	}
	if child.State == "" {
		return parent
	}
	result := child
	if child.State == parser.ConditionalInactive || parent.State == parser.ConditionalInactive {
		if child.State == parser.ConditionalInactive {
			result = child
		} else {
			result = parent
		}
	} else if parent.State == parser.ConditionalUnknown {
		result = parent
	} else if child.State == parser.ConditionalUnknown {
		result = child
	}
	parents := make([]ConditionalProvenance, 0, len(parent.ParentConditions)+len(child.ParentConditions)+2)
	parents = append(parents, parent.ParentConditions...)
	parents = append(parents, parent)
	parents = append(parents, child.ParentConditions...)
	parents = append(parents, child)
	result.ParentConditions = parents
	return result
}
