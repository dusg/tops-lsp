//
// Created by carl.du on 5/6/25.
//

#ifndef TRANSLATIONUNITWARPPER_H
#define TRANSLATIONUNITWARPPER_H

#include <fstream>

#include "proto/TopsAstProto.pb.h"

class TranslationUnitWrapper {
  TopsAstProto::TranslationUnit tu_;
  std::map<std::string, uint32_t> str_table_map_;

 public:
  TranslationUnitWrapper() = default;
  std::pair<uint32_t, TopsAstProto::Variable*> addVariable(TopsAstProto::Variable_VarType type);

  std::pair<uint32_t, TopsAstProto::Variable*> addParam() {
    return addVariable(TopsAstProto::Variable_VarType::Variable_VarType_PARAMETER);
  }
  std::pair<uint32_t, TopsAstProto::Variable*> addLocalVar() {
    return addVariable(TopsAstProto::Variable_VarType::Variable_VarType_LOCAL);
  }
  std::pair<uint32_t, TopsAstProto::Variable*> addGlobalVar() {
    auto entry = addVariable(TopsAstProto::Variable_VarType::Variable_VarType_GLOBAL);
    tu_.add_global_vars(entry.first);
    return entry;
  }
  std::pair<uint32_t, TopsAstProto::Function*> addFunction();
  TopsAstProto::Function* getFunction(uint32_t idx) { return tu_.mutable_function_table(idx); }
  uint32_t AddStringToTable(const std::string& str);
  void setSrcFile(const std::string& src_file) { tu_.set_file_path(src_file); }
  void setCompileArgs(const std::string& compile_args) { tu_.set_compile_args(compile_args); }

  TopsAstProto::DeclRef* addDeclRef() { return tu_.add_decl_refs(); }

  void addIncludedHeader(const std::string& header) {
    auto* proto_header = tu_.add_included_headers();
    proto_header->mutable_file_name()->set_index(AddStringToTable(header));
  }
  bool SerializeToProtobuf(const std::string& output_file) {
    std::ofstream output(output_file, std::ios::binary);
    return tu_.SerializeToOstream(&output);
  }
};

#endif  // TRANSLATIONUNITWARPPER_H
