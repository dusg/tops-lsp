package position

import "testing"

func TestMapperUsesUTF16AndPreservesCRLFLineEnd(t *testing.T) {
	mapper := New("A😀\r\n尾")

	position, err := mapper.PositionAt(len("A😀"))
	if err != nil {
		t.Fatalf("PositionAt() error = %v", err)
	}
	if position != (Position{Line: 0, Character: 3}) {
		t.Fatalf("PositionAt() = %+v", position)
	}

	offset, err := mapper.OffsetAt(Position{Line: 0, Character: 3})
	if err != nil {
		t.Fatalf("OffsetAt() error = %v", err)
	}
	if offset != len("A😀") {
		t.Fatalf("OffsetAt() = %d, want %d", offset, len("A😀"))
	}

	lineEnd, err := mapper.OffsetAt(Position{Line: 0, Character: 3})
	if err != nil {
		t.Fatalf("line end OffsetAt() error = %v", err)
	}
	if lineEnd != 5 {
		t.Fatalf("line end offset = %d, want 5", lineEnd)
	}
}

func TestMapperRejectsInvalidUTF8AndOutOfRangePositions(t *testing.T) {
	mapper := New("ok")
	if _, err := mapper.PositionAt(3); err == nil {
		t.Fatal("PositionAt() accepted an out-of-range offset")
	}
	if _, err := mapper.OffsetAt(Position{Line: 1, Character: 0}); err == nil {
		t.Fatal("OffsetAt() accepted an out-of-range line")
	}

	invalid := New(string([]byte{'o', 0xff}))
	if _, err := invalid.PositionAt(2); err == nil {
		t.Fatal("PositionAt() accepted invalid UTF-8")
	}
}
