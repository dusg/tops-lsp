#include "Diagnostics.h"
#include "library.h"
#include "clang/Basic/AllDiagnostics.h" // IWYU pragma: keep
#include "clang/Basic/Diagnostic.h"
#include "clang/Basic/DiagnosticIDs.h"
#include "clang/Basic/LLVM.h"
#include "clang/Basic/SourceLocation.h"
#include "clang/Basic/SourceManager.h"
#include "clang/Basic/TokenKinds.h"
#include "clang/Lex/Lexer.h"
#include "clang/Lex/Token.h"
#include "llvm/ADT/ArrayRef.h"
#include "llvm/ADT/DenseSet.h"
#include "llvm/ADT/STLExtras.h"
#include "llvm/ADT/ScopeExit.h"
#include "llvm/ADT/SmallString.h"
#include "llvm/ADT/SmallVector.h"
#include "llvm/ADT/StringExtras.h"
#include "llvm/ADT/StringRef.h"
#include "llvm/ADT/StringSet.h"
#include "llvm/ADT/Twine.h"
#include "llvm/Support/ErrorHandling.h"
#include "llvm/Support/FormatVariadic.h"
#include "llvm/Support/Path.h"
#include "llvm/Support/SourceMgr.h"
#include "llvm/Support/raw_ostream.h"

using namespace clangd;
static void fillNonLocationData(DiagnosticsEngine::Level DiagLevel, const clang::Diagnostic &Info,
                                clangd::DiagBase &D) {
  llvm::SmallString<64> Message;
  Info.FormatDiagnostic(Message);

  D.Message = std::string(Message.str());
  D.Severity = DiagLevel;
  D.Category = DiagnosticIDs::getCategoryNameFromID(DiagnosticIDs::getCategoryNumberForDiag(Info.getID())).str();
}
Key<OffsetEncoding> kCurrentOffsetEncoding;
static OffsetEncoding lspEncoding() { return OffsetEncoding::UTF8; }
// Like most strings in clangd, the input is UTF-8 encoded.
size_t lspLength(llvm::StringRef Code) { return Code.size(); }

Position sourceLocToPosition(const SourceManager &SM, SourceLocation Loc) {
  // We use the SourceManager's line tables, but its column number is in bytes.
  FileID FID;
  unsigned Offset;
  std::tie(FID, Offset) = SM.getDecomposedSpellingLoc(Loc);
  Position P;
  P.line = static_cast<int>(SM.getLineNumber(FID, Offset)) - 1;
  bool Invalid = false;
  llvm::StringRef Code = SM.getBufferData(FID, &Invalid);
  if (!Invalid) {
    auto ColumnInBytes = SM.getColumnNumber(FID, Offset) - 1;
    auto LineSoFar = Code.substr(Offset - ColumnInBytes, ColumnInBytes);
    P.character = lspLength(LineSoFar);
  }
  return P;
}
Range halfOpenToRange(const SourceManager &SM, CharSourceRange R) {
  // Clang is 1-based, LSP uses 0-based indexes.
  Position Begin = sourceLocToPosition(SM, R.getBegin());
  Position End = sourceLocToPosition(SM, R.getEnd());

  return {Begin, End};
}
bool isInsideMainFile(SourceLocation Loc, const SourceManager &SM) {
  if (!Loc.isValid())
    return false;
  FileID FID = SM.getFileID(SM.getExpansionLoc(Loc));
  return FID == SM.getMainFileID() || FID == SM.getPreambleFileID();
}
// Checks whether a location is within a half-open range.
// Note that clang also uses closed source ranges, which this can't handle!
bool locationInRange(SourceLocation L, CharSourceRange R, const SourceManager &M) {
  assert(R.isCharRange());
  if (!R.isValid() || M.getFileID(R.getBegin()) != M.getFileID(R.getEnd()) ||
      M.getFileID(R.getBegin()) != M.getFileID(L))
    return false;
  return L != R.getEnd() && M.isPointWithin(L, R.getBegin(), R.getEnd());
}

// Clang diags have a location (shown as ^) and 0 or more ranges (~~~~).
// LSP needs a single range.
std::optional<Range> diagnosticRange(const clang::Diagnostic &D, const LangOptions &L) {
  auto &M = D.getSourceManager();
  auto Loc = M.getFileLoc(D.getLocation());
  for (const auto &CR : D.getRanges()) {
    auto R = Lexer::makeFileCharRange(CR, M, L);
    if (locationInRange(Loc, R, M))
      return halfOpenToRange(M, R);
  }
  // The range may be given as a fixit hint instead.
  for (const auto &F : D.getFixItHints()) {
    auto R = Lexer::makeFileCharRange(F.RemoveRange, M, L);
    if (locationInRange(Loc, R, M))
      return halfOpenToRange(M, R);
  }
  // Source locations from stale preambles might become OOB.
  // FIXME: These diagnostics might point to wrong locations even when they're
  // not OOB.
  auto [FID, Offset] = M.getDecomposedLoc(Loc);
  if (Offset > M.getBufferData(FID).size())
    return std::nullopt;
  // If the token at the location is not a comment, we use the token.
  // If we can't get the token at the location, fall back to using the location
  auto R = CharSourceRange::getCharRange(Loc);
  Token Tok;
  if (!Lexer::getRawToken(Loc, Tok, M, L, true) && Tok.isNot(tok::comment))
    R = CharSourceRange::getTokenRange(Tok.getLocation(), Tok.getEndLoc());
  return halfOpenToRange(M, R);
}

std::optional<std::string> getCanonicalPath(const FileEntryRef F, FileManager &FileMgr) {
  llvm::SmallString<128> FilePath = F.getName();
  if (!llvm::sys::path::is_absolute(FilePath)) {
    if (auto EC = FileMgr.getVirtualFileSystem().makeAbsolute(FilePath)) {
      elog("Could not turn relative path '{0}' to absolute: {1}", FilePath, EC.message());
      return std::nullopt;
    }
  }

  // Handle the symbolic link path case where the current working directory
  // (getCurrentWorkingDirectory) is a symlink. We always want to the real
  // file path (instead of the symlink path) for the  C++ symbols.
  //
  // Consider the following example:
  //
  //   src dir: /project/src/foo.h
  //   current working directory (symlink): /tmp/build -> /project/src/
  //
  //  The file path of Symbol is "/project/src/foo.h" instead of
  //  "/tmp/build/foo.h"
  if (auto Dir = FileMgr.getOptionalDirectoryRef(llvm::sys::path::parent_path(FilePath))) {
    llvm::SmallString<128> RealPath;
    llvm::StringRef DirName = FileMgr.getCanonicalName(&Dir.getValue().getDirEntry());
    llvm::sys::path::append(RealPath, DirName, llvm::sys::path::filename(FilePath));
    return RealPath.str().str();
  }

  return FilePath.str().str();
}

// Try to find a location in the main-file to report the diagnostic D.
// Returns a description like "in included file", or nullptr on failure.
const char *getMainFileRange(const Diag &D, const SourceManager &SM, SourceLocation DiagLoc, Range &R) {
  // Look for a note in the main file indicating template instantiation.
  for (const auto &N : D.Notes) {
    if (N.InsideMainFile) {
      switch (N.ID) {
      case diag::note_template_class_instantiation_was_here:
      case diag::note_template_class_explicit_specialization_was_here:
      case diag::note_template_class_instantiation_here:
      case diag::note_template_member_class_here:
      case diag::note_template_member_function_here:
      case diag::note_function_template_spec_here:
      case diag::note_template_static_data_member_def_here:
      case diag::note_template_variable_def_here:
      case diag::note_template_enum_def_here:
      case diag::note_template_nsdmi_here:
      case diag::note_template_type_alias_instantiation_here:
      case diag::note_template_exception_spec_instantiation_here:
      case diag::note_template_requirement_instantiation_here:
      case diag::note_evaluating_exception_spec_here:
      case diag::note_default_arg_instantiation_here:
      case diag::note_default_function_arg_instantiation_here:
      case diag::note_explicit_template_arg_substitution_here:
      case diag::note_function_template_deduction_instantiation_here:
      case diag::note_deduced_template_arg_substitution_here:
      case diag::note_prior_template_arg_substitution:
      case diag::note_template_default_arg_checking:
      case diag::note_concept_specialization_here:
      case diag::note_nested_requirement_here:
      case diag::note_checking_constraints_for_template_id_here:
      case diag::note_checking_constraints_for_var_spec_id_here:
      case diag::note_checking_constraints_for_class_spec_id_here:
      case diag::note_checking_constraints_for_function_here:
      case diag::note_constraint_substitution_here:
      case diag::note_constraint_normalization_here:
      case diag::note_parameter_mapping_substitution_here:
        R = N.Range;
        return "in template";
      default:
        break;
      }
    }
  }
  // Look for where the file with the error was #included.
  auto GetIncludeLoc = [&SM](SourceLocation SLoc) { return SM.getIncludeLoc(SM.getFileID(SLoc)); };
  for (auto IncludeLocation = GetIncludeLoc(SM.getExpansionLoc(DiagLoc)); IncludeLocation.isValid();
       IncludeLocation = GetIncludeLoc(IncludeLocation)) {
    if (isInsideMainFile(IncludeLocation, SM)) {
      R.start = sourceLocToPosition(SM, IncludeLocation);
      R.end = sourceLocToPosition(SM, Lexer::getLocForEndOfToken(IncludeLocation, 0, SM, LangOptions()));
      return "in included file";
    }
  }
  return nullptr;
}
bool mentionsMainFile(const Diag &D) {
  if (D.InsideMainFile)
    return true;
  // Fixes are always in the main file.
  if (!D.Fixes.empty())
    return true;
  for (auto &N : D.Notes) {
    if (N.InsideMainFile)
      return true;
  }
  return false;
}

bool tryMoveToMainFile(Diag &D, FullSourceLoc DiagLoc) {
  const SourceManager &SM = DiagLoc.getManager();
  DiagLoc = DiagLoc.getExpansionLoc();
  Range R;
  const char *Prefix = getMainFileRange(D, SM, DiagLoc, R);
  if (!Prefix)
    return false;

  // Add a note that will point to real diagnostic.
  auto FE = *SM.getFileEntryRefForID(SM.getFileID(DiagLoc));
  D.Notes.emplace(D.Notes.begin());
  Note &N = D.Notes.front();
  N.AbsFile = std::string(FE.getFileEntry().tryGetRealPathName());
  N.File = std::string(FE.getName());
  N.Message = "error occurred here";
  N.Range = D.Range;

  // Update diag to point at include inside main file.
  D.File = SM.getFileEntryRefForID(SM.getMainFileID())->getName().str();
  D.Range = std::move(R);
  D.InsideMainFile = true;
  // Update message to mention original file.
  D.Message = llvm::formatv("{0}: {1}", Prefix, D.Message);
  return true;
}

bool isNote(DiagnosticsEngine::Level L) { return L == DiagnosticsEngine::Note || L == DiagnosticsEngine::Remark; }
bool isExcluded(unsigned DiagID) {
  // clang will always fail parsing MS ASM, we don't link in desc + asm parser.
  if (DiagID == clang::diag::err_msasm_unable_to_create_target || DiagID == clang::diag::err_msasm_unsupported_arch ||
      DiagID == diag::err_drv_unsupported_option_argument)
    return true;
  return false;
}

TextEdit toTextEdit(const FixItHint &FixIt, const SourceManager &M, const LangOptions &L) {
  TextEdit Result;
  Result.range = halfOpenToRange(M, Lexer::makeFileCharRange(FixIt.RemoveRange, M, L));
  Result.newText = FixIt.CodeToInsert;
  return Result;
}

/// Sanitizes a piece for presenting it in a synthesized fix message. Ensures
/// the result is not too large and does not contain newlines.
static void writeCodeToFixMessage(llvm::raw_ostream &OS, llvm::StringRef Code) {
  constexpr unsigned MaxLen = 50;
  if (Code == "\n") {
    OS << "\\n";
    return;
  }
  // Only show the first line if there are many.
  llvm::StringRef R = Code.split('\n').first;
  // Shorten the message if it's too long.
  R = R.take_front(MaxLen);

  OS << R;
  if (R.size() != Code.size())
    OS << "…";
}
void clangd::StoreDiags::HandleDiagnostic(DiagnosticsEngine::Level DiagLevel, const clang::Diagnostic &Info) {
  // If the diagnostic was generated for a different SourceManager, skip it.
  // This happens when a module is imported and needs to be implicitly built.
  // The compilation of that module will use the same StoreDiags, but different
  // SourceManager.
  if (OrigSrcMgr && Info.hasSourceManager() && OrigSrcMgr != &Info.getSourceManager()) {
    return;
  }

  DiagnosticConsumer::HandleDiagnostic(DiagLevel, Info);
  bool OriginallyError = Info.getDiags()->getDiagnosticIDs()->isDefaultMappingAsError(Info.getID());

  if (Info.getLocation().isInvalid()) {
    // Handle diagnostics coming from command-line arguments. The source manager
    // is *not* available at this point, so we cannot use it.
    if (!OriginallyError) {
      return; // non-errors add too much noise, do not show them.
    }

    flushLastDiag();

    LastDiag = Diag();
    LastDiagLoc.reset();
    LastDiagOriginallyError = OriginallyError;
    LastDiag->ID = Info.getID();
    fillNonLocationData(DiagLevel, Info, *LastDiag);
    LastDiag->InsideMainFile = true;
    // Put it at the start of the main file, for a lack of a better place.
    LastDiag->Range.start = Position{0, 0};
    LastDiag->Range.end = Position{0, 0};
    return;
  }
  // llvm::SmallString<64> Message;
  // Info.FormatDiagnostic(Message);
  // llvm::errs() << Message << "\n";
  if (!LangOpts || !Info.hasSourceManager()) {
    return;
  }

  SourceManager &SM = Info.getSourceManager();

  auto FillDiagBase = [&](DiagBase &D) {
    fillNonLocationData(DiagLevel, Info, D);

    // SourceLocation PatchLoc =
    //     translatePreamblePatchLocation(Info.getLocation(), SM);
    D.InsideMainFile = isInsideMainFile(Info.getLocation(), SM);
    if (auto DRange = diagnosticRange(Info, *LangOpts))
      D.Range = *DRange;
    else
      D.Severity = DiagnosticsEngine::Ignored;
    auto FID = SM.getFileID(Info.getLocation());
    if (const auto FE = SM.getFileEntryRefForID(FID)) {
      D.File = FE->getName().str();
      D.AbsFile = getCanonicalPath(*FE, SM.getFileManager());
    }
    D.ID = Info.getID();
    return D;
  };

  auto AddFix = [&](bool SyntheticMessage) -> bool {
    assert(!Info.getFixItHints().empty() && "diagnostic does not have attached fix-its");
    // No point in generating fixes, if the diagnostic is for a different file.
    if (!LastDiag->InsideMainFile)
      return false;
    // Copy as we may modify the ranges.
    auto FixIts = Info.getFixItHints().vec();
    llvm::SmallVector<TextEdit, 1> Edits;
    for (auto &FixIt : FixIts) {
      // Allow fixits within a single macro-arg expansion to be applied.
      // This can be incorrect if the argument is expanded multiple times in
      // different contexts. Hopefully this is rare!
      if (FixIt.RemoveRange.getBegin().isMacroID() && FixIt.RemoveRange.getEnd().isMacroID() &&
          SM.getFileID(FixIt.RemoveRange.getBegin()) == SM.getFileID(FixIt.RemoveRange.getEnd())) {
        FixIt.RemoveRange = CharSourceRange({SM.getTopMacroCallerLoc(FixIt.RemoveRange.getBegin()),
                                             SM.getTopMacroCallerLoc(FixIt.RemoveRange.getEnd())},
                                            FixIt.RemoveRange.isTokenRange());
      }
      // Otherwise, follow clang's behavior: no fixits in macros.
      if (FixIt.RemoveRange.getBegin().isMacroID() || FixIt.RemoveRange.getEnd().isMacroID())
        return false;
      if (!isInsideMainFile(FixIt.RemoveRange.getBegin(), SM))
        return false;
      Edits.push_back(toTextEdit(FixIt, SM, *LangOpts));
    }

    llvm::SmallString<64> Message;
    // If requested and possible, create a message like "change 'foo' to 'bar'".
    if (SyntheticMessage && FixIts.size() == 1) {
      const auto &FixIt = FixIts.front();
      bool Invalid = false;
      llvm::StringRef Remove = Lexer::getSourceText(FixIt.RemoveRange, SM, *LangOpts, &Invalid);
      llvm::StringRef Insert = FixIt.CodeToInsert;
      if (!Invalid) {
        llvm::raw_svector_ostream M(Message);
        if (!Remove.empty() && !Insert.empty()) {
          M << "change '";
          writeCodeToFixMessage(M, Remove);
          M << "' to '";
          writeCodeToFixMessage(M, Insert);
          M << "'";
        } else if (!Remove.empty()) {
          M << "remove '";
          writeCodeToFixMessage(M, Remove);
          M << "'";
        } else if (!Insert.empty()) {
          M << "insert '";
          writeCodeToFixMessage(M, Insert);
          M << "'";
        }
        // Don't allow source code to inject newlines into diagnostics.
        std::replace(Message.begin(), Message.end(), '\n', ' ');
      }
    }
    if (Message.empty()) // either !SyntheticMessage, or we failed to make one.
      Info.FormatDiagnostic(Message);
    LastDiag->Fixes.push_back(Fix{std::string(Message.str()), std::move(Edits), {}});
    return true;
  };

  if (!isNote(DiagLevel)) {
    // Handle the new main diagnostic.
    flushLastDiag();

    LastDiag = Diag();

    FillDiagBase(*LastDiag);
    if (isExcluded(LastDiag->ID))
      LastDiag->Severity = DiagnosticsEngine::Ignored;
    // Don't bother filling in the rest if diag is going to be dropped.
    if (LastDiag->Severity == DiagnosticsEngine::Ignored)
      return;

    LastDiagLoc.emplace(Info.getLocation(), Info.getSourceManager());
    LastDiagOriginallyError = OriginallyError;
    if (!Info.getFixItHints().empty())
      AddFix(true /* try to invent a message instead of repeating the diag */);
    return;
  }
  assert(isNote(DiagLevel));

  // Handle a note to an existing diagnostic.
  if (!LastDiag) {
    assert(false && "Adding a note without main diagnostic");
  }

  // If a diagnostic was suppressed due to the suppression filter,
  // also suppress notes associated with it.
  if (LastDiag->Severity == DiagnosticsEngine::Ignored)
    return;

  if (!Info.getFixItHints().empty()) {
    // A clang note with fix-it is not a separate diagnostic in clangd. We
    // attach it as a Fix to the main diagnostic instead.
    AddFix(false /* use the note as the message */);
    return;
  }
  // A clang note without fix-its corresponds to clangd::Note.
  Note N;
  FillDiagBase(N);

  LastDiag->Notes.push_back(std::move(N));
}

void StoreDiags::BeginSourceFile(const LangOptions &Opts, const Preprocessor *PP) {
  LangOpts = Opts;
  if (PP) {
    OrigSrcMgr = &PP->getSourceManager();
  }
}

void StoreDiags::EndSourceFile() {
  flushLastDiag();
  LangOpts = std::nullopt;
  OrigSrcMgr = nullptr;
}

void StoreDiags::flushLastDiag() {
  if (!LastDiag)
    return;
  auto Finish = llvm::make_scope_exit([&, NDiags(Output.size())] {
    if (Output.size() == NDiags) // No new diag emitted.
      elog("Dropped diagnostic: {0}: {1}", LastDiag->File, LastDiag->Message);
    LastDiag.reset();
  });

  if (LastDiag->Severity == DiagnosticsEngine::Ignored)
    return;
  // Move errors that occur from headers into main file.
  if (!LastDiag->InsideMainFile && LastDiagLoc && LastDiagOriginallyError) {
    if (tryMoveToMainFile(*LastDiag, *LastDiagLoc)) {
      // Suppress multiple errors from the same inclusion.
      if (!IncludedErrorLocations.insert({LastDiag->Range.start.line, LastDiag->Range.start.character}).second)
        return;
    }
  }
  if (!mentionsMainFile(*LastDiag))
    return;
  Output.push_back(std::move(*LastDiag));
}

llvm::StringRef diagLeveltoString(DiagnosticsEngine::Level Lvl) {
  switch (Lvl) {
  case DiagnosticsEngine::Ignored:
    return "ignored";
  case DiagnosticsEngine::Note:
    return "note";
  case DiagnosticsEngine::Remark:
    return "remark";
  case DiagnosticsEngine::Warning:
    return "warning";
  case DiagnosticsEngine::Error:
    return "error";
  case DiagnosticsEngine::Fatal:
    return "fatal error";
  }
  llvm_unreachable("unhandled DiagnosticsEngine::Level");
}

/// Prints a single diagnostic in a clang-like manner, the output includes
/// location, severity and error message. An example of the output message is:
///
///     main.cpp:12:23: error: undeclared identifier
///
/// For main file we only print the basename and for all other files we print
/// the filename on a separate line to provide a slightly more readable output
/// in the editors:
///
///     dir1/dir2/dir3/../../dir4/header.h:12:23
///     error: undeclared identifier
void printDiag(llvm::raw_string_ostream &OS, const DiagBase &D) {
  if (D.InsideMainFile) {
    // Paths to main files are often taken from compile_command.json, where they
    // are typically absolute. To reduce noise we print only basename for them,
    // it should not be confusing and saves space.
    OS << llvm::sys::path::filename(D.File) << ":";
  } else {
    OS << D.File << ":";
  }
  // Note +1 to line and character. clangd::Range is zero-based, but when
  // printing for users we want one-based indexes.
  auto Pos = D.Range.start;
  OS << (Pos.line + 1) << ":" << (Pos.character + 1) << ":";
  // The non-main-file paths are often too long, putting them on a separate
  // line improves readability.
  if (D.InsideMainFile)
    OS << " ";
  else
    OS << "\n";
  OS << diagLeveltoString(D.Severity) << ": " << D.Message;
}

/// Capitalizes the first word in the diagnostic's message.
std::string capitalize(std::string Message) {
  if (!Message.empty())
    Message[0] = llvm::toUpper(Message[0]);
  return Message;
}

/// Returns a message sent to LSP for the main diagnostic in \p D.
/// This message may include notes, if they're not emitted in some other way.
/// Example output:
///
///     no matching function for call to 'foo'
///
///     main.cpp:3:5: note: candidate function not viable: requires 2 arguments
///
///     dir1/dir2/dir3/../../dir4/header.h:12:23
///     note: candidate function not viable: requires 3 arguments
std::string clangd::mainMessage(const Diag &D) {
  std::string Result;
  llvm::raw_string_ostream OS(Result);
  OS << D.Message;
  if (!D.Fixes.empty())
    OS << " (" << (D.Fixes.size() > 1 ? "fixes" : "fix") << " available)";
  OS.flush();
  return capitalize(std::move(Result));
}

/// Returns a message sent to LSP for the note of the main diagnostic.
std::string clangd::noteMessage(const Diag &Main, const DiagBase &Note) {
  std::string Result;
  llvm::raw_string_ostream OS(Result);
  OS << Note.Message;
  OS.flush();
  return capitalize(std::move(Result));
}
