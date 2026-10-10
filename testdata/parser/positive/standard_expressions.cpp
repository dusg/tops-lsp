template <typename T>
T transform(T value) {
  if (value > 0) return [value](T extra) { return value + extra; }(1);
  for (int index = 0; index < 2; ++index) value += index;
  return value;
}
