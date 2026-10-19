package semantic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"tops-lsp/internal/index"
)

// SourceVersionStatus identifies a dependency whose version cannot be used.
type SourceVersionStatus string

const (
	SourceVersionStatusMissing    SourceVersionStatus = "missing"
	SourceVersionStatusUnresolved SourceVersionStatus = "unresolved"
	SourceVersionStatusUnreadable SourceVersionStatus = "unreadable"
	SourceVersionStatusStale      SourceVersionStatus = "stale"
)

var sourceVersionStatuses = map[SourceVersionStatus]struct{}{
	SourceVersionStatusMissing:    {},
	SourceVersionStatusUnresolved: {},
	SourceVersionStatusUnreadable: {},
	SourceVersionStatusStale:      {},
}

// SourceVersionEntry is one source or header input to AggregateSourceVersion.
// Exactly one of Version and Status must be populated.
type SourceVersionEntry struct {
	URI     string
	Version string
	Status  SourceVersionStatus
}

type sourceVersionValues struct {
	versions map[string]struct{}
	statuses map[SourceVersionStatus]struct{}
}

// AggregateSourceVersion creates the deterministic source/header fingerprint.
func AggregateSourceVersion(documentURI string, entries []SourceVersionEntry) (string, error) {
	rootURI := index.NormalizeSourceURI(documentURI)
	if rootURI == "" {
		return "", fmt.Errorf("document URI is not normalizable")
	}
	if !utf8.ValidString(documentURI) {
		return "", fmt.Errorf("document URI is not valid UTF-8")
	}

	values := make(map[string]*sourceVersionValues, len(entries))
	rootFound := false
	rootHasVersion := false
	for _, entry := range entries {
		if !utf8.ValidString(entry.URI) || !utf8.ValidString(entry.Version) {
			return "", fmt.Errorf("source version entry is not valid UTF-8")
		}
		uri := index.NormalizeSourceURI(entry.URI)
		if uri == "" {
			return "", fmt.Errorf("source version entry URI is not normalizable")
		}
		hasVersion := entry.Version != ""
		hasStatus := entry.Status != ""
		if hasVersion == hasStatus {
			return "", fmt.Errorf("source version entry %q must have exactly one of version or status", uri)
		}
		if hasStatus {
			if !utf8.ValidString(string(entry.Status)) {
				return "", fmt.Errorf("source version entry %q status is not valid UTF-8", uri)
			}
			if _, ok := sourceVersionStatuses[entry.Status]; !ok {
				return "", fmt.Errorf("source version entry %q has invalid status %q", uri, entry.Status)
			}
		}
		if uri == rootURI {
			rootFound = true
			rootHasVersion = rootHasVersion || hasVersion
		}
		current := values[uri]
		if current == nil {
			current = &sourceVersionValues{versions: make(map[string]struct{}), statuses: make(map[SourceVersionStatus]struct{})}
			values[uri] = current
		}
		if hasVersion {
			current.versions[entry.Version] = struct{}{}
		} else {
			current.statuses[entry.Status] = struct{}{}
		}
	}
	if !rootFound {
		return "", fmt.Errorf("root source %q is missing from source versions", rootURI)
	}
	if !rootHasVersion {
		return "", fmt.Errorf("root source %q must have a version", rootURI)
	}

	uris := make([]string, 0, len(values))
	for uri := range values {
		uris = append(uris, uri)
	}
	sort.Strings(uris)
	var builder strings.Builder
	for _, uri := range uris {
		current := values[uri]
		if len(current.versions) == 1 && len(current.statuses) == 0 {
			for version := range current.versions {
				fmt.Fprintf(&builder, "%s\x00version:%s\n", uri, version)
			}
			continue
		}
		if len(current.versions) == 0 && len(current.statuses) == 1 {
			for status := range current.statuses {
				fmt.Fprintf(&builder, "%s\x00status:%s\n", uri, status)
			}
			continue
		}
		fmt.Fprintf(&builder, "%s\x00status:conflict\n", uri)
	}
	digest := sha256.Sum256([]byte(builder.String()))
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
