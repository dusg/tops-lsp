package semantic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

var commandFieldKeys = []string{
	"source_uri",
	"working_directory",
	"raw_arguments",
	"normalized_arguments",
	"arch",
	"driver",
	"language",
	"language_standard",
	"pass_kind",
	"toolchain_include_roots",
	"workspace_include_roots",
	"toolchain_macros",
	"workspace_macros",
}

func CommandFieldKeys() []string {
	return append([]string(nil), commandFieldKeys...)
}

func CanonicalJSON(value any) ([]byte, error) {
	if raw, ok := value.(json.RawMessage); ok {
		parsed, err := parseRawJSON(raw)
		if err != nil {
			return nil, err
		}
		value = parsed
	}
	var output bytes.Buffer
	if err := writeCanonicalJSON(&output, value); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func parseRawJSON(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := parseJSONValue(decoder)
	if err != nil {
		return nil, err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("trailing JSON token %v", token)
		}
		return nil, err
	}
	return value, nil
}

func parseJSONValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	switch token := token.(type) {
	case json.Delim:
		switch token {
		case '{':
			object := make(map[string]any)
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, fmt.Errorf("JSON object key is not a string")
				}
				if _, exists := object[key]; exists {
					return nil, fmt.Errorf("duplicate JSON object key %q", key)
				}
				value, err := parseJSONValue(decoder)
				if err != nil {
					return nil, err
				}
				object[key] = value
			}
			if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
				if err != nil {
					return nil, err
				}
				return nil, fmt.Errorf("JSON object did not close")
			}
			return object, nil
		case '[':
			array := make([]any, 0)
			for decoder.More() {
				value, err := parseJSONValue(decoder)
				if err != nil {
					return nil, err
				}
				array = append(array, value)
			}
			if end, err := decoder.Token(); err != nil || end != json.Delim(']') {
				if err != nil {
					return nil, err
				}
				return nil, fmt.Errorf("JSON array did not close")
			}
			return array, nil
		default:
			return nil, fmt.Errorf("unexpected JSON delimiter %q", token)
		}
	case string, bool, nil, json.Number:
		return token, nil
	default:
		return nil, fmt.Errorf("unsupported JSON token %T", token)
	}
}

func writeCanonicalJSON(output *bytes.Buffer, value any) error {
	if value == nil {
		output.WriteString("null")
		return nil
	}
	switch value := value.(type) {
	case string:
		return writeJSONString(output, value)
	case bool:
		if value {
			output.WriteString("true")
		} else {
			output.WriteString("false")
		}
		return nil
	case json.Number:
		number, err := normalizeJSONNumber(string(value))
		if err != nil {
			return err
		}
		output.WriteString(number)
		return nil
	case float32:
		return writeCanonicalJSON(output, float64(value))
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("non-finite JSON number")
		}
		return writeCanonicalJSON(output, json.Number(strconv.FormatFloat(value, 'g', -1, 64)))
	case int:
		output.WriteString(strconv.FormatInt(int64(value), 10))
		return nil
	case int8:
		output.WriteString(strconv.FormatInt(int64(value), 10))
		return nil
	case int16:
		output.WriteString(strconv.FormatInt(int64(value), 10))
		return nil
	case int32:
		output.WriteString(strconv.FormatInt(int64(value), 10))
		return nil
	case int64:
		output.WriteString(strconv.FormatInt(value, 10))
		return nil
	case uint:
		output.WriteString(strconv.FormatUint(uint64(value), 10))
		return nil
	case uint8:
		output.WriteString(strconv.FormatUint(uint64(value), 10))
		return nil
	case uint16:
		output.WriteString(strconv.FormatUint(uint64(value), 10))
		return nil
	case uint32:
		output.WriteString(strconv.FormatUint(uint64(value), 10))
		return nil
	case uint64:
		output.WriteString(strconv.FormatUint(value, 10))
		return nil
	case []byte:
		return writeReflectJSON(output, reflect.ValueOf(value))
	case json.RawMessage:
		parsed, err := parseRawJSON(value)
		if err != nil {
			return err
		}
		return writeCanonicalJSON(output, parsed)
	case []any:
		output.WriteByte('[')
		for index, item := range value {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := writeCanonicalJSON(output, item); err != nil {
				return err
			}
		}
		output.WriteByte(']')
		return nil
	case map[string]any:
		return writeCanonicalObject(output, value)
	default:
		return writeReflectJSON(output, reflect.ValueOf(value))
	}
}

func writeCanonicalObject(output *bytes.Buffer, value map[string]any) error {
	keys := make([]string, 0, len(value))
	for key := range value {
		if !utf8.ValidString(key) {
			return fmt.Errorf("JSON object key is not valid UTF-8")
		}
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool { return strings.Compare(keys[left], keys[right]) < 0 })
	output.WriteByte('{')
	for index, key := range keys {
		if index > 0 {
			output.WriteByte(',')
		}
		if err := writeJSONString(output, key); err != nil {
			return err
		}
		output.WriteByte(':')
		if err := writeCanonicalJSON(output, value[key]); err != nil {
			return err
		}
	}
	output.WriteByte('}')
	return nil
}

func writeReflectJSON(output *bytes.Buffer, value reflect.Value) error {
	if !value.IsValid() {
		output.WriteString("null")
		return nil
	}
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			output.WriteString("null")
			return nil
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		output.WriteByte('[')
		for index := 0; index < value.Len(); index++ {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := writeCanonicalJSON(output, value.Index(index).Interface()); err != nil {
				return err
			}
		}
		output.WriteByte(']')
		return nil
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("JSON object key is not a string")
		}
		object := make(map[string]any, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			object[iter.Key().String()] = iter.Value().Interface()
		}
		return writeCanonicalObject(output, object)
	default:
		return fmt.Errorf("unsupported canonical JSON value %T", value.Interface())
	}
}

func writeJSONString(output *bytes.Buffer, value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("JSON string is not valid UTF-8")
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return err
	}
	output.Write(bytes.TrimSuffix(encoded.Bytes(), []byte{'\n'}))
	return nil
}

func normalizeJSONNumber(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("empty JSON number")
	}
	negative := false
	if value[0] == '-' {
		negative = true
		value = value[1:]
	}
	if value == "" {
		return "", fmt.Errorf("invalid JSON number")
	}
	exponent := 0
	if marker := strings.IndexAny(value, "eE"); marker >= 0 {
		exponentValue, err := strconv.Atoi(value[marker+1:])
		if err != nil {
			return "", fmt.Errorf("invalid JSON exponent: %w", err)
		}
		exponent = exponentValue
		value = value[:marker]
	}
	integerPart, fractionalPart, hasFraction := value, "", false
	if dot := strings.IndexByte(value, '.'); dot >= 0 {
		if strings.IndexByte(value[dot+1:], '.') >= 0 {
			return "", fmt.Errorf("invalid JSON number")
		}
		integerPart, fractionalPart, hasFraction = value[:dot], value[dot+1:], true
	}
	if integerPart == "" || (len(integerPart) > 1 && integerPart[0] == '0') || (hasFraction && fractionalPart == "") {
		return "", fmt.Errorf("invalid JSON number")
	}
	for _, digit := range integerPart + fractionalPart {
		if digit < '0' || digit > '9' {
			return "", fmt.Errorf("invalid JSON number")
		}
	}
	digits := integerPart + fractionalPart
	decimalPosition := len(integerPart) + exponent
	leading := 0
	for leading < len(digits)-1 && digits[leading] == '0' {
		leading++
	}
	digits = digits[leading:]
	decimalPosition -= leading
	if strings.Trim(digits, "0") == "" {
		return "0", nil
	}
	for len(digits) > 1 && digits[len(digits)-1] == '0' {
		digits = digits[:len(digits)-1]
	}
	var result string
	switch {
	case decimalPosition <= 0:
		result = "0." + strings.Repeat("0", -decimalPosition) + digits
	case decimalPosition >= len(digits):
		result = digits + strings.Repeat("0", decimalPosition-len(digits))
	default:
		result = digits[:decimalPosition] + "." + digits[decimalPosition:]
	}
	if negative {
		return "-" + result, nil
	}
	return result, nil
}

func ExpandResponseFiles(arguments []string, workingDirectory string) ([]string, error) {
	if workingDirectory == "" {
		return nil, fmt.Errorf("response file working directory is empty")
	}
	if !filepath.IsAbs(workingDirectory) {
		absolute, err := filepath.Abs(workingDirectory)
		if err != nil {
			return nil, err
		}
		workingDirectory = absolute
	}
	return expandResponseArguments(arguments, workingDirectory, 0, make(map[string]bool))
}

func expandResponseArguments(arguments []string, workingDirectory string, depth int, stack map[string]bool) ([]string, error) {
	if depth > 16 {
		return nil, fmt.Errorf("response file nesting exceeds 16 levels")
	}
	result := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		if len(argument) < 2 || argument[0] != '@' {
			result = append(result, argument)
			continue
		}
		path := argument[1:]
		if path == "" {
			return nil, fmt.Errorf("empty response file path")
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(workingDirectory, path)
		}
		path, err := filepath.Abs(filepath.Clean(path))
		if err != nil {
			return nil, err
		}
		if stack[path] {
			return nil, fmt.Errorf("response file cycle at %q", path)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read response file %q: %w", path, err)
		}
		if !utf8.Valid(body) {
			return nil, fmt.Errorf("response file %q is not valid UTF-8", path)
		}
		tokens, err := tokenizeResponseFile(string(body))
		if err != nil {
			return nil, fmt.Errorf("parse response file %q: %w", path, err)
		}
		stack[path] = true
		nested, err := expandResponseArguments(tokens, workingDirectory, depth+1, stack)
		delete(stack, path)
		if err != nil {
			return nil, err
		}
		result = append(result, nested...)
	}
	return result, nil
}

func tokenizeResponseFile(body string) ([]string, error) {
	var tokens []string
	var token strings.Builder
	inToken := false
	var quote rune
	for index := 0; index < len(body); index++ {
		character := rune(body[index])
		if character == '\\' {
			if index+1 >= len(body) {
				return nil, fmt.Errorf("trailing escape")
			}
			index++
			token.WriteByte(body[index])
			inToken = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
				inToken = true
				continue
			}
			token.WriteByte(body[index])
			inToken = true
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			inToken = true
			continue
		}
		if character == ' ' || character == '\t' || character == '\r' || character == '\n' {
			if inToken {
				tokens = append(tokens, token.String())
				token.Reset()
				inToken = false
			}
			continue
		}
		token.WriteByte(body[index])
		inToken = true
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote")
	}
	if inToken {
		tokens = append(tokens, token.String())
	}
	return tokens, nil
}
