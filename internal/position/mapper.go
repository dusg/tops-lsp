package position

import (
	"errors"
	"sort"
	"unicode/utf8"
)

var (
	ErrInvalidUTF8     = errors.New("source text is not valid UTF-8")
	ErrInvalidOffset   = errors.New("source byte offset is invalid")
	ErrInvalidPosition = errors.New("source position is invalid")
	ErrInvalidRange    = errors.New("source range is invalid")
)

type Position struct {
	Line      int
	Character int
}

type Range struct {
	Start Position
	End   Position
}

type Mapper struct {
	text       string
	lineStarts []int
	lineEnds   []int
}

func New(text string) *Mapper {
	mapper := &Mapper{text: text, lineStarts: []int{0}}
	for offset := 0; offset < len(text); offset++ {
		if text[offset] != '\n' {
			continue
		}
		lineEnd := offset
		if lineEnd > mapper.lineStarts[len(mapper.lineStarts)-1] && text[lineEnd-1] == '\r' {
			lineEnd--
		}
		mapper.lineEnds = append(mapper.lineEnds, lineEnd)
		mapper.lineStarts = append(mapper.lineStarts, offset+1)
	}
	lastStart := mapper.lineStarts[len(mapper.lineStarts)-1]
	lastEnd := len(text)
	if lastEnd > lastStart && text[lastEnd-1] == '\r' {
		lastEnd--
	}
	mapper.lineEnds = append(mapper.lineEnds, lastEnd)
	return mapper
}

func (mapper *Mapper) PositionAt(offset int) (Position, error) {
	if mapper == nil || !utf8.ValidString(mapper.text) {
		return Position{}, ErrInvalidUTF8
	}
	if offset < 0 || offset > len(mapper.text) {
		return Position{}, ErrInvalidOffset
	}

	line := sort.Search(len(mapper.lineStarts), func(index int) bool {
		return mapper.lineStarts[index] > offset
	}) - 1
	if line < 0 {
		line = 0
	}
	if offset > mapper.lineEnds[line] {
		if line+1 < len(mapper.lineStarts) && offset == mapper.lineStarts[line+1] {
			return Position{Line: line + 1}, nil
		}
		return Position{}, ErrInvalidOffset
	}
	character, err := utf16Length(mapper.text[mapper.lineStarts[line]:offset])
	if err != nil {
		return Position{}, err
	}
	return Position{Line: line, Character: character}, nil
}

func (mapper *Mapper) OffsetAt(position Position) (int, error) {
	if mapper == nil || !utf8.ValidString(mapper.text) {
		return 0, ErrInvalidUTF8
	}
	if position.Line < 0 || position.Line >= len(mapper.lineStarts) || position.Character < 0 {
		return 0, ErrInvalidPosition
	}

	start := mapper.lineStarts[position.Line]
	end := mapper.lineEnds[position.Line]
	if position.Character == 0 {
		return start, nil
	}
	units := 0
	for offset := start; offset < end; {
		runeValue, size := utf8.DecodeRuneInString(mapper.text[offset:end])
		if runeValue == utf8.RuneError && size == 1 {
			return 0, ErrInvalidUTF8
		}
		runeUnits := 1
		if runeValue > 0xffff {
			runeUnits = 2
		}
		if units+runeUnits > position.Character {
			return 0, ErrInvalidPosition
		}
		units += runeUnits
		offset += size
		if units == position.Character {
			return offset, nil
		}
	}
	if units != position.Character {
		return 0, ErrInvalidPosition
	}
	return end, nil
}

func (mapper *Mapper) Range(startOffset, endOffset int) (Range, error) {
	if startOffset < 0 || endOffset < startOffset || endOffset > len(mapper.text) {
		return Range{}, ErrInvalidRange
	}
	start, err := mapper.PositionAt(startOffset)
	if err != nil {
		return Range{}, err
	}
	end, err := mapper.PositionAt(endOffset)
	if err != nil {
		return Range{}, err
	}
	return Range{Start: start, End: end}, nil
}

func UTF16Length(text string) (int, error) {
	return utf16Length(text)
}

func utf16Length(text string) (int, error) {
	if !utf8.ValidString(text) {
		return 0, ErrInvalidUTF8
	}
	length := 0
	for offset := 0; offset < len(text); {
		runeValue, size := utf8.DecodeRuneInString(text[offset:])
		if runeValue == utf8.RuneError && size == 1 {
			return 0, ErrInvalidUTF8
		}
		if runeValue > 0xffff {
			length += 2
		} else {
			length++
		}
		offset += size
	}
	return length, nil
}
