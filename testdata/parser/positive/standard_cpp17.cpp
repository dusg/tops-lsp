int read_pair() {
  auto [first, second] = pair;
  if constexpr (first) {
    return second;
  }
  return first;
}