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

#include <llvm/ADT/StringMap.h>
#include <llvm/ADT/StringRef.h>
#include <sys/stat.h>

#include <fstream>
#include <unordered_map>

#include "clang/AST/ASTConsumer.h"
#include "clang/AST/RecursiveASTVisitor.h"
#include "clang/Basic/FileManager.h"
#include "clang/Basic/SourceLocation.h"
#include "clang/Frontend/CompilerInstance.h"
#include "clang/Frontend/FrontendPluginRegistry.h"
#include "clang/Lex/HeaderSearch.h"
#include "clang/Lex/Preprocessor.h"
#include "llvm/Support/raw_ostream.h"
#include "proto/TopsAstProto.pb.h"  // 包含生成的 Protobuf 头文件

using namespace clang;

static std::string g_output_file = "output.idx";  // 默认输出文件名

std::string GetCommandLineArgs() {
  std::ifstream cmdline("/proc/self/cmdline");
  std::string content((std::istreambuf_iterator<char>(cmdline)),
                      std::istreambuf_iterator<char>());

  std::replace(content.begin(), content.end(), '\0', ' ');
  return content;
}

static TopsAstProto::TranslationUnit g_proto_tu;  // 全局变量改为 Google 风格
static llvm::StringMap<uint32_t>
    g_string_table_map;  // 使用 LLVM 的 StringMap 加速字符串表查找

// 添加字符串到字符串表并返回索引
static uint32_t AddStringToTable(const std::string &str) {
  auto it = g_string_table_map.find(str);
  if (it != g_string_table_map.end()) {
    return it->second;
  }

  auto *string_table = g_proto_tu.mutable_string_table();
  uint32_t index = string_table->entries_size();
  string_table->add_entries(str);
  g_string_table_map[str] = index;
  return index;
}

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
  bool VisitCallExpr(CallExpr *call_expr);

  void CollectTranslationUnitInfo(CompilerInstance &ci);
  void SerializeToProtobuf(const std::string &output_file);

  void SetContext(ASTContext *ctx) {
    context_ = ctx;  // 设置 AST 上下文
  }

 private:
  ASTContext *context_;
  std::unordered_map<FunctionDecl *, TopsAstProto::Function *> func_def_map_;
  std::map<FunctionDecl *, uint32_t> func_decl_map_;
};

void TopsAstVisitor::CollectTranslationUnitInfo(CompilerInstance &ci) {
  // 设置文件路径
  g_proto_tu.set_file_path((ci.getFrontendOpts().Inputs[0].getFile().str()));

  // 设置编译参数
  g_proto_tu.set_compile_args(GetCommandLineArgs());
}

bool TopsAstVisitor::VisitCXXRecordDecl(CXXRecordDecl *Declaration) {
  return true;
}

bool FillFuncInfo(TopsAstProto::Function *proto_func, FunctionDecl *decl,
                  ASTContext *ctx) {
  FullSourceLoc full_location = ctx->getFullLoc(decl->getLocation());
  if (!full_location.isValid()) return true;

  if (full_location.isMacroID())
    full_location = full_location.getExpansionLoc();

  proto_func->set_name((decl->getNameAsString()));
  proto_func->set_return_type((decl->getReturnType().getAsString()));

  auto *func_loc = proto_func->mutable_location();
  func_loc->mutable_file_name()->set_index(
      AddStringToTable(full_location.getFileEntry()->getName().str()));
  func_loc->set_line(full_location.getSpellingLineNumber());
  func_loc->set_column(full_location.getSpellingColumnNumber());
  auto len = Lexer::MeasureTokenLength(
      decl->getLocation(), ctx->getSourceManager(), ctx->getLangOpts());
  func_loc->set_length(len);

  for (ParmVarDecl *param : decl->parameters()) {
    auto *proto_param = proto_func->add_parameters();
    proto_param->set_name((param->getNameAsString()));
    proto_param->set_type((param->getType().getAsString()));

    FullSourceLoc param_location = ctx->getFullLoc(param->getLocation());
    auto *proto_param_loc = proto_param->mutable_location();
    proto_param_loc->mutable_file_name()->set_index(
        AddStringToTable(full_location.getFileEntry()->getName().str()));
    proto_param_loc->set_line(param_location.getSpellingLineNumber());
    proto_param_loc->set_column(param_location.getSpellingColumnNumber());
    auto tokenLen = Lexer::MeasureTokenLength(
        param->getLocation(), ctx->getSourceManager(), ctx->getLangOpts());
    proto_param_loc->set_length(tokenLen);
  }

  return true;
}
bool TopsAstVisitor::VisitFunctionDecl(FunctionDecl *decl) {
  FullSourceLoc full_location = context_->getFullLoc(decl->getLocation());
  if (!full_location.isValid()) return true;

  if (full_location.isMacroID())
    full_location = full_location.getExpansionLoc();

  TopsAstProto::Function *proto_func = nullptr;

  if (decl->isThisDeclarationADefinition()) {
    proto_func = g_proto_tu.add_func_defs();
  } else {
    if (this->func_decl_map_.count(decl)) return true;
    proto_func = g_proto_tu.add_func_decls();
  }

  FillFuncInfo(proto_func, decl, context_);

  if (!decl->isThisDeclarationADefinition()) {
    this->func_decl_map_[decl] = g_proto_tu.func_decls_size() - 1;
  }
  if (decl->isThisDeclarationADefinition())
    this->func_def_map_[decl] = proto_func;
  return true;
}

bool TopsAstVisitor::VisitVarDecl(VarDecl *decl) {
  if (dyn_cast<ParmVarDecl>(decl)) return true;
  auto loc = decl->getLocation();
  auto spell_loc = context_->getSourceManager().getSpellingLoc(loc);
  auto expan_loc = context_->getSourceManager().getExpansionLoc(loc);
  if (spell_loc != expan_loc) return true;
  FullSourceLoc full_location = context_->getFullLoc(loc);
  if (!full_location.isValid()) return true;

  // if (full_location.isMacroID())
  //   full_location = full_location.getExpansionLoc();

  TopsAstProto::Variable *proto_var = nullptr;
  if (decl->isLocalVarDecl()) {
    auto func = dyn_cast<FunctionDecl>(decl->getDeclContext());
    if (!func || !func_def_map_.count(func)) return true;
    proto_var = func_def_map_[func]->add_local_vars();
  } else {
    proto_var = g_proto_tu.add_global_vars();
  }
  auto name = decl->getNameAsString();
  proto_var->set_name(name);
  auto type = decl->getType().getAsString();
  proto_var->set_type(type);

  auto *proto_loc = proto_var->mutable_location();
  auto *file = full_location.getFileEntry();
  if (!file) {
    full_location.dump();
    file = context_->getFullLoc(decl->getBeginLoc()).getFileEntry();
  }
  proto_loc->mutable_file_name()->set_index(
      AddStringToTable(file->getName().str()));
  auto line = full_location.getSpellingLineNumber();
  proto_loc->set_line(line);
  auto col = full_location.getSpellingColumnNumber();
  proto_loc->set_column(col);
  auto tokenLen = Lexer::MeasureTokenLength(decl->getLocation(),
                                            context_->getSourceManager(),
                                            context_->getLangOpts());
  proto_loc->set_length(tokenLen);

  return true;
}

static FullSourceLoc GetVarDeclLoc(ASTContext *ctx, DeclRefExpr *decl) {
  auto loc = decl->getLocation();
  auto spell_loc = ctx->getSourceManager().getSpellingLoc(loc);
  auto expan_loc = ctx->getSourceManager().getExpansionLoc(loc);
  if (spell_loc != expan_loc) return FullSourceLoc();
  FullSourceLoc full_location = ctx->getFullLoc(loc);
  return full_location;
}
bool TopsAstVisitor::VisitDeclRefExpr(DeclRefExpr *expr) {
  ValueDecl *referenced_decl = expr->getDecl();

  // 仅保留对变量或函数参数的引用
  if (!isa<VarDecl>(referenced_decl) && !isa<ParmVarDecl>(referenced_decl))
    return true;

  auto full_location = GetVarDeclLoc(context_, expr);
  if (!full_location.isValid()) return true;

  auto *proto_ref = g_proto_tu.add_decl_refs();
  auto name = referenced_decl->getNameAsString();
  if (name == "addrd") full_location.dump();
  proto_ref->set_referenced_name((referenced_decl->getNameAsString()));
  proto_ref->set_referenced_type((referenced_decl->getType().getAsString()));

  auto *proto_loc = proto_ref->mutable_location();
  proto_loc->mutable_file_name()->set_index(
      AddStringToTable(full_location.getFileEntry()->getName().str()));
  proto_loc->set_line(full_location.getSpellingLineNumber());
  proto_loc->set_column(full_location.getSpellingColumnNumber());
  auto tokenLen = Lexer::MeasureTokenLength(expr->getLocation(),
                                            context_->getSourceManager(),
                                            context_->getLangOpts());
  proto_loc->set_length(tokenLen);

  return true;
}

bool TopsAstVisitor::VisitCallExpr(CallExpr *call_expr) {
  const Expr *callee_expr = call_expr->getCallee();
  const auto *decl_ref =
          dyn_cast<DeclRefExpr>(callee_expr->IgnoreImpCasts());
  if (!decl_ref)
    return true;


  FullSourceLoc full_location = context_->getFullLoc(decl_ref->getLocation());
  if (!full_location.isValid()) return true;

  if (full_location.isMacroID())
    full_location = full_location.getExpansionLoc();

  auto *proto_call = g_proto_tu.add_func_calls();

  // 设置调用位置
  auto *call_loc = proto_call->mutable_location();
  call_loc->mutable_file_name()->set_index(
      AddStringToTable(full_location.getFileEntry()->getName().str()));
  call_loc->set_line(full_location.getSpellingLineNumber());
  call_loc->set_column(full_location.getSpellingColumnNumber());
  auto len = Lexer::MeasureTokenLength(decl_ref->getLocation(),
                                       context_->getSourceManager(),
                                       context_->getLangOpts());
  call_loc->set_length(len);

  // 设置被调用函数信息
  if (FunctionDecl *callee = call_expr->getDirectCallee()) {
    if (!this->func_decl_map_.count(callee)) {
      auto *proto = g_proto_tu.add_func_decls();
      FillFuncInfo(proto, callee, context_);
      this->func_decl_map_[callee] = g_proto_tu.func_decls_size() - 1;
    }
    proto_call->set_func_decl_index(this->func_decl_map_[callee]);
    proto_call->set_name(callee->getNameAsString());
  }

  return true;
}

void TopsAstVisitor::SerializeToProtobuf(const std::string &output_file) {
  std::ofstream output(output_file, std::ios::binary);
  llvm::errs() << "Writing to " << output_file << "\n";
  if (!g_proto_tu.SerializeToOstream(&output)) {
    llvm::errs() << "Failed to serialize data to " << output_file << "\n";
  }
}

//-----------------------------------------------------------------------------
// ASTConsumer
//-----------------------------------------------------------------------------
class TopsASTConsumer : public clang::ASTConsumer {
 public:
  explicit TopsASTConsumer(ASTContext *ctx, CompilerInstance &ci)
      : visitor_(ctx), ci_(ci) {}

  void HandleTranslationUnit(clang::ASTContext &ctx) override {
    visitor_.CollectTranslationUnitInfo(ci_);  // 收集文件路径和头文件
    visitor_.TraverseDecl(ctx.getTranslationUnitDecl());
    visitor_.SerializeToProtobuf(g_output_file);  // 序列化到 Protobuf 文件
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
  std::unique_ptr<clang::ASTConsumer> CreateASTConsumer(
      clang::CompilerInstance &compiler, llvm::StringRef in_file) override {
    compiler.getPreprocessor().addPPCallbacks(
        std::make_unique<MyPPCallbacks>());
    return std::make_unique<TopsASTConsumer>(&compiler.getASTContext(),
                                             compiler);
  }

  bool ParseArgs(const CompilerInstance &ci,
                 const std::vector<std::string> &args) override {
    if (args.empty()) {
      llvm::errs() << "Tops LSP plugin output arguments provided.\n";
      return false;
    }
    g_output_file = args.front();
    return true;
  }
};

void MyPPCallbacks::InclusionDirective(
    SourceLocation hash_loc, const Token &include_tok, StringRef file_name,
    bool is_angled, CharSourceRange filename_range, const FileEntry *file,
    StringRef search_path, StringRef relative_path, const Module *imported,
    SrcMgr::CharacteristicKind file_type) {
  struct stat file_stat{};
  const auto header = file->getName();
  auto str_idx = AddStringToTable(header.str());
  if (processed_.count(str_idx)) return;
  if (stat(header.str().c_str(), &file_stat) == 0) {
    auto *proto_header = g_proto_tu.add_included_headers();
    proto_header->mutable_file_name()->set_index(str_idx);
    processed_.insert(str_idx);
  }
}

//-----------------------------------------------------------------------------
// Registration
//-----------------------------------------------------------------------------
static FrontendPluginRegistry::Add<FindSymbolAction> x(
    /*Name=*/"tops-lsp", /*Description=*/"The tops lsp plugin");
