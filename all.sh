#! /usr/bin/bash
set -x

make all
# pip install antlr4-tools

# cd parser && antlr4 -Dlanguage=Go Cpp.g4
# cd ..

# protoc --cpp_out=./clang-plugin/proto --proto_path clang-plugin/proto/ clang-plugin/proto/TopsAstProto.proto

# go build -o extension/bin/tops-lsp
# cd extension && npm install && npm version patch && npm run package