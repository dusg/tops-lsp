package lsp

import (
	"reflect"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    `gcc -o output file.c`,
			expected: []string{"gcc", "-o", "output", "file.c"},
		},
		{
			input:    `gcc --flag="value with spaces" file.c`,
			expected: []string{"gcc", `--flag="value with spaces"`, "file.c"},
		},
		{
			input:    `gcc -DNAME="value" file.c`,
			expected: []string{"gcc", `-DNAME="value"`, "file.c"},
		},
		{
			input:    `gcc -I"path/to/include" file.c`,
			expected: []string{"gcc", `-I"path/to/include"`, "file.c"},
		},
		{
			input:    `gcc -o "output file" "source file.c"`,
			expected: []string{"gcc", "-o", "output file", "source file.c"},
		},
	}

	for _, test := range tests {
		result := parseCommand(test.input)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("parseCommand(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}
