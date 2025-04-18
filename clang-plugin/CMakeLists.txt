cmake_minimum_required(VERSION 3.25)
project(clang-tops-lsp-plugin)
set(CMAKE_CXX_STANDARD 14)
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

add_compile_definitions(GOOGLE_PROTOBUF_NO_RTTI)

# 静态链接 Protobuf 运行时库
find_package(Protobuf REQUIRED)
set(PROTOBUF_LIBRARIES ${Protobuf_LIBRARIES})
set(PROTOBUF_INCLUDE_DIRS ${Protobuf_INCLUDE_DIRS})
include_directories(${PROTOBUF_INCLUDE_DIRS})
add_library(tops-lsp SHARED library.cpp proto/TopsAstProto.pb.cc)
target_link_libraries(tops-lsp ${PROTOBUF_LIBRARIES} -static-libstdc++ -static-libgcc)

add_executable(decoder decoder.cpp proto/TopsAstProto.pb.cc)
target_link_libraries(decoder ${PROTOBUF_LIBRARIES} -static-libstdc++ -static-libgcc)