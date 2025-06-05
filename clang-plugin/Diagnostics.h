#pragma once
#include <optional>
#include <string>
#include <tuple>

#include "Types.h"
#include "clang/Basic/Diagnostic.h"
#include "clang/Basic/LangOptions.h"
#include "clang/Basic/SourceLocation.h"
#include "llvm/ADT/ArrayRef.h"
#include "llvm/ADT/DenseSet.h"
#include "llvm/ADT/SmallVector.h"
#include "llvm/ADT/StringSet.h"
#include "llvm/Support/JSON.h"
#include "llvm/Support/SourceMgr.h"
using namespace clang;
namespace clang::clangd {

struct Fix {
  /// Message for the fix-it.
  std::string Message;
  /// TextEdits from clang's fix-its. Must be non-empty.
  llvm::SmallVector<TextEdit, 1> Edits;

  /// Annotations for the Edits.
  llvm::SmallVector<std::pair<ChangeAnnotationIdentifier, ChangeAnnotation>, 8> Annotations;
};

struct DiagBase {
  std::string Message;
  // Intended to be used only in error messages.
  // May be relative, absolute or even artificially constructed.
  std::string File;
  // Absolute path to containing file, if available.
  std::optional<std::string> AbsFile;

  clangd::Range Range;
  DiagnosticsEngine::Level Severity = DiagnosticsEngine::Note;
  std::string Category;
  // Since File is only descriptive, we store a separate flag to distinguish
  // diags from the main file.
  bool InsideMainFile = false;
  unsigned ID = 0;  // e.g. member of clang::diag, or clang-tidy assigned ID.
};
struct Note : DiagBase {};

struct Diag : DiagBase {
  std::string Name;  // if ID was recognized.
  // The source of this diagnostic.
  enum DiagSource {
    Unknown,
    Clang,
    ClangTidy,
    Clangd,
    ClangdConfig,
  } Source = Unknown;
  /// Elaborate on the problem, usually pointing to a related piece of code.
  std::vector<Note> Notes;
  /// *Alternative* fixes for this diagnostic, one should be chosen.
  std::vector<Fix> Fixes;
  llvm::SmallVector<DiagnosticTag, 1> Tags;
};

class StoreDiags : public DiagnosticConsumer {
 public:
  void BeginSourceFile(const LangOptions &Opts, const Preprocessor *PP) override;
  void EndSourceFile() override;
  void HandleDiagnostic(DiagnosticsEngine::Level DiagLevel, const clang::Diagnostic &Info) override;

  std::vector<Diag>& take() {
    flushLastDiag();
    return Output;
  }
 private:
  void flushLastDiag();

  std::vector<Diag> Output;
  std::optional<LangOptions> LangOpts;
  std::optional<Diag> LastDiag;
  std::optional<FullSourceLoc> LastDiagLoc;  // Valid only when LastDiag is set.
  bool LastDiagOriginallyError = false;      // Valid only when LastDiag is set.
  SourceManager *OrigSrcMgr = nullptr;

  llvm::DenseSet<std::pair<unsigned, unsigned>> IncludedErrorLocations;
};

std::string mainMessage(const Diag &D);
std::string noteMessage(const Diag &Main, const DiagBase &Note);
}  // namespace clangd
