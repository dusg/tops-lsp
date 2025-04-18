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

  void CollectTranslationUnitInfo(CompilerInstance &ci);
  void SerializeToProtobuf(const std::string &output_file);

  void SetContext(ASTContext *ctx) {
    context_ = ctx;  // 设置 AST 上下文
  }

 private:
  ASTContext *context_;
  std::unordered_map<FunctionDecl*, TopsAstProto::Function*> func_def_map_;
};

void TopsAstVisitor::CollectTranslationUnitInfo(CompilerInstance &ci) {
  // 设置文件路径
  g_proto_tu.mutable_file_path()->set_index(
      AddStringToTable(ci.getFrontendOpts().Inputs[0].getFile().str()));

  // 设置源文件及包含的头文件路径和最后修改时间
  struct stat file_stat;
  if (stat(ci.getFrontendOpts().Inputs[0].getFile().str().c_str(),
           &file_stat) == 0) {
    auto *proto_header = g_proto_tu.add_included_headers();
    proto_header->mutable_file_name()->set_index(
        AddStringToTable(ci.getFrontendOpts().Inputs[0].getFile().str()));
  }

  // 设置编译参数
  g_proto_tu.set_compile_args(GetCommandLineArgs());
}

bool TopsAstVisitor::VisitCXXRecordDecl(CXXRecordDecl *Declaration) {
  return true;
}

bool TopsAstVisitor::VisitFunctionDecl(FunctionDecl *decl) {
  FullSourceLoc full_location = context_->getFullLoc(decl->getLocation());
  if (!full_location.isValid()) return true;

  if (full_location.isMacroID())
    full_location = full_location.getExpansionLoc();

  auto *proto_func = decl->isThisDeclarationADefinition()
                         ? g_proto_tu.add_func_defs()
                         : g_proto_tu.add_func_decls();

  proto_func->mutable_name()->set_index(
      AddStringToTable(decl->getNameAsString()));
  proto_func->mutable_return_type()->set_index(
      AddStringToTable(decl->getReturnType().getAsString()));

  auto *func_loc = proto_func->mutable_location();
  func_loc->mutable_file_name()->set_index(
      AddStringToTable(full_location.getFileEntry()->getName().str()));
  func_loc->set_line(full_location.getSpellingLineNumber());
  func_loc->set_column(full_location.getSpellingColumnNumber());
  auto len = Lexer::MeasureTokenLength(decl->getLocation(),
                                       context_->getSourceManager(),
                                       context_->getLangOpts());
  func_loc->set_length(len);

  for (ParmVarDecl *param : decl->parameters()) {
    auto *proto_param = proto_func->add_parameters();
    proto_param->mutable_name()->set_index(
        AddStringToTable(param->getNameAsString()));
    proto_param->mutable_type()->set_index(
        AddStringToTable(param->getType().getAsString()));

    FullSourceLoc param_location = context_->getFullLoc(param->getLocation());
    auto *proto_param_loc = proto_param->mutable_location();
    proto_param_loc->mutable_file_name()->set_index(
        AddStringToTable(full_location.getFileEntry()->getName().str()));
    proto_param_loc->set_line(param_location.getSpellingLineNumber());
    proto_param_loc->set_column(param_location.getSpellingColumnNumber());
    auto tokenLen = Lexer::MeasureTokenLength(param->getLocation(),
                                              context_->getSourceManager(),
                                              context_->getLangOpts());
    proto_param_loc->set_length(tokenLen);
  }

  if (decl->isThisDeclarationADefinition() )
    this->func_def_map_[decl] = proto_func;
  return true;
}

bool TopsAstVisitor::VisitVarDecl(VarDecl *decl) {
  if (dyn_cast<ParmVarDecl>(decl)) return true;
  FullSourceLoc full_location = context_->getFullLoc(decl->getBeginLoc());
  if (!full_location.isValid()) return true;

  if (full_location.isMacroID())
    full_location = full_location.getExpansionLoc();

  TopsAstProto::Variable *proto_var = nullptr;
  if (decl->isLocalVarDecl()) {
    auto func = dyn_cast<FunctionDecl>(decl->getDeclContext());
    if (!func || !func_def_map_.count(func))
      return true;
    proto_var = func_def_map_[func]->add_local_vars();
  } else {
    proto_var = g_proto_tu.add_global_vars();
  }
  proto_var->mutable_name()->set_index(
      AddStringToTable(decl->getNameAsString()));
  proto_var->mutable_type()->set_index(
      AddStringToTable(decl->getType().getAsString()));

  auto *proto_loc = proto_var->mutable_location();
  proto_loc->mutable_file_name()->set_index(
      AddStringToTable(full_location.getFileEntry()->getName().str()));
  proto_loc->set_line(full_location.getSpellingLineNumber());
  proto_loc->set_column(full_location.getSpellingColumnNumber());
  auto tokenLen = Lexer::MeasureTokenLength(decl->getLocation(),
                                            context_->getSourceManager(),
                                            context_->getLangOpts());
  proto_loc->set_length(tokenLen);

  return true;
}

bool TopsAstVisitor::VisitDeclRefExpr(DeclRefExpr *expr) {
  FullSourceLoc full_location = context_->getFullLoc(expr->getBeginLoc());
  if (!full_location.isValid()) return true;

  if (full_location.isMacroID())
    full_location = full_location.getExpansionLoc();

  ValueDecl *referenced_decl = expr->getDecl();

  // 仅保留对变量或函数参数的引用
  if (!isa<VarDecl>(referenced_decl) && !isa<ParmVarDecl>(referenced_decl))
    return true;

  auto *proto_ref = g_proto_tu.add_decl_refs();
  proto_ref->mutable_referenced_name()->set_index(
      AddStringToTable(referenced_decl->getNameAsString()));
  proto_ref->mutable_referenced_type()->set_index(
      AddStringToTable(referenced_decl->getType().getAsString()));

  auto *proto_loc = proto_ref->mutable_location();
  proto_loc->mutable_file_name()->set_index(
      AddStringToTable(full_location.getFileEntry()->getName().str()));
  proto_loc->set_line(full_location.getSpellingLineNumber());
  proto_loc->set_column(full_location.getSpellingColumnNumber());
  proto_loc->set_length(static_cast<unsigned>(
      expr->getSourceRange().getEnd().getRawEncoding() -
      expr->getSourceRange().getBegin().getRawEncoding()));

  return true;
}

void TopsAstVisitor::SerializeToProtobuf(const std::string &output_file) {
  std::ofstream output(output_file, std::ios::binary);
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
    visitor_.SerializeToProtobuf("output.idx");  // 序列化到 Protobuf 文件
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
  if (stat(header.str().c_str(), &file_stat) == 0) {
    auto *proto_header = g_proto_tu.add_included_headers();
    proto_header->mutable_file_name()->set_index(
        AddStringToTable(header.str()));
  }
}

//-----------------------------------------------------------------------------
// Registration
//-----------------------------------------------------------------------------
static FrontendPluginRegistry::Add<FindSymbolAction> x(
    /*Name=*/"tops-lsp", /*Description=*/"The tops lsp plugin");
