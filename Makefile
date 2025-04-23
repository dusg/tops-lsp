.PHONY: all antlr build-extension clang-plugin

export GOPATH :=

all: build-extension

antlr: parser/cpp_parser.go

parser/cpp_parser.go: parser/Cpp.g4
	pip install antlr4-tools
	cd parser && antlr4 -Dlanguage=Go Cpp.g4

go-proc: lsp/data/data.pb.go

lsp/data/data.pb.go: lsp/data/data.proto
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
	protoc --go_out=./lsp/data --proto_path lsp/data/ lsp/data/data.proto

# clang-plugin: generate-proto
clang-plugin:
	cd clang-plugin && mkdir -p build && cd build && cmake .. -G Ninja && ninja

build-lsp: antlr go-proc
	go build -o extension/bin/tops-lsp

build-extension: clang-plugin build-lsp
	cd extension && npm install && npm version patch && npm run package

test: clang-plugin
	cd clang-plugin/build && topscc -ltops -arch gcu300 -fsyntax-only /home/carl.du/work/tops-lsp/test-files/test.tops -Xclang -load -Xclang ./libtops-lsp.so -Xclang -plugin -Xclang tops-lsp -w --cuda-device-only

clean-clang-plugin:
	rm -rf clang-plugin/proto/*.pb.*
	rm -rf clang-plugin/build
	
clean:
	rm -rf parser/*.go
	rm -rf extension/bin/tops-lsp
	rm -rf extension/node_modules
	rm -rf extension/*.vsix
	rm -rf extension/out