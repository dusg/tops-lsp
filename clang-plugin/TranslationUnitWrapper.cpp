//
// Created by carl.du on 5/6/25.
//

#include "TranslationUnitWrapper.h"

#include <map>
std::pair<uint32_t, TopsAstProto::Variable*> TranslationUnitWrapper::addVariable(TopsAstProto::Variable_VarType type) {
  auto* entry = tu_.add_variable_table();
  entry->set_var_type(type);
  return {tu_.variable_table_size()-1, entry};
}

std::pair<uint32_t, TopsAstProto::Function*> TranslationUnitWrapper::addFunction() {
  auto* entry = tu_.mutable_function_table()->Add();
  return {tu_.function_table_size() - 1, entry};
}
uint32_t TranslationUnitWrapper::AddStringToTable(const std::string& str) {
  auto it = str_table_map_.find(str);
  if (it != str_table_map_.end()) {
    return it->second;
  }

  uint32_t index = tu_.string_table_size();
  tu_.add_string_table(str);
  str_table_map_[str] = index;
  return index;
}
