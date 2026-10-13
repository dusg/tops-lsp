package index

import (
	"fmt"
	"sort"
	"strings"

	"tops-lsp/internal/parser"
)

type builder struct {
	snapshot           IndexSnapshot
	text               string
	context            parser.ParseContext
	declarationRanges  []parser.SourceRange
	symbolsByName      map[string][]string
	scopeForSymbol     map[string]string
	conditionOverride  ConditionalProvenance
	conditionalRegions []parser.ConditionalRegion
}

var builtinSpellings = map[string]struct{}{
	"threadIdx":    {},
	"blockIdx":     {},
	"blockDim":     {},
	"gridDim":      {},
	"threadDim":    {},
	"subThreadIdx": {},
	"warpSize":     {},
}

func Build(text string, result parser.ParseResult, context parser.ParseContext) IndexSnapshot {
	builder := &builder{
		text:               text,
		context:            context,
		declarationRanges:  make([]parser.SourceRange, 0),
		symbolsByName:      make(map[string][]string),
		scopeForSymbol:     make(map[string]string),
		conditionalRegions: append([]parser.ConditionalRegion(nil), result.ConditionalRegions...),
		snapshot: IndexSnapshot{
			RootURI:            context.DocumentURI,
			SourceVersion:      SourceVersion{URI: context.DocumentURI, DocumentVersion: context.DocumentVersion, ContextVersion: context.ContextVersion, SourceVersion: SourceDigest(text)},
			ContextVersion:     context.ContextVersion,
			ParseStatus:        result.Status,
			IndexStatus:        indexStatusForParse(result, context),
			ASTRoot:            result.Root,
			Tokens:             append([]parser.Token(nil), result.Tokens...),
			ConditionalRegions: append([]parser.ConditionalRegion(nil), result.ConditionalRegions...),
			ConditionalRegionsByURI: map[string][]parser.ConditionalRegion{
				context.DocumentURI: append([]parser.ConditionalRegion(nil), result.ConditionalRegions...),
			},
			Sources:          map[string]string{context.DocumentURI: text},
			TargetProfile:    context.TargetProfile,
			PassKind:         context.PassKind,
			PredefinedMacros: cloneMacroContext(context.PredefinedMacros),
			Scopes:           make(map[string]ScopeRecord),
			Symbols:          make(map[string]SymbolRecord),
			References:       make(map[string]ReferenceRecord),
			RootSymbols:      make([]string, 0),
			Dependencies:     make([]HeaderDependency, 0),
			Diagnostics:      make([]IndexDiagnostic, 0),
		},
	}
	rootID := builder.addScope(ScopeRecord{
		ScopeID:       builder.scopeID("", parser.SourceRange{Start: 0, End: len(text)}),
		SourceURI:     context.DocumentURI,
		Kind:          ScopeTranslationUnit,
		QualifiedName: "",
		Range:         parser.SourceRange{Start: 0, End: len(text)},
	})
	builder.walk(result.Root, rootID)
	for regionIndex, region := range result.ConditionalRegions {
		for branchIndex, branch := range region.Branches {
			if branch.State == parser.ConditionalActive {
				continue
			}
			builder.conditionOverride = conditionalProvenance(context.DocumentURI, regionIndex, branchIndex, region, branch)
			for _, node := range branch.Nodes {
				parentScope := builder.enclosingScopeForNode(node, rootID)
				if parentScope == "" {
					parentScope = rootID
				}
				builder.walk(node, parentScope)
			}
		}
	}
	builder.conditionOverride = ConditionalProvenance{}
	builder.collectReferences(result.Tokens)
	collectAttributeReferencesForSnapshot(&builder.snapshot)
	builder.finishChildren()
	builder.snapshot.RootSymbols = append([]string(nil), builder.snapshot.Scopes[rootID].Symbols...)
	applyFindingStatus(&builder.snapshot)
	return builder.snapshot
}

func BuildWithProvider(text string, result parser.ParseResult, context parser.ParseContext, provider SourceProvider) IndexSnapshot {
	cache := make(map[string]IndexSnapshot)
	stack := map[string]bool{context.DocumentURI: true}
	snapshot := buildWithProvider(text, result, context, provider, cache, stack)
	resolveSourceProvenance(&snapshot, provider)
	rebuildReferences(&snapshot)
	return snapshot
}

func resolveSourceProvenance(snapshot *IndexSnapshot, provider SourceProvider) {
	if provider == nil {
		return
	}
	attributeResolver, hasAttributeResolver := provider.(AttributeSourceResolver)
	verifiedAttributeResolver, hasVerifiedAttributeResolver := provider.(VerifiedAttributeSourceResolver)
	builtinResolver, hasBuiltinResolver := provider.(BuiltinSourceResolver)
	verifiedBuiltinResolver, hasVerifiedBuiltinResolver := provider.(VerifiedBuiltinSourceResolver)
	for id, symbol := range snapshot.Symbols {
		switch symbol.Kind {
		case SymbolKindAttribute:
			symbol.AttributeSources = nil
			symbol.AttributeProvenance = BuiltinProvenance{TargetProfile: snapshot.TargetProfile, PassKind: snapshot.PassKind, MacroContext: cloneMacroContext(snapshot.PredefinedMacros), Availability: "unresolved"}
			for _, usage := range symbol.AttributeUsages {
				var source SymbolLocation
				var found bool
				resolutionContext := sourceResolutionContext(*snapshot, usage.URI)
				if hasVerifiedAttributeResolver {
					source, found = verifiedAttributeResolver.ResolveAttributeSource(resolutionContext, symbol.Name)
				} else if hasAttributeResolver {
					source, found = attributeResolver.ResolveAttribute(usage.URI, symbol.Name)
					if found && !HasCompleteSourceProvenance(resolutionContext, source) {
						found = false
					}
				}
				if found {
					source, found = registerVerifiedSource(snapshot, provider, source)
				}
				if !found {
					continue
				}
				source.Role = SymbolLocationAttribute
				symbol.AttributeSources = appendUniqueLocation(symbol.AttributeSources, source)
				symbol.AttributeProvenance = BuiltinProvenance{TargetProfile: snapshot.TargetProfile, PassKind: snapshot.PassKind, MacroContext: cloneMacroContext(snapshot.PredefinedMacros), SourceURI: source.URI, Availability: "available"}
				symbol.Origin = source.URI
				symbol.OriginKind = "header"
				symbol.Status = SymbolStatusActive
			}
		case SymbolKindBuiltin:
			symbol.Builtin.Availability = "unresolved"
			symbol.AttributeSources = nil
			if hasBuiltinResolver || hasVerifiedBuiltinResolver {
				resolutionContext := sourceResolutionContext(*snapshot, symbol.SourceVersion.URI)
				var source SymbolLocation
				var ok bool
				if hasVerifiedBuiltinResolver {
					source, ok = verifiedBuiltinResolver.ResolveBuiltinSource(resolutionContext, symbol.Name)
				} else {
					source, ok = builtinResolver.ResolveBuiltin(resolutionContext.RequesterURI, symbol.Name)
					if ok && !HasCompleteSourceProvenance(resolutionContext, source) {
						ok = false
					}
				}
				if ok {
					source, ok = registerVerifiedSource(snapshot, provider, source)
					if !ok {
						continue
					}
					source.Role = SymbolLocationBuiltin
					symbol.AttributeSources = appendUniqueLocation(symbol.AttributeSources, source)
					symbol.Builtin.SourceURI = source.URI
					symbol.Builtin.Availability = "available"
					symbol.Origin = source.URI
					symbol.OriginKind = "builtin"
				}
			}
		}
		snapshot.Symbols[id] = symbol
	}
}

func registerVerifiedSource(snapshot *IndexSnapshot, provider SourceProvider, source SymbolLocation) (SymbolLocation, bool) {
	if source.URI == "" {
		return SymbolLocation{}, false
	}
	if !isHeaderURI(source.URI) {
		return SymbolLocation{}, false
	}
	text, ok := snapshot.Sources[source.URI]
	if !ok {
		document, found := provider.OpenDocument(source.URI)
		if !found {
			return SymbolLocation{}, false
		}
		text = document.Text
		snapshot.Sources[source.URI] = text
		if source.SourceVersion.URI == "" {
			source.SourceVersion.URI = document.URI
		}
		if source.SourceVersion.DocumentVersion == 0 {
			source.SourceVersion.DocumentVersion = document.Version
		}
		if source.SourceVersion.SourceVersion == "" {
			source.SourceVersion.SourceVersion = document.SourceVersion
		}
	} else if source.SourceVersion.DocumentVersion == 0 || source.SourceVersion.SourceVersion == "" {
		if document, found := provider.OpenDocument(source.URI); found {
			if source.SourceVersion.URI == "" {
				source.SourceVersion.URI = document.URI
			}
			if source.SourceVersion.DocumentVersion == 0 {
				source.SourceVersion.DocumentVersion = document.Version
			}
			if source.SourceVersion.SourceVersion == "" {
				source.SourceVersion.SourceVersion = document.SourceVersion
			}
		}
	}
	if source.SourceVersion.URI == "" {
		source.SourceVersion.URI = source.URI
	}
	if source.SourceVersion.ContextVersion == 0 {
		source.SourceVersion.ContextVersion = snapshot.ContextVersion
	}
	if source.SourceVersion.DocumentVersion == 0 {
		return SymbolLocation{}, false
	}
	if source.SourceVersion.SourceVersion == "" || source.SourceVersion.ContextVersion != snapshot.ContextVersion {
		return SymbolLocation{}, false
	}
	if !validSourceRangeForIndex(text, source.Range) || !validSourceRangeForIndex(text, source.SelectionRange) {
		return SymbolLocation{}, false
	}
	return source, true
}

func sourceResolutionContext(snapshot IndexSnapshot, requesterURI string) SourceResolutionContext {
	allowedSourceURIs := make([]string, 0, len(snapshot.Sources))
	for uri := range snapshot.Sources {
		if isHeaderURI(uri) {
			allowedSourceURIs = append(allowedSourceURIs, uri)
		}
	}
	sort.Strings(allowedSourceURIs)
	return SourceResolutionContext{
		RequesterURI:      requesterURI,
		ContextVersion:    snapshot.ContextVersion,
		TargetProfile:     snapshot.TargetProfile,
		PassKind:          snapshot.PassKind,
		MacroContext:      cloneMacroContext(snapshot.PredefinedMacros),
		AllowedSourceURIs: allowedSourceURIs,
	}
}

func validSourceRangeForIndex(text string, sourceRange parser.SourceRange) bool {
	return sourceRange.Start >= 0 && sourceRange.Start <= sourceRange.End && sourceRange.End <= len(text)
}

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

func sourceVersionForDocument(source SourceDocument) string {
	if source.SourceVersion != "" {
		return source.SourceVersion
	}
	return SourceDigest(source.Text)
}

func rebuildReferences(snapshot *IndexSnapshot) {
	preserved := make(map[string]ReferenceRecord)
	for id, reference := range snapshot.References {
		if reference.URI != snapshot.RootURI {
			preserved[id] = reference
		}
	}
	snapshot.References = preserved
	for id, symbol := range snapshot.Symbols {
		symbol.References = nil
		snapshot.Symbols[id] = symbol
	}
	for _, reference := range preserved {
		reference = normalizeBuiltinReference(snapshot, reference)
		snapshot.References[reference.ReferenceID] = reference
		attachReference(snapshot, reference)
	}
	declarationRanges := make([]parser.SourceRange, 0)
	for _, symbol := range snapshot.Symbols {
		if symbol.SourceVersion.URI == snapshot.RootURI {
			declarationRanges = append(declarationRanges, symbol.NameRange)
		}
	}
	collectSnapshotReferences(snapshot, snapshot.RootURI, snapshot.Tokens, snapshot.ConditionalRegions, declarationRanges, snapshot.SourceVersion)
	collectAttributeReferencesForSnapshot(snapshot)
}

func collectSnapshotReferences(snapshot *IndexSnapshot, uri string, tokens []parser.Token, regions []parser.ConditionalRegion, declarationRanges []parser.SourceRange, sourceVersion SourceVersion) {
	for tokenIndex, token := range tokens {
		if token.Kind != parser.TokenIdentifier || containsRange(declarationRanges, token.Range) {
			continue
		}
		scopeID := scopeAtSnapshot(snapshot, uri, token.Range.Start)
		qualifiedName := qualifiedReferenceName(tokens, tokenIndex)
		candidates := []string(nil)
		if token.ConditionalState != parser.ConditionalInactive {
			candidates = candidateIDsForReference(tokens, tokenIndex, snapshot, token.Spelling, qualifiedName, scopeID, token.ConditionalState)
		}
		if hasAttributeCandidate(snapshot, candidates) {
			continue
		}
		condition := provenanceAtRegions(regions, token.Range.Start, uri)
		if condition.State == "" {
			condition = ConditionalProvenance{SourceURI: uri, State: token.ConditionalState}
		}
		reference := newReferenceRecord(uri, token, qualifiedName, scopeID, candidates, condition, referenceKind(tokens, tokenIndex, snapshot, candidates), sourceVersion)
		reference.Builtin = builtinProvenanceForSnapshot(snapshot, candidates, token)
		reference = normalizeBuiltinReference(snapshot, reference)
		snapshot.References[reference.ReferenceID] = reference
		attachReference(snapshot, reference)
	}
}

func symbolIDsByName(snapshot *IndexSnapshot, name string) []string {
	ids := make([]string, 0)
	for id, symbol := range snapshot.Symbols {
		if symbol.Name == name {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func scopeAtSnapshot(snapshot *IndexSnapshot, uri string, offset int) string {
	best := ""
	bestSize := int(^uint(0) >> 1)
	for id, scope := range snapshot.Scopes {
		if scope.SourceURI != "" && scope.SourceURI != uri {
			continue
		}
		if scope.Range.Start <= offset && offset < scope.Range.End {
			size := scope.Range.End - scope.Range.Start
			if size < bestSize {
				best = id
				bestSize = size
			}
		}
	}
	return best
}

func newReferenceRecord(uri string, token parser.Token, qualifiedName, scopeID string, candidates []string, condition ConditionalProvenance, kind string, sourceVersion SourceVersion) ReferenceRecord {
	return ReferenceRecord{
		ReferenceID:   fmt.Sprintf("ref:%s:%d:%d", uri, token.Range.Start, token.Range.End),
		URI:           uri,
		Range:         token.Range,
		Name:          token.Spelling,
		QualifiedName: qualifiedName,
		ScopeID:       scopeID,
		Candidates:    append([]string(nil), candidates...),
		Condition:     condition,
		Kind:          kind,
		Status:        referenceStatus(token.ConditionalState, candidates),
		SourceVersion: sourceVersion,
	}
}

func cloneMacroContext(macros map[string]parser.MacroValue) map[string]parser.MacroValue {
	if macros == nil {
		return nil
	}
	if len(macros) == 0 {
		return map[string]parser.MacroValue{}
	}
	result := make(map[string]parser.MacroValue, len(macros))
	for name, value := range macros {
		result[name] = value
	}
	return result
}

func builtinProvenanceForContext(context parser.ParseContext, sourceURI, availability string) BuiltinProvenance {
	return BuiltinProvenance{
		TargetProfile: context.TargetProfile,
		PassKind:      context.PassKind,
		MacroContext:  cloneMacroContext(context.PredefinedMacros),
		SourceURI:     sourceURI,
		Availability:  availability,
	}
}

func builtinProvenanceForSnapshot(snapshot *IndexSnapshot, candidates []string, token parser.Token) BuiltinProvenance {
	if !isBuiltinSpelling(token.Spelling) {
		return BuiltinProvenance{}
	}
	for _, candidate := range candidates {
		if symbol, ok := snapshot.Symbols[candidate]; ok && symbol.Kind == SymbolKindBuiltin {
			provenance := symbol.Builtin
			if provenance.Availability == "" {
				provenance.Availability = "available"
			}
			return provenance
		}
	}
	return BuiltinProvenance{
		TargetProfile: snapshot.TargetProfile,
		PassKind:      snapshot.PassKind,
		MacroContext:  cloneMacroContext(snapshot.PredefinedMacros),
		Availability:  "unresolved",
	}
}

func normalizeBuiltinReference(snapshot *IndexSnapshot, reference ReferenceRecord) ReferenceRecord {
	if reference.Kind != "builtin-use" {
		return reference
	}
	if reference.Builtin.Availability != "available" {
		reference.Candidates = nil
		reference.Status = ReferenceStatusUnresolved
	}
	return reference
}

func referenceStatus(state parser.ConditionalState, candidates []string) ReferenceStatus {
	if state == parser.ConditionalInactive {
		return ReferenceStatusInactive
	}
	if len(candidates) == 0 {
		if state == parser.ConditionalUnknown {
			return ReferenceStatusUnknown
		}
		return ReferenceStatusUnresolved
	}
	if state == parser.ConditionalUnknown {
		return ReferenceStatusUnknown
	}
	if len(candidates) > 1 {
		return ReferenceStatusAmbiguous
	}
	return ReferenceStatusResolved
}

func referenceKind(tokens []parser.Token, index int, snapshot *IndexSnapshot, candidates []string) string {
	if index+1 < len(tokens) && tokens[index+1].Spelling == "(" {
		return "call"
	}
	if index > 0 && (tokens[index-1].Spelling == "." || tokens[index-1].Spelling == "->") {
		return "member-use"
	}
	for _, candidate := range candidates {
		symbol, ok := snapshot.Symbols[candidate]
		if !ok {
			continue
		}
		switch symbol.Kind {
		case SymbolKindType:
			return "type-use"
		case SymbolKindAttribute:
			return "attribute-use"
		case SymbolKindBuiltin:
			return "builtin-use"
		case SymbolKindMacro:
			return "macro-use"
		}
	}
	if isBuiltinSpelling(tokens[index].Spelling) {
		return "builtin-use"
	}
	return "other"
}

func attachReference(snapshot *IndexSnapshot, reference ReferenceRecord) {
	for _, candidate := range reference.Candidates {
		symbol, ok := snapshot.Symbols[candidate]
		if !ok {
			continue
		}
		symbol.References = appendUniqueString(symbol.References, reference.ReferenceID)
		if reference.Status == ReferenceStatusAmbiguous {
			symbol.Status = SymbolStatusAmbiguous
		}
		snapshot.Symbols[candidate] = symbol
	}
}

func provenanceAtSnapshot(snapshot *IndexSnapshot, offset int) ConditionalProvenance {
	return provenanceAtRegions(snapshot.ConditionalRegions, offset, snapshot.RootURI)
}

func provenanceAtRegions(regions []parser.ConditionalRegion, offset int, sourceURI string) ConditionalProvenance {
	matches := make([]int, 0)
	for regionIndex, region := range regions {
		for _, branch := range region.Branches {
			if branch.BodyRange.Start <= offset && offset < branch.BodyRange.End {
				matches = append(matches, regionIndex)
				break
			}
		}
	}
	sort.SliceStable(matches, func(left, right int) bool {
		return conditionalRegionDepth(regions, matches[left]) < conditionalRegionDepth(regions, matches[right])
	})
	condition := ConditionalProvenance{}
	for _, regionIndex := range matches {
		region := regions[regionIndex]
		for branchIndex, branch := range region.Branches {
			if branch.BodyRange.Start <= offset && offset < branch.BodyRange.End {
				condition = combineConditionalProvenance(condition, conditionalProvenance(sourceURI, regionIndex, branchIndex, region, branch))
				break
			}
		}
	}
	return condition
}

func conditionalRegionDepth(regions []parser.ConditionalRegion, regionIndex int) int {
	depth := 0
	for parentID := regions[regionIndex].ParentID; parentID >= 0 && parentID < len(regions); parentID = regions[parentID].ParentID {
		depth++
	}
	return depth
}

func indexStatusForParse(result parser.ParseResult, context parser.ParseContext) IndexStatus {
	if result.Status == parser.ParseStatusInvalid {
		if result.Root == nil || len(result.Root.Children) == 0 {
			return IndexStatusDegraded
		}
		return IndexStatusPartial
	}
	if result.Status == parser.ParseStatusRecovered || result.Status == parser.ParseStatusPartial {
		return IndexStatusPartial
	}
	if context.CompilerContextStatus != "resolved" {
		return IndexStatusPartial
	}
	return IndexStatusComplete
}

func (builder *builder) addScope(scope ScopeRecord) string {
	if scope.ScopeID == "" {
		scope.ScopeID = builder.scopeID(scope.QualifiedName, scope.Range)
	}
	builder.snapshot.Scopes[scope.ScopeID] = scope
	if scope.ParentID != "" {
		parent := builder.snapshot.Scopes[scope.ParentID]
		parent.Children = append(parent.Children, scope.ScopeID)
		builder.snapshot.Scopes[scope.ParentID] = parent
	}
	return scope.ScopeID
}

func (builder *builder) walk(node *parser.SyntaxNode, parentScope string) {
	if node == nil {
		return
	}
	currentScope := parentScope
	symbolID := ""
	if name, kind, typeKind, ok := symbolInfo(node, parentScope, builder.snapshot.Scopes, builder.context); ok {
		symbolID = builder.addSymbol(node, parentScope, name, kind, typeKind)
		if scopeKind, isScope := scopeKindForNode(node); isScope {
			qualified := builder.qualifiedName(parentScope, name)
			currentScope = builder.addScope(ScopeRecord{
				ScopeID:       builder.scopeID(qualified, node.Range),
				SourceURI:     builder.context.DocumentURI,
				Kind:          scopeKind,
				Name:          name,
				QualifiedName: qualified,
				ParentID:      parentScope,
				Range:         node.Range,
			})
			builder.scopeForSymbol[symbolID] = currentScope
		}
	} else if scopeKind, isScope := scopeKindForNode(node); isScope {
		currentScope = builder.addScope(ScopeRecord{
			ScopeID:   builder.scopeID(builder.snapshot.Scopes[parentScope].QualifiedName, node.Range),
			SourceURI: builder.context.DocumentURI,
			Kind:      scopeKind,
			ParentID:  parentScope,
			Range:     node.Range,
		})
	}
	if node.Kind == parser.NodeFunction && currentScope != parentScope {
		builder.addFunctionParameters(node, currentScope)
	}
	for _, child := range node.Children {
		builder.walk(child, currentScope)
	}
	if symbolID != "" {
		if scopeID := builder.scopeForSymbol[symbolID]; scopeID != "" {
			symbol := builder.snapshot.Symbols[symbolID]
			scope := builder.snapshot.Scopes[scopeID]
			symbol.Children = append([]string(nil), scope.Symbols...)
			builder.snapshot.Symbols[symbolID] = symbol
		}
	}
}

func (builder *builder) addFunctionParameters(node *parser.SyntaxNode, functionScope string) {
	if node == nil || functionScope == "" {
		return
	}
	openParen := -1
	for index, token := range node.DeclaratorTokens {
		if token.Spelling == "(" {
			openParen = index
			break
		}
	}
	if openParen < 0 {
		return
	}
	closeParen := matchingParameterToken(node.DeclaratorTokens, openParen)
	if closeParen <= openParen {
		return
	}
	start := openParen + 1
	depth := 0
	for index := start; index <= closeParen; index++ {
		atEnd := index == closeParen
		if !atEnd {
			switch node.DeclaratorTokens[index].Spelling {
			case "(", "[", "<":
				depth++
			case ")", "]", ">":
				if depth > 0 {
					depth--
				}
			case ",":
				if depth == 0 {
					builder.addFunctionParameter(node, functionScope, node.DeclaratorTokens[start:index])
					start = index + 1
				}
			}
		}
		if atEnd {
			builder.addFunctionParameter(node, functionScope, node.DeclaratorTokens[start:index])
		}
	}
}

func matchingParameterToken(tokens []parser.Token, openParen int) int {
	depth := 0
	for index := openParen; index < len(tokens); index++ {
		switch tokens[index].Spelling {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func (builder *builder) addFunctionParameter(function *parser.SyntaxNode, functionScope string, tokens []parser.Token) {
	if len(tokens) == 0 {
		return
	}
	end := len(tokens)
	depth := 0
	for index, token := range tokens {
		switch token.Spelling {
		case "(", "[", "<":
			depth++
		case ")", "]", ">":
			if depth > 0 {
				depth--
			}
		case "=":
			if depth == 0 {
				end = index
			}
		}
		if index >= end {
			break
		}
	}
	if end == 0 {
		return
	}
	nameIndex := -1
	for index := end - 1; index >= 0; index-- {
		if tokens[index].Kind == parser.TokenIdentifier {
			nameIndex = index
			break
		}
	}
	if nameIndex <= 0 {
		return
	}
	parameterTokens := append([]parser.Token(nil), tokens[:end]...)
	parameter := &parser.SyntaxNode{
		Kind:             parser.NodeDeclaration,
		Range:            parser.SourceRange{Start: parameterTokens[0].Range.Start, End: parameterTokens[len(parameterTokens)-1].Range.End},
		NameRange:        parameterTokens[nameIndex].Range,
		DeclarationRole:  parser.DeclarationRoleDeclaration,
		Name:             parameterTokens[nameIndex].Spelling,
		TypeTokens:       append([]parser.Token(nil), parameterTokens[:nameIndex]...),
		DeclaratorTokens: append([]parser.Token(nil), parameterTokens[nameIndex:]...),
		ConditionalState: function.ConditionalState,
	}
	builder.addSymbol(parameter, functionScope, parameter.Name, SymbolKindVariable, "")
}

func symbolInfo(node *parser.SyntaxNode, parentScope string, scopes map[string]ScopeRecord, context parser.ParseContext) (string, SymbolKind, string, bool) {
	if node.Kind == parser.NodePreprocessorDirective {
		if name := macroName(node); name != "" {
			return name, SymbolKindMacro, "", true
		}
		return "", "", "", false
	}
	if node.Name == "" {
		if node.Kind == parser.NodeAttribute {
			if name := attributeSymbolName(node); name != "" {
				return name, SymbolKindAttribute, "", true
			}
		}
		return "", "", "", false
	}
	switch node.Kind {
	case parser.NodeNamespace:
		return node.Name, SymbolKindNamespace, "", true
	case parser.NodeFunction:
		return node.Name, SymbolKindFunction, "", true
	case parser.NodeClass:
		return node.Name, SymbolKindType, "class", true
	case parser.NodeStruct:
		return node.Name, SymbolKindType, "struct", true
	case parser.NodeEnum:
		return node.Name, SymbolKindType, "enum", true
	case parser.NodeUnion:
		return node.Name, SymbolKindType, "union", true
	case parser.NodeUsing:
		return node.Name, SymbolKindType, "alias", true
	case parser.NodeAttribute, parser.NodeTopsQualifier:
		return attributeSymbolName(node), SymbolKindAttribute, "", true
	case parser.NodeDeclaration:
		if isBuiltinSpelling(node.Name) && context.LanguageID == "tops-header" {
			return node.Name, SymbolKindBuiltin, "", true
		}
		if scope := scopes[parentScope]; scope.Kind == ScopeClass || scope.Kind == ScopeStruct || scope.Kind == ScopeUnion {
			return node.Name, SymbolKindField, "", true
		}
		return node.Name, SymbolKindVariable, "", true
	default:
		return "", "", "", false
	}
}

func isBuiltinSpelling(name string) bool {
	_, ok := builtinSpellings[name]
	return ok
}

func macroName(node *parser.SyntaxNode) string {
	if node == nil {
		return ""
	}
	for index := 0; index+2 < len(node.Tokens); index++ {
		if node.Tokens[index].Spelling != "#" || node.Tokens[index+1].Spelling != "define" {
			continue
		}
		for nameIndex := index + 2; nameIndex < len(node.Tokens); nameIndex++ {
			token := node.Tokens[nameIndex]
			if token.Kind == parser.TokenComment || token.Kind == parser.TokenNewline {
				continue
			}
			if token.Kind == parser.TokenIdentifier || token.Kind == parser.TokenKeyword {
				return token.Spelling
			}
			return ""
		}
	}
	return ""
}

func macroNameRange(node *parser.SyntaxNode, name string) parser.SourceRange {
	if node == nil || name == "" {
		return parser.SourceRange{}
	}
	for index := 0; index+2 < len(node.Tokens); index++ {
		if node.Tokens[index].Spelling != "#" || node.Tokens[index+1].Spelling != "define" {
			continue
		}
		for nameIndex := index + 2; nameIndex < len(node.Tokens); nameIndex++ {
			token := node.Tokens[nameIndex]
			if token.Kind == parser.TokenComment || token.Kind == parser.TokenNewline {
				continue
			}
			if token.Spelling == name {
				return token.Range
			}
			break
		}
	}
	return node.NameRange
}

func attributeSymbolName(node *parser.SyntaxNode) string {
	if node == nil {
		return ""
	}
	if node.Kind == parser.NodeTopsQualifier {
		return node.Name
	}
	for _, token := range node.Tokens {
		if token.Kind == parser.TokenIdentifier || token.Kind == parser.TokenKeyword {
			if token.Spelling != "__attribute__" {
				return token.Spelling
			}
		}
	}
	return node.Name
}

func scopeKindForNode(node *parser.SyntaxNode) (ScopeKind, bool) {
	if node == nil {
		return "", false
	}
	switch node.Kind {
	case parser.NodeNamespace:
		return ScopeNamespace, true
	case parser.NodeClass:
		return ScopeClass, true
	case parser.NodeStruct:
		return ScopeStruct, true
	case parser.NodeUnion:
		return ScopeUnion, true
	case parser.NodeFunction:
		return ScopeFunction, true
	case parser.NodeStatement:
		if node.BodyRange.Start < node.BodyRange.End {
			return ScopeBlock, true
		}
	default:
		return "", false
	}
	return "", false
}

func (builder *builder) addSymbol(node *parser.SyntaxNode, parentScope, name string, kind SymbolKind, typeKind string) string {
	qualified := builder.qualifiedName(parentScope, name)
	role := string(node.DeclarationRole)
	if role == "" {
		role = "unknown"
	}
	identity := SymbolIdentity{
		OriginURI:       builder.context.DocumentURI,
		Kind:            kind,
		QualifiedName:   qualified,
		DeclarationRole: role,
		Anchor:          builder.declarationAnchor(node),
	}
	if kind == SymbolKindAttribute {
		identity.Anchor = "attribute:" + name
	}
	symbolID := StableSymbolID(identity)
	nameRange := node.NameRange
	if kind == SymbolKindMacro {
		nameRange = macroNameRange(node, name)
	}
	locationRole := SymbolLocationDeclaration
	if node.DeclarationRole == parser.DeclarationRoleDefinition {
		locationRole = SymbolLocationDefinition
	}
	location := SymbolLocation{
		URI:            builder.context.DocumentURI,
		Range:          node.Range,
		SelectionRange: nameRange,
		Role:           locationRole,
		SourceVersion:  builder.snapshot.SourceVersion,
	}
	symbol := builder.snapshot.Symbols[symbolID]
	if symbol.SymbolID == "" {
		symbol = SymbolRecord{
			SymbolID:            symbolID,
			Name:                name,
			QualifiedName:       qualified,
			Kind:                kind,
			TypeKind:            typeKind,
			TypeName:            declaredTypeName(node),
			ScopeID:             parentScope,
			ScopeName:           builder.snapshot.Scopes[parentScope].QualifiedName,
			NameRange:           nameRange,
			Condition:           builder.conditionForNode(node),
			Origin:              builder.context.DocumentURI,
			OriginKind:          builder.originKind(kind),
			Detail:              builder.nodeDetail(node),
			Signature:           builder.declarationAnchor(node),
			ParameterSummary:    parameterSummary(node),
			ReturnTypeSummary:   returnTypeSummary(node),
			ExecutionQualifiers: executionQualifiers(node),
			Status:              builder.symbolStatus(node),
			SourceVersion:       builder.snapshot.SourceVersion,
		}
		if kind == SymbolKindAttribute {
			symbol.Origin = ""
			symbol.OriginKind = "unresolved"
			symbol.Status = SymbolStatusUnresolved
		}
	}
	if kind == SymbolKindBuiltin {
		symbol.Builtin = builtinProvenanceForContext(builder.context, "", "unresolved")
	}
	if kind == SymbolKindAttribute {
		symbol.AttributeUsages = appendUniqueLocation(symbol.AttributeUsages, SymbolLocation{URI: builder.context.DocumentURI, Range: node.Range, SelectionRange: nameRange, Role: SymbolLocationAttribute, Condition: builder.conditionForNode(node), SourceVersion: builder.snapshot.SourceVersion})
	} else if locationRole == SymbolLocationDefinition {
		symbol.Definitions = appendUniqueLocation(symbol.Definitions, location)
		if symbol.Status == SymbolStatusDeclarationOnly {
			symbol.Status = builder.symbolStatus(node)
		}
	} else {
		symbol.Declarations = appendUniqueLocation(symbol.Declarations, location)
		if len(symbol.Definitions) == 0 && symbol.Status != SymbolStatusIncomplete {
			symbol.Status = SymbolStatusDeclarationOnly
		}
	}
	builder.snapshot.Symbols[symbolID] = symbol
	builder.symbolsByName[name] = append(builder.symbolsByName[name], symbolID)
	parent := builder.snapshot.Scopes[parentScope]
	parent.Symbols = appendUniqueString(parent.Symbols, symbolID)
	builder.snapshot.Scopes[parentScope] = parent
	if nameRange.Start < nameRange.End {
		builder.declarationRanges = append(builder.declarationRanges, nameRange)
	}
	return symbolID
}

func (builder *builder) originKind(kind SymbolKind) string {
	if kind == SymbolKindBuiltin {
		return "builtin"
	}
	if builder.context.LanguageID == "tops-header" {
		return "header"
	}
	return "document"
}

func (builder *builder) collectReferences(tokens []parser.Token) {
	for tokenIndex, token := range tokens {
		if token.Kind != parser.TokenIdentifier || containsRange(builder.declarationRanges, token.Range) {
			continue
		}
		scopeID := builder.scopeAt(token.Range.Start)
		qualifiedName := qualifiedReferenceName(tokens, tokenIndex)
		candidates := []string(nil)
		if token.ConditionalState != parser.ConditionalInactive {
			candidates = candidateIDsForReference(builder.snapshot.Tokens, tokenIndex, &builder.snapshot, token.Spelling, qualifiedName, scopeID, token.ConditionalState)
		}
		if hasAttributeCandidate(&builder.snapshot, candidates) {
			continue
		}
		condition := builder.provenanceAt(token.Range.Start)
		if condition.State == "" {
			condition = ConditionalProvenance{SourceURI: builder.context.DocumentURI, State: token.ConditionalState}
		}
		condition.SourceURI = builder.context.DocumentURI
		reference := newReferenceRecord(builder.context.DocumentURI, token, qualifiedName, scopeID, candidates, condition, referenceKind(tokens, tokenIndex, &builder.snapshot, candidates), builder.snapshot.SourceVersion)
		reference.Builtin = builtinProvenanceForSnapshot(&builder.snapshot, candidates, token)
		reference = normalizeBuiltinReference(&builder.snapshot, reference)
		builder.snapshot.References[reference.ReferenceID] = reference
		attachReference(&builder.snapshot, reference)
	}
}

func collectAttributeReferencesForSnapshot(snapshot *IndexSnapshot) {
	ids := make([]string, 0, len(snapshot.Symbols))
	for id, symbol := range snapshot.Symbols {
		if symbol.Kind == SymbolKindAttribute {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		symbol := snapshot.Symbols[id]
		for _, usage := range symbol.AttributeUsages {
			selectionRange := usage.SelectionRange
			if selectionRange.Start == selectionRange.End {
				selectionRange = usage.Range
			}
			condition := usage.Condition
			if condition.State == "" {
				condition = symbol.Condition
			}
			reference := ReferenceRecord{
				ReferenceID:   fmt.Sprintf("attribute-ref:%s:%d:%d", usage.URI, selectionRange.Start, selectionRange.End),
				URI:           usage.URI,
				Range:         selectionRange,
				Name:          symbol.Name,
				QualifiedName: symbol.QualifiedName,
				ScopeID:       symbol.ScopeID,
				Candidates:    []string{id},
				Condition:     condition,
				Kind:          "attribute-use",
				Status:        referenceStatus(condition.State, []string{id}),
				SourceVersion: usage.SourceVersion,
			}
			snapshot.References[reference.ReferenceID] = reference
			attachReference(snapshot, reference)
		}
	}
}

func hasAttributeCandidate(snapshot *IndexSnapshot, candidates []string) bool {
	for _, candidate := range candidates {
		if symbol, ok := snapshot.Symbols[candidate]; ok && symbol.Kind == SymbolKindAttribute {
			return true
		}
	}
	return false
}

func candidateIDsForSnapshot(snapshot *IndexSnapshot, ids []string, name, qualifiedName, scopeID string, state parser.ConditionalState) []string {
	active := make([]string, 0, len(ids))
	unknown := make([]string, 0, len(ids))
	qualified := strings.Contains(qualifiedName, "::")
	for _, id := range ids {
		symbol := snapshot.Symbols[id]
		if symbol.Name != name || (qualifiedName != "" && qualified && qualifiedName != symbol.QualifiedName) {
			continue
		}
		if !qualified && scopeID != "" && !scopeIsVisible(snapshot.Scopes, scopeID, symbol.ScopeID) {
			continue
		}
		switch symbol.Condition.State {
		case parser.ConditionalUnknown:
			unknown = append(unknown, id)
		case parser.ConditionalInactive:
			continue
		default:
			active = append(active, id)
		}
	}
	if len(active) > 0 {
		sort.Strings(active)
		return active
	}
	if state != parser.ConditionalInactive && len(unknown) > 0 {
		sort.Strings(unknown)
		return unknown
	}
	return nil
}

func candidateIDsForReference(tokens []parser.Token, tokenIndex int, snapshot *IndexSnapshot, name, qualifiedName, scopeID string, state parser.ConditionalState) []string {
	if candidates, ok := memberCandidateIDs(tokens, tokenIndex, snapshot, name, scopeID, state); ok {
		return candidates
	}
	return candidateIDsForSnapshot(snapshot, symbolIDsByName(snapshot, name), name, qualifiedName, scopeID, state)
}

func memberCandidateIDs(tokens []parser.Token, tokenIndex int, snapshot *IndexSnapshot, name, scopeID string, state parser.ConditionalState) ([]string, bool) {
	receiverIndex, ok := memberReceiverIndex(tokens, tokenIndex)
	if !ok {
		return nil, false
	}

	typeScopes := make(map[string]struct{})
	receiver := tokens[receiverIndex].Spelling
	if receiver == "this" {
		for current := scopeID; current != ""; {
			scope, exists := snapshot.Scopes[current]
			if !exists {
				break
			}
			if scope.Kind == ScopeClass || scope.Kind == ScopeStruct || scope.Kind == ScopeUnion {
				typeScopes[current] = struct{}{}
			}
			current = scope.ParentID
		}
	} else {
		for _, receiverID := range symbolIDsByName(snapshot, receiver) {
			receiverSymbol, exists := snapshot.Symbols[receiverID]
			if !exists || (receiverSymbol.Kind != SymbolKindVariable && receiverSymbol.Kind != SymbolKindField) {
				continue
			}
			if !scopeIsVisible(snapshot.Scopes, scopeID, receiverSymbol.ScopeID) {
				continue
			}
			typeName := normalizeTypeName(receiverSymbol.TypeName)
			if typeName == "" {
				continue
			}
			for _, typeSymbol := range snapshot.Symbols {
				if typeSymbol.Kind != SymbolKindType || !typeNameMatches(typeSymbol, typeName) {
					continue
				}
				for typeScopeID, typeScope := range snapshot.Scopes {
					if typeScope.SourceURI == typeSymbol.SourceVersion.URI && typeScope.QualifiedName == typeSymbol.QualifiedName && (typeScope.Kind == ScopeClass || typeScope.Kind == ScopeStruct || typeScope.Kind == ScopeUnion) {
						typeScopes[typeScopeID] = struct{}{}
					}
				}
			}
		}
	}

	fieldIDs := make([]string, 0)
	for id, symbol := range snapshot.Symbols {
		if symbol.Kind != SymbolKindField || symbol.Name != name {
			continue
		}
		if _, exists := typeScopes[symbol.ScopeID]; exists {
			fieldIDs = append(fieldIDs, id)
		}
	}
	return conditionalCandidates(snapshot, fieldIDs, state), true
}

func memberReceiverIndex(tokens []parser.Token, tokenIndex int) (int, bool) {
	operatorIndex := previousSignificantToken(tokens, tokenIndex)
	if operatorIndex < 0 || (tokens[operatorIndex].Spelling != "." && tokens[operatorIndex].Spelling != "->") {
		return 0, false
	}
	receiverIndex := previousSignificantToken(tokens, operatorIndex)
	if receiverIndex < 0 || (tokens[receiverIndex].Kind != parser.TokenIdentifier && tokens[receiverIndex].Spelling != "this") {
		return 0, false
	}
	return receiverIndex, true
}

func previousSignificantToken(tokens []parser.Token, index int) int {
	for index--; index >= 0; index-- {
		if tokens[index].Kind != parser.TokenComment && tokens[index].Kind != parser.TokenNewline {
			return index
		}
	}
	return -1
}

func conditionalCandidates(snapshot *IndexSnapshot, ids []string, state parser.ConditionalState) []string {
	active := make([]string, 0, len(ids))
	unknown := make([]string, 0, len(ids))
	for _, id := range ids {
		symbol, ok := snapshot.Symbols[id]
		if !ok {
			continue
		}
		switch symbol.Condition.State {
		case parser.ConditionalInactive:
			continue
		case parser.ConditionalUnknown:
			unknown = append(unknown, id)
		default:
			active = append(active, id)
		}
	}
	if len(active) > 0 {
		sort.Strings(active)
		return active
	}
	if state != parser.ConditionalInactive {
		sort.Strings(unknown)
		return unknown
	}
	return nil
}

func scopeIsVisible(scopes map[string]ScopeRecord, fromID, targetID string) bool {
	if targetID == "" {
		return false
	}
	target, ok := scopes[targetID]
	if !ok {
		return false
	}
	if target.Kind == ScopeTranslationUnit || target.Kind == ScopeHeader {
		return true
	}
	for current := fromID; current != ""; {
		if current == targetID {
			return true
		}
		scope, ok := scopes[current]
		if !ok {
			break
		}
		current = scope.ParentID
	}
	return false
}

func (builder *builder) scopeAt(offset int) string {
	best := ""
	bestSize := int(^uint(0) >> 1)
	for id, scope := range builder.snapshot.Scopes {
		if scope.SourceURI != "" && scope.SourceURI != builder.context.DocumentURI {
			continue
		}
		if scope.Range.Start <= offset && offset < scope.Range.End {
			size := scope.Range.End - scope.Range.Start
			if size < bestSize {
				best = id
				bestSize = size
			}
		}
	}
	return best
}

func (builder *builder) enclosingScopeForNode(node *parser.SyntaxNode, fallback string) string {
	if node == nil {
		return fallback
	}
	current := builder.scopeAt(node.Range.Start)
	for current != "" {
		scope, ok := builder.snapshot.Scopes[current]
		if !ok {
			break
		}
		if scope.Range.Start != node.Range.Start || scope.Range.End != node.Range.End {
			return current
		}
		current = scope.ParentID
	}
	return fallback
}

func qualifiedReferenceName(tokens []parser.Token, index int) string {
	if index < 0 || index >= len(tokens) {
		return ""
	}
	parts := []string{tokens[index].Spelling}
	for current := index; current >= 2 && tokens[current-1].Spelling == "::"; current -= 2 {
		if tokens[current-2].Kind != parser.TokenIdentifier && tokens[current-2].Kind != parser.TokenKeyword {
			break
		}
		parts = append([]string{tokens[current-2].Spelling}, parts...)
	}
	return strings.Join(parts, "::")
}

func (builder *builder) finishChildren() {
	for id, symbol := range builder.snapshot.Symbols {
		if len(symbol.Definitions) == 0 && len(symbol.Declarations) > 0 && symbol.Status == SymbolStatusActive {
			symbol.Status = SymbolStatusDeclarationOnly
			builder.snapshot.Symbols[id] = symbol
		}
	}
}

func (builder *builder) qualifiedName(scopeID, name string) string {
	scope := builder.snapshot.Scopes[scopeID]
	if scope.QualifiedName == "" {
		return name
	}
	return scope.QualifiedName + "::" + name
}

func (builder *builder) scopeID(qualified string, sourceRange parser.SourceRange) string {
	return fmt.Sprintf("scope:%s:%s:%d:%d", builder.context.DocumentURI, qualified, sourceRange.Start, sourceRange.End)
}

func (builder *builder) declarationAnchor(node *parser.SyntaxNode) string {
	tokens := node.Tokens
	if len(tokens) == 0 {
		tokens = append(append([]parser.Token(nil), node.TypeTokens...), node.DeclaratorTokens...)
	}
	if len(tokens) == 0 {
		return node.Name
	}
	parts := make([]string, 0, len(tokens))
	parameterDepth := 0
	for index, token := range tokens {
		if token.Spelling == "{" {
			break
		}
		if token.Spelling == ";" {
			continue
		}
		if token.Kind == parser.TokenComment || token.Kind == parser.TokenNewline {
			continue
		}
		if node.Kind == parser.NodeFunction {
			if token.Spelling == "(" {
				parameterDepth++
			} else if token.Spelling == ")" && parameterDepth > 0 {
				parameterDepth--
			}
			if parameterDepth == 1 && isParameterName(tokens, index) {
				continue
			}
		}
		parts = append(parts, token.Spelling)
	}
	return strings.Join(parts, " ")
}

func isParameterName(tokens []parser.Token, index int) bool {
	if index <= 0 || tokens[index].Kind != parser.TokenIdentifier {
		return false
	}
	previous := tokens[index-1].Spelling
	if previous == "(" || previous == "," || previous == "::" {
		return false
	}
	if index+1 >= len(tokens) {
		return true
	}
	next := tokens[index+1].Spelling
	if next == "::" {
		return false
	}
	return next == "," || next == ")" || next == "=" || next == "[" || next == "..."
}

func (builder *builder) nodeDetail(node *parser.SyntaxNode) string {
	return builder.declarationAnchor(node)
}

func parameterSummary(node *parser.SyntaxNode) string {
	if node.Kind != parser.NodeFunction {
		return ""
	}
	parts := make([]string, 0, len(node.DeclaratorTokens))
	depth := 0
	started := false
	for _, token := range node.DeclaratorTokens {
		if token.Spelling == "(" {
			depth++
			started = true
			continue
		}
		if token.Spelling == ")" {
			depth--
			if started && depth == 0 {
				break
			}
		}
		if started {
			parts = append(parts, token.Spelling)
		}
	}
	return strings.Join(parts, " ")
}

func returnTypeSummary(node *parser.SyntaxNode) string {
	if node.Kind != parser.NodeFunction || len(node.TypeTokens) == 0 {
		return ""
	}
	parts := make([]string, 0, len(node.TypeTokens))
	for _, token := range node.TypeTokens {
		if strings.HasPrefix(token.Spelling, "__") {
			continue
		}
		parts = append(parts, token.Spelling)
	}
	return strings.Join(parts, " ")
}

func declaredTypeName(node *parser.SyntaxNode) string {
	if node == nil || len(node.TypeTokens) == 0 {
		return ""
	}
	parts := make([]string, 0, len(node.TypeTokens))
	for _, token := range node.TypeTokens {
		if strings.HasPrefix(token.Spelling, "__") {
			continue
		}
		parts = append(parts, token.Spelling)
	}
	return strings.Join(parts, " ")
}

func typeNameMatches(symbol SymbolRecord, typeName string) bool {
	return normalizeTypeName(symbol.Name) == typeName || normalizeTypeName(symbol.QualifiedName) == typeName
}

func normalizeTypeName(typeName string) string {
	parts := strings.Fields(typeName)
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "const", "volatile", "struct", "class", "union", "enum":
			continue
		default:
			filtered = append(filtered, part)
		}
	}
	return strings.Trim(strings.Join(filtered, ""), "*&")
}

func executionQualifiers(node *parser.SyntaxNode) []string {
	qualifiers := make([]string, 0)
	for _, child := range node.Children {
		if child.Kind == parser.NodeTopsQualifier {
			qualifiers = append(qualifiers, child.Name)
		}
	}
	return qualifiers
}

func (builder *builder) conditionForNode(node *parser.SyntaxNode) ConditionalProvenance {
	var condition ConditionalProvenance
	if provenance := builder.provenanceAt(node.Range.Start); provenance.State != "" {
		condition = provenance
	} else if builder.conditionOverride.State != "" {
		condition = builder.conditionOverride
	} else {
		condition = ConditionalProvenance{SourceURI: builder.context.DocumentURI, State: node.ConditionalState}
	}
	condition.SourceURI = builder.context.DocumentURI
	return condition
}

func conditionalProvenance(sourceURI string, regionIndex, branchIndex int, region parser.ConditionalRegion, branch parser.ConditionalBranch) ConditionalProvenance {
	return ConditionalProvenance{
		SourceURI:       sourceURI,
		RegionID:        regionIndex,
		BranchIndex:     branchIndex,
		ExpressionRange: conditionExpressionRange(region, branch),
		State:           branch.State,
		ParentRegionID:  region.ParentID,
		ConditionTokens: append([]parser.Token(nil), branch.Condition...),
	}
}

func conditionExpressionRange(region parser.ConditionalRegion, branch parser.ConditionalBranch) parser.SourceRange {
	if len(branch.Condition) > 0 {
		return parser.SourceRange{Start: branch.Condition[0].Range.Start, End: branch.Condition[len(branch.Condition)-1].Range.End}
	}
	if branch.DirectiveRange.Start < branch.DirectiveRange.End {
		return branch.DirectiveRange
	}
	return region.ConditionRange
}

func (builder *builder) provenanceAt(offset int) ConditionalProvenance {
	return provenanceAtRegions(builder.conditionalRegions, offset, builder.context.DocumentURI)
}

func (builder *builder) symbolStatus(node *parser.SyntaxNode) SymbolStatus {
	incompleteAtEOF := builder.snapshot.ParseStatus != parser.ParseStatusComplete && node.Kind == parser.NodeFunction && node.Range.End >= len(builder.text) && node.BodyRange.Start == node.BodyRange.End
	if node.Incomplete || node.DeclarationRole == parser.DeclarationRoleUnknown || incompleteAtEOF {
		return SymbolStatusIncomplete
	}
	if builder.conditionForNode(node).State == parser.ConditionalUnknown {
		return SymbolStatusConditional
	}
	return SymbolStatusActive
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueLocation(values []SymbolLocation, value SymbolLocation) []SymbolLocation {
	for _, existing := range values {
		if existing.URI == value.URI && existing.Range == value.Range && existing.Role == value.Role {
			return values
		}
	}
	return append(values, value)
}

func containsRange(ranges []parser.SourceRange, candidate parser.SourceRange) bool {
	for _, current := range ranges {
		if current.Start == candidate.Start && current.End == candidate.End {
			return true
		}
	}
	return false
}
