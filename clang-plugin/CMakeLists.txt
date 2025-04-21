cmake_minimum_required(VERSION 3.25)
project(clang-tops-lsp-plugin)
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(LLVM_INSTALL_DIR "/home/carl.du/work/caps/build/llvm" CACHE PATH "Clang installation directory")
list(APPEND CMAKE_PREFIX_PATH "/home/carl.du/work/caps/build/llvm/cmake/clang/")
include_directories(SYSTEM "${LLVM_INSTALL_DIR}/include")

if(NOT LLVM_ENABLE_RTTI)
    set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fno-rtti")
endif()

include(CheckCXXCompilerFlag)
check_cxx_compiler_flag("-fvisibility-inlines-hidden"
  SUPPORTS_FVISIBILITY_INLINES_HIDDEN_FLAG)
if(${SUPPORTS_FVISIBILITY_INLINES_HIDDEN_FLAG} EQUAL "1")
  set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -fvisibility-inlines-hidden")
endif()

add_compile_options(-w -fPIC)

set(protobuf_WITH_ZLIB OFF)
set(protobuf_ALLOW_CCACHE ON)
set(protobuf_FORCE_FETCH_DEPENDENCIES ON)
set(protobuf_BUILD_TESTS OFF)
add_subdirectory(3rd/protobuf)

include(3rd/protobuf/cmake/protobuf-generate.cmake)
file(GLOB PROTO_FILES "proto/*.proto")
protobuf_generate(OUT_VAR PB_FILEs PROTOS ${PROTO_FILES})

# add_compile_definitions(GOOGLE_PROTOBUF_NO_RTTI)
include_directories(${CMAKE_BINARY_DIR})

add_library(tops-lsp SHARED library.cpp ${PB_FILEs})
target_link_libraries(tops-lsp PRIVATE libprotobuf -static-libstdc++ -static-libgcc)

add_executable(decoder decoder.cpp proto/TopsAstProto.pb.cc)
target_link_libraries(decoder PRIVATE libprotobuf -static-libstdc++ -static-libgcc)