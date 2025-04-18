#include <fstream>
#include <iostream>
#include <google/protobuf/text_format.h>
#include <google/protobuf/message.h>
#include "proto/TopsAstProto.pb.h" // 包含生成的proto头文件
// ...existing code...

void ConvertProtoToText(const std::string& input_file, const std::string& output_file, google::protobuf::Message& message) {
    // 读取序列化的 proto 文件
    std::ifstream input(input_file, std::ios::binary);
    if (!input) {
        std::cerr << "无法打开输入文件: " << input_file << std::endl;
        return;
    }
    if (!message.ParseFromIstream(&input)) {
        std::cerr << "解析序列化文件失败: " << input_file << std::endl;
        return;
    }
    input.close();

    // 转换为可读性文本格式
    std::string text_format;
    if (!google::protobuf::TextFormat::PrintToString(message, &text_format)) {
        std::cerr << "转换为文本格式失败" << std::endl;
        return;
    }

    // 写入输出文件
    std::ofstream output(output_file);
    if (!output) {
        std::cerr << "无法打开输出文件: " << output_file << std::endl;
        return;
    }
    output << text_format;
    output.close();

    std::cout << "转换完成: " << output_file << std::endl;
}

// ...existing code...

int main(int argc, char* argv[]) {
    if (argc != 3) {
        std::cerr << "用法: " << argv[0] << " <输入文件> <输出文件>" << std::endl;
        return 1;
    }

    const std::string input_file = argv[1];
    const std::string output_file = argv[2];

    TopsAstProto::TranslationUnit translation_unit;
    ConvertProtoToText(input_file, output_file, translation_unit);

    return 0;
}