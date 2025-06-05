#pragma once
#include "llvm/Support/FormatVariadic.h"
#include <llvm/Support/Error.h>
#include <llvm/Support/FormatAdapters.h>

#include <optional>
#include <string>
#include <system_error>
#include <tuple>
namespace clang::clangd {
struct Position {
  /// Line position in a document (zero-based).
  int line = 0;

  /// Character offset on a line in a document (zero-based).
  /// WARNING: this is in UTF-16 codepoints, not bytes or characters!
  /// Use the functions in SourceCode.h to construct/interpret Positions.
  int character = 0;

  friend bool operator==(const Position &LHS, const Position &RHS) {
    return std::tie(LHS.line, LHS.character) == std::tie(RHS.line, RHS.character);
  }
  friend bool operator!=(const Position &LHS, const Position &RHS) { return !(LHS == RHS); }
  friend bool operator<(const Position &LHS, const Position &RHS) {
    return std::tie(LHS.line, LHS.character) < std::tie(RHS.line, RHS.character);
  }
  friend bool operator<=(const Position &LHS, const Position &RHS) {
    return std::tie(LHS.line, LHS.character) <= std::tie(RHS.line, RHS.character);
  }
};

struct Range {
  /// The range's start position.
  Position start;

  /// The range's end position.
  Position end;

  friend bool operator==(const Range &LHS, const Range &RHS) {
    return std::tie(LHS.start, LHS.end) == std::tie(RHS.start, RHS.end);
  }
  friend bool operator!=(const Range &LHS, const Range &RHS) { return !(LHS == RHS); }
  friend bool operator<(const Range &LHS, const Range &RHS) {
    return std::tie(LHS.start, LHS.end) < std::tie(RHS.start, RHS.end);
  }

  bool contains(Position Pos) const { return start <= Pos && Pos < end; }
  bool contains(Range Rng) const { return start <= Rng.start && Rng.end <= end; }
};
using ChangeAnnotationIdentifier = std::string;

struct TextEdit {
  /// The range of the text document to be manipulated. To insert
  /// text into a document create a range where start === end.
  Range range;

  /// The string to be inserted. For delete operations use an
  /// empty string.
  std::string newText;

  /// The actual annotation identifier (optional)
  /// If empty, then this field is nullopt.
  ChangeAnnotationIdentifier annotationId = "";
};

enum DiagnosticTag {
  /// Unused or unnecessary code.
  ///
  /// Clients are allowed to render diagnostics with this tag faded out instead
  /// of having an error squiggle.
  Unnecessary = 1,
  /// Deprecated or obsolete code.
  ///
  /// Clients are allowed to rendered diagnostics with this tag strike through.
  Deprecated = 2,
};

struct ChangeAnnotation {
  /// A human-readable string describing the actual change. The string
  /// is rendered prominent in the user interface.
  std::string label;

  /// A flag which indicates that user confirmation is needed
  /// before applying the change.
  std::optional<bool> needsConfirmation;

  /// A human-readable string which is rendered less prominent in
  /// the user interface.
  std::string description;
};

enum class OffsetEncoding {
  // Any string is legal on the wire. Unrecognized encodings parse as this.
  UnsupportedEncoding,
  // Length counts code units of UTF-16 encoded text. (Standard LSP behavior).
  UTF16,
  // Length counts bytes of UTF-8 encoded text. (Clangd extension).
  UTF8,
  // Length counts codepoints in unicode text. (Clangd extension).
  UTF32,
};

template <class Type> class Key {
public:
  static_assert(!std::is_reference<Type>::value, "Reference arguments to Key<> are not allowed");

  constexpr Key() = default;

  Key(Key const &) = delete;
  Key &operator=(Key const &) = delete;
  Key(Key &&) = delete;
  Key &operator=(Key &&) = delete;
};

// Like llvm::StringError but with fewer options and no gratuitous copies.
class SimpleStringError : public llvm::ErrorInfo<SimpleStringError> {
  std::error_code EC;
  std::string Message;

public:
  SimpleStringError(std::error_code EC, std::string &&Message) : EC(EC), Message(std::move(Message)) {}
  void log(llvm::raw_ostream &OS) const override { OS << Message; }
  std::string message() const override { return Message; }
  std::error_code convertToErrorCode() const override { return EC; }
  static char ID;
};
namespace detail {
template <typename T> T &&wrap(T &&V) { return std::forward<T>(V); }
inline decltype(fmt_consume(llvm::Error::success())) wrap(llvm::Error &&V) { return fmt_consume(std::move(V)); }
inline llvm::Error error(std::error_code EC, std::string &&Msg) {
  return llvm::make_error<SimpleStringError>(EC, std::move(Msg));
}
} // namespace detail

template <typename... Ts> void elog(const char *Fmt, Ts &&...Vals) {
  llvm::errs() << llvm::formatv(Fmt, clangd::detail::wrap(std::forward<Ts>(Vals))...) << "\n";
}

// error() constructs an llvm::Error object, using formatv()-style arguments.
// It is not automatically logged! (This function is a little out of place).
// The error simply embeds the message string.
template <typename... Ts> llvm::Error error(std::error_code EC, const char *Fmt, Ts &&...Vals) {
  // We must render the formatv_object eagerly, while references are valid.
  return detail::error(EC, llvm::formatv(Fmt, detail::wrap(std::forward<Ts>(Vals))...).str());
}
// Overload with no error_code conversion, the error will be inconvertible.
template <typename... Ts> llvm::Error error(const char *Fmt, Ts &&...Vals) {
  return detail::error(llvm::inconvertibleErrorCode(),
                       llvm::formatv(Fmt, detail::wrap(std::forward<Ts>(Vals))...).str());
}
// Overload to avoid formatv complexity for simple strings.
inline llvm::Error error(std::error_code EC, std::string Msg) { return detail::error(EC, std::move(Msg)); }
// Overload for simple strings with no error_code conversion.
inline llvm::Error error(std::string Msg) { return detail::error(llvm::inconvertibleErrorCode(), std::move(Msg)); }
} // namespace clang::clangd
