//==============================================================================
// FILE:
//    HelloWorld.cpp
//
// DESCRIPTION:
//    Counts the number of C++ record declarations in the input translation
//    unit. The results are printed on a file-by-file basis (i.e. for each
//    included header file separately).
//
//    Internally, this implementation leverages llvm::StringMap to map file
//    names to the corresponding #count of declarations.
//
// USAGE:
//   clang -cc1 -load <BUILD_DIR>/lib/libHelloWorld.dylib '\'
//    -plugin hello-world test/HelloWorld-basic.cpp
//
// License: The Unlicense
//==============================================================================
#include "library.h"

#include <clang/AST/Decl.h>
#include <llvm/ADT/StringMap.h>
#include <llvm/ADT/StringRef.h>
#include <llvm/Support/Casting.h>
#include <sys/stat.h>

#include <fstream>
#include <unordered_map>

#include "TranslationUnitWrapper.h"
#include "proto/TopsAstProto.pb.h" // 包含生成的 Protobuf 头文件
#include "clang/AST/ASTConsumer.h"
#include "clang/AST/RecursiveASTVisitor.h"
#include "clang/Basic/FileManager.h"
#include "clang/Basic/SourceLocation.h"
#include "clang/Frontend/CompilerInstance.h"
#include "clang/Frontend/FrontendPluginRegistry.h"
#include "clang/Lex/HeaderSearch.h"
#include "clang/Lex/Preprocessor.h"
#include "llvm/Support/raw_ostream.h"

#include "Diagnostics.h"
#include "URI.h"
using namespace clang;

static std::string g_output_file = "output.idx"; // 默认输出文件名

std::string GetCommandLineArgs() {
  std::ifstream cmdline("/proc/self/cmdline");
  std::string content((std::istreambuf_iterator<char>(cmdline)), std::istreambuf_iterator<char>());

  std::replace(content.begin(), content.end(), '\0', ' ');
  return content;
}

static TranslationUnitWrapper g_tu;                  // 全局变量改为 Google 风格
static llvm::StringMap<uint32_t> g_string_table_map; // 使用 LLVM 的 StringMap 加速字符串表查找

//-----------------------------------------------------------------------------
// RecursiveASTVisitor
//-----------------------------------------------------------------------------
class TopsAstVisitor : public RecursiveASTVisitor<TopsAstVisitor> {
public:
  explicit TopsAstVisitor(ASTContext *context) : context_(context) {}
  bool VisitCXXRecordDecl(CXXRecordDecl *decl);
  bool VisitFunctionDecl(FunctionDecl *decl);
  bool VisitVarDecl(VarDecl *decl);
  bool VisitDeclRefExpr(DeclRefExpr *expr);

  void CollectTranslationUnitInfo(CompilerInstance &ci);
  void SerializeToProtobuf(const std::string &output_file);
  void CollectDiagnostics(ASTContext *ctx, DiagnosticsEngine &diag_engine);

  void SetContext(ASTContext *ctx) {
    context_ = ctx; // 设置 AST 上下文
  }

private:
  TopsAstProto::Function *GetFunctionProto(FunctionDecl *decl) {
    auto it = func_idx_map_.find(decl);
    if (it != func_idx_map_.end()) {
      return g_tu.getFunction(it->second);
    }
    return nullptr;
  }

  template <typename T> bool SetLocation(TopsAstProto::Location *loc, T *decl) {
    FullSourceLoc full_location = context_->getFullLoc(decl->getLocation());
    if (!full_location.isValid())
      return false;

    if (full_location.isMacroID())
      full_location = full_location.getExpansionLoc();

    loc->mutable_file_name()->set_index(g_tu.AddStringToTable(full_location.getFileEntry()->getName().str()));
    loc->set_line(full_location.getSpellingLineNumber());
    loc->set_column(full_location.getSpellingColumnNumber());
    auto len = Lexer::MeasureTokenLength(decl->getLocation(), context_->getSourceManager(), context_->getLangOpts());
    loc->set_length(len);
  }
  template <typename T> bool isValidLocation(T *decl) {
    FullSourceLoc full_location = context_->getFullLoc(decl->getLocation());
    if (!full_location.isValid())
      return false;
    return true;
  }

private:
  ASTContext *context_;
  std::map<FunctionDecl *, uint32_t> func_idx_map_;
  std::map<VarDecl *, uint32_t> var_idx_map_;
};

void TopsAstVisitor::CollectTranslationUnitInfo(CompilerInstance &ci) {
  // 设置文件路径
  g_tu.setSrcFile((ci.getFrontendOpts().Inputs[0].getFile().str()));
  llvm::SmallString<128> abs_path = ci.getFrontendOpts().Inputs[0].getFile();
  if (llvm::sys::fs::make_absolute(abs_path)) {
    llvm::errs() << "Failed to get absolute path for source file: " << ci.getFrontendOpts().Inputs[0].getFile() << "\n";
  } else {
    g_tu.get().set_file_path(abs_path.c_str());
  }
  // 设置编译参数
  g_tu.setCompileArgs(GetCommandLineArgs());
}

bool TopsAstVisitor::VisitCXXRecordDecl(CXXRecordDecl *Declaration) { return true; }

bool FillFuncInfo(TopsAstProto::Function *proto_func, FunctionDecl *decl, ASTContext *ctx) {
  FullSourceLoc full_location = ctx->getFullLoc(decl->getLocation());
  if (!full_location.isValid())
    return true;

  if (full_location.isMacroID())
    full_location = full_location.getExpansionLoc();

  proto_func->set_name((decl->getNameAsString()));
  proto_func->set_return_type((decl->getReturnType().getAsString()));

  auto *func_loc = proto_func->mutable_location();
  func_loc->mutable_file_name()->set_index(g_tu.AddStringToTable(full_location.getFileEntry()->getName().str()));
  func_loc->set_line(full_location.getSpellingLineNumber());
  func_loc->set_column(full_location.getSpellingColumnNumber());
  auto len = Lexer::MeasureTokenLength(decl->getLocation(), ctx->getSourceManager(), ctx->getLangOpts());
  func_loc->set_length(len);

  return true;
}
bool TopsAstVisitor::VisitFunctionDecl(FunctionDecl *decl) {
  if (func_idx_map_.count(decl))
    return true;

  if (!isValidLocation(decl))
    return true;

  TopsAstProto::Function *proto_func = nullptr;
  uint32_t func_idx = 0;
  std::tie(func_idx, proto_func) = g_tu.addFunction();

  if (decl->isThisDeclarationADefinition()) {
    proto_func->set_is_definition(true);
  }

  proto_func->set_name((decl->getNameAsString()));
  proto_func->set_return_type((decl->getReturnType().getAsString()));

  SetLocation(proto_func->mutable_location(), decl);

  func_idx_map_[decl] = func_idx;

  return true;
}

bool TopsAstVisitor::VisitVarDecl(VarDecl *decl) {
  if (var_idx_map_.count(decl))
    return true;

  if (!isValidLocation(decl))
    return true;

  TopsAstProto::Variable *proto_var = nullptr;
  uint32_t var_idx = 0;
  if (decl->isLocalVarDeclOrParm()) {
    auto func = dyn_cast<FunctionDecl>(decl->getDeclContext());
    auto *func_proto = GetFunctionProto(func);
    if (!func || !func_proto)
      return true;
    if (decl->isLocalVarDecl()) {
      std::tie(var_idx, proto_var) = g_tu.addLocalVar();
      func_proto->add_local_vars(var_idx);
    } else {
      std::tie(var_idx, proto_var) = g_tu.addParam();
      func_proto->add_parameters(var_idx);
    }

  } else {
    std::tie(var_idx, proto_var) = g_tu.addGlobalVar();
  }
  auto name = decl->getNameAsString();
  proto_var->set_name(name);
  auto type = decl->getType().getAsString();
  proto_var->set_type(type);

  auto *proto_loc = proto_var->mutable_location();
  SetLocation(proto_loc, decl);

  var_idx_map_[decl] = var_idx;

  return true;
}

bool TopsAstVisitor::VisitDeclRefExpr(DeclRefExpr *expr) {
  ValueDecl *ref_decl = expr->getDecl();

  // 仅保留对变量或函数参数的引用
  if (!isa<VarDecl>(ref_decl) && !isa<ParmVarDecl>(ref_decl) && !isa<FunctionDecl>(ref_decl))
    return true;

  if (!isValidLocation(expr))
    return true;

  auto *proto_ref = g_tu.addDeclRef();
  auto name = ref_decl->getNameAsString();
  proto_ref->set_referenced_name(name);
  // if (name == "__gcu_abs_cc") {
  //   expr->getLocation().dump(context_->getSourceManager());
  //   ref_decl->dump();
  // }

  if (auto *decl = dyn_cast<ParmVarDecl>(ref_decl)) {
    proto_ref->set_ref_type(TopsAstProto::DeclRef_RefType_PARAMETER);
    auto it = var_idx_map_.find(decl);
    if (it != var_idx_map_.end()) {
      proto_ref->set_variable(it->second);
    }
  } else if (auto *decl = dyn_cast<VarDecl>(ref_decl)) {
    proto_ref->set_ref_type(TopsAstProto::DeclRef_RefType_VARIABLE);
    auto it = var_idx_map_.find(decl);
    if (it != var_idx_map_.end()) {
      proto_ref->set_variable(it->second);
    }
  } else if (auto *decl = dyn_cast<FunctionDecl>(ref_decl)) {
    proto_ref->set_ref_type(TopsAstProto::DeclRef_RefType_FUNCTION);
    auto it = func_idx_map_.find(decl);
    if (it != func_idx_map_.end()) {
      proto_ref->set_function(it->second);
    }
  }

  auto *proto_loc = proto_ref->mutable_location();
  SetLocation(proto_loc, expr);

  return true;
}
TopsAstProto::DiagnosticSeverity getSeverity(DiagnosticsEngine::Level L) {
  static std::map<DiagnosticsEngine::Level, TopsAstProto::DiagnosticSeverity> severity_map = {
      {DiagnosticsEngine::Remark, TopsAstProto::DiagnosticSeverity::Hint},
      {DiagnosticsEngine::Note, TopsAstProto::DiagnosticSeverity::Information},
      {DiagnosticsEngine::Warning, TopsAstProto::DiagnosticSeverity::Warning},
      {DiagnosticsEngine::Error, TopsAstProto::DiagnosticSeverity::Error},
      {DiagnosticsEngine::Fatal, TopsAstProto::DiagnosticSeverity::Error},
      {DiagnosticsEngine::Ignored, TopsAstProto::DiagnosticSeverity::Ignore}};
  return severity_map[L];
}
TopsAstProto::Position getPosition(clangd::Position pos) {
  TopsAstProto::Position P;
  P.set_line(pos.line);
  P.set_character(pos.character);
  return P;
}
TopsAstProto::Range getRange(clangd::Range range) {
  TopsAstProto::Range R;
  *R.mutable_start() = getPosition(range.start);
  *R.mutable_end() = getPosition(range.end);
  return R;
}
// Diagnostic 信息收集实现
void TopsAstVisitor::CollectDiagnostics(ASTContext *ctx, DiagnosticsEngine &diag_engine) {
  auto Diags = (clangd::StoreDiags *)(diag_engine.getClient());
  auto File = clangd::URIForFile::canonicalize(g_tu.get().file_path(), "");
  for (auto &D : Diags->take()) {
    auto *diag = g_tu.get().add_diagnostics();
    diag->set_severity(getSeverity(D.Severity));
    if (D.InsideMainFile) {
      *diag->mutable_range() = getRange(D.Range);
    } else {
      auto It = llvm::find_if(D.Notes, [](const clangd::Note &N) { return N.InsideMainFile; });
      assert(It != D.Notes.end() && "neither the main diagnostic nor notes are inside main file");
      *diag->mutable_range() = getRange(It->Range);
    }
    diag->set_source("tops-lsp");
    diag->set_message(clangd::mainMessage(D));
    llvm::dbgs() << "Diagnostic: " << diag->message() << "\n";

    for (auto &Note : D.Notes) {
      if (!Note.AbsFile) {
        continue;
      }
      auto *RelInfo = diag->add_relatedinformation();
      *RelInfo->mutable_location()->mutable_range() = getRange(Note.Range);
      RelInfo->mutable_location()->set_uri(clangd::URIForFile::canonicalize(*Note.AbsFile, File.file()).uri());
      RelInfo->set_message(noteMessage(D, Note));
    }
  }
}

void TopsAstVisitor::SerializeToProtobuf(const std::string &output_file) {
  // llvm::dbgs() << "Writing to " << output_file << "\n";
  if (!g_tu.SerializeToProtobuf(output_file)) {
    llvm::errs() << "Failed to serialize data to " << output_file << "\n";
  }
}

//-----------------------------------------------------------------------------
// ASTConsumer
//-----------------------------------------------------------------------------
class TopsASTConsumer : public clang::ASTConsumer {
public:
  explicit TopsASTConsumer(ASTContext *ctx, CompilerInstance &ci) : visitor_(ctx), ci_(ci) {}

  void HandleTranslationUnit(clang::ASTContext &ctx) override {
    visitor_.CollectTranslationUnitInfo(ci_); // 收集文件路径和头文件
    visitor_.TraverseDecl(ctx.getTranslationUnitDecl());
    // 收集diagnostic信息
    visitor_.CollectDiagnostics(&ctx, ci_.getDiagnostics());
    visitor_.SerializeToProtobuf(g_output_file); // 序列化到 Protobuf 文件
  }

private:
  TopsAstVisitor visitor_;
  CompilerInstance &ci_;
};

//-----------------------------------------------------------------------------
// FrontendAction for HelloWorld
//-----------------------------------------------------------------------------
class FindSymbolAction : public clang::PluginASTAction {
public:
  std::unique_ptr<clang::ASTConsumer> CreateASTConsumer(clang::CompilerInstance &compiler,
                                                        llvm::StringRef in_file) override {
    compiler.getPreprocessor().addPPCallbacks(std::make_unique<MyPPCallbacks>());
    auto *Diags = new clangd::StoreDiags();
    Diags->BeginSourceFile(compiler.getLangOpts(), &compiler.getPreprocessor());
    compiler.getDiagnostics().setClient(Diags, true);
    return std::make_unique<TopsASTConsumer>(&compiler.getASTContext(), compiler);
  }

  bool ParseArgs(const CompilerInstance &ci, const std::vector<std::string> &args) override {
    if (args.empty()) {
      llvm::errs() << "Tops LSP plugin output arguments provided.\n";
      return false;
    }
    g_output_file = args.front();
    return true;
  }
};

void MyPPCallbacks::InclusionDirective(SourceLocation hash_loc, const Token &include_tok, StringRef file_name,
                                       bool is_angled, CharSourceRange filename_range, const FileEntry *file,
                                       StringRef search_path, StringRef relative_path, const Module *imported,
                                       SrcMgr::CharacteristicKind file_type) {
  struct stat file_stat{};
  const auto header = file->getName();
  auto str_idx = g_tu.AddStringToTable(header.str());
  if (processed_.count(str_idx))
    return;
  if (stat(header.str().c_str(), &file_stat) == 0) {
    g_tu.addIncludedHeader(header.str().c_str());
    processed_.insert(str_idx);
  }
}

//-----------------------------------------------------------------------------
// Registration
//-----------------------------------------------------------------------------
static FrontendPluginRegistry::Add<FindSymbolAction> x(
    /*Name=*/"tops-lsp", /*Description=*/"The tops lsp plugin");

char clangd::SimpleStringError::ID;
