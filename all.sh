#! /usr/bin/bash
set -x
pip install antlr4-tools

cd parser && antlr4 -Dlanguage=Go Cpp.g4
cd ..

go build -o extension/bin/tops-lsp
cd extension && npm install && npm version patch && npm run package