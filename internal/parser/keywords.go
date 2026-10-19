package parser

import "sort"

// CppKeywords returns a sorted copy of the standard C++ keyword spellings.
func CppKeywords() []string {
	return sortedSpellings(keywordSpellings)
}

// TopsKeywords returns a sorted copy of the Tops keyword spellings.
func TopsKeywords() []string {
	return sortedSpellings(topsKeywordSpellings)
}

func sortedSpellings(spellings map[string]bool) []string {
	result := make([]string, 0, len(spellings))
	for spelling := range spellings {
		result = append(result, spelling)
	}
	sort.Strings(result)
	return result
}
