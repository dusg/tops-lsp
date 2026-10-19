package index

import (
	"sort"
	"tops-lsp/internal/parser"
)

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
