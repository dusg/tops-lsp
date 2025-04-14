package lsp

type Set[T comparable] map[T]struct{}

func MakeSet[T comparable](keys ...T) Set[T] {
	set := make(map[T]struct{})

	for _, key := range keys {
		set[key] = struct{}{}
	}

	return set
}
func (s Set[T]) Has(key T) bool {
	_, ok := s[key]
	return ok
}
func (s Set[T]) Add(key T) {
	s[key] = struct{}{}
}
