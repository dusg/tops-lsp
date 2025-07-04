// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v3.6.1
// source: data.proto

package data

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type DiagnosticSeverity int32

const (
	DiagnosticSeverity_Ignore      DiagnosticSeverity = 0
	DiagnosticSeverity_Error       DiagnosticSeverity = 1
	DiagnosticSeverity_Warning     DiagnosticSeverity = 2
	DiagnosticSeverity_Information DiagnosticSeverity = 3
	DiagnosticSeverity_Hint        DiagnosticSeverity = 4
)

// Enum value maps for DiagnosticSeverity.
var (
	DiagnosticSeverity_name = map[int32]string{
		0: "Ignore",
		1: "Error",
		2: "Warning",
		3: "Information",
		4: "Hint",
	}
	DiagnosticSeverity_value = map[string]int32{
		"Ignore":      0,
		"Error":       1,
		"Warning":     2,
		"Information": 3,
		"Hint":        4,
	}
)

func (x DiagnosticSeverity) Enum() *DiagnosticSeverity {
	p := new(DiagnosticSeverity)
	*p = x
	return p
}

func (x DiagnosticSeverity) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DiagnosticSeverity) Descriptor() protoreflect.EnumDescriptor {
	return file_data_proto_enumTypes[0].Descriptor()
}

func (DiagnosticSeverity) Type() protoreflect.EnumType {
	return &file_data_proto_enumTypes[0]
}

func (x DiagnosticSeverity) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use DiagnosticSeverity.Descriptor instead.
func (DiagnosticSeverity) EnumDescriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{0}
}

type Variable_VarType int32

const (
	Variable_GLOBAL    Variable_VarType = 0 // 全局变量
	Variable_LOCAL     Variable_VarType = 1 // 局部变量
	Variable_PARAMETER Variable_VarType = 2 // 参数
)

// Enum value maps for Variable_VarType.
var (
	Variable_VarType_name = map[int32]string{
		0: "GLOBAL",
		1: "LOCAL",
		2: "PARAMETER",
	}
	Variable_VarType_value = map[string]int32{
		"GLOBAL":    0,
		"LOCAL":     1,
		"PARAMETER": 2,
	}
)

func (x Variable_VarType) Enum() *Variable_VarType {
	p := new(Variable_VarType)
	*p = x
	return p
}

func (x Variable_VarType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Variable_VarType) Descriptor() protoreflect.EnumDescriptor {
	return file_data_proto_enumTypes[1].Descriptor()
}

func (Variable_VarType) Type() protoreflect.EnumType {
	return &file_data_proto_enumTypes[1]
}

func (x Variable_VarType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Variable_VarType.Descriptor instead.
func (Variable_VarType) EnumDescriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{6, 0}
}

type DeclRef_RefType int32

const (
	DeclRef_FUNCTION  DeclRef_RefType = 0 // 函数引用
	DeclRef_VARIABLE  DeclRef_RefType = 1 // 变量引用
	DeclRef_PARAMETER DeclRef_RefType = 2 // 参数引用
)

// Enum value maps for DeclRef_RefType.
var (
	DeclRef_RefType_name = map[int32]string{
		0: "FUNCTION",
		1: "VARIABLE",
		2: "PARAMETER",
	}
	DeclRef_RefType_value = map[string]int32{
		"FUNCTION":  0,
		"VARIABLE":  1,
		"PARAMETER": 2,
	}
)

func (x DeclRef_RefType) Enum() *DeclRef_RefType {
	p := new(DeclRef_RefType)
	*p = x
	return p
}

func (x DeclRef_RefType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DeclRef_RefType) Descriptor() protoreflect.EnumDescriptor {
	return file_data_proto_enumTypes[2].Descriptor()
}

func (DeclRef_RefType) Type() protoreflect.EnumType {
	return &file_data_proto_enumTypes[2]
}

func (x DeclRef_RefType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use DeclRef_RefType.Descriptor instead.
func (DeclRef_RefType) EnumDescriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{7, 0}
}

type StringTable struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Entries       []string               `protobuf:"bytes,1,rep,name=entries,proto3" json:"entries,omitempty"` // 字符串表
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *StringTable) Reset() {
	*x = StringTable{}
	mi := &file_data_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StringTable) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StringTable) ProtoMessage() {}

func (x *StringTable) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StringTable.ProtoReflect.Descriptor instead.
func (*StringTable) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{0}
}

func (x *StringTable) GetEntries() []string {
	if x != nil {
		return x.Entries
	}
	return nil
}

type StringIndex struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Index         uint32                 `protobuf:"varint,1,opt,name=index,proto3" json:"index,omitempty"` // 字符串表索引
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *StringIndex) Reset() {
	*x = StringIndex{}
	mi := &file_data_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *StringIndex) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StringIndex) ProtoMessage() {}

func (x *StringIndex) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StringIndex.ProtoReflect.Descriptor instead.
func (*StringIndex) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{1}
}

func (x *StringIndex) GetIndex() uint32 {
	if x != nil {
		return x.Index
	}
	return 0
}

type Location struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FileName      *StringIndex           `protobuf:"bytes,1,opt,name=file_name,json=fileName,proto3" json:"file_name,omitempty"` // 引用字符串表中的文件名
	Line          uint32                 `protobuf:"varint,2,opt,name=line,proto3" json:"line,omitempty"`
	Column        uint32                 `protobuf:"varint,3,opt,name=column,proto3" json:"column,omitempty"`
	Length        uint32                 `protobuf:"varint,4,opt,name=length,proto3" json:"length,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Location) Reset() {
	*x = Location{}
	mi := &file_data_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Location) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Location) ProtoMessage() {}

func (x *Location) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Location.ProtoReflect.Descriptor instead.
func (*Location) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{2}
}

func (x *Location) GetFileName() *StringIndex {
	if x != nil {
		return x.FileName
	}
	return nil
}

func (x *Location) GetLine() uint32 {
	if x != nil {
		return x.Line
	}
	return 0
}

func (x *Location) GetColumn() uint32 {
	if x != nil {
		return x.Column
	}
	return 0
}

func (x *Location) GetLength() uint32 {
	if x != nil {
		return x.Length
	}
	return 0
}

type FileInfo struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FileName      *StringIndex           `protobuf:"bytes,1,opt,name=file_name,json=fileName,proto3" json:"file_name,omitempty"` // 引用字符串表中的文件名
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FileInfo) Reset() {
	*x = FileInfo{}
	mi := &file_data_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FileInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileInfo) ProtoMessage() {}

func (x *FileInfo) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileInfo.ProtoReflect.Descriptor instead.
func (*FileInfo) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{3}
}

func (x *FileInfo) GetFileName() *StringIndex {
	if x != nil {
		return x.FileName
	}
	return nil
}

type Function struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`                               // 引用字符串表中的函数名
	ReturnType    string                 `protobuf:"bytes,2,opt,name=return_type,json=returnType,proto3" json:"return_type,omitempty"` // 引用字符串表中的返回类型
	Location      *Location              `protobuf:"bytes,3,opt,name=location,proto3" json:"location,omitempty"`
	Parameters    []uint32               `protobuf:"varint,4,rep,packed,name=parameters,proto3" json:"parameters,omitempty"`
	LocalVars     []uint32               `protobuf:"varint,5,rep,packed,name=local_vars,json=localVars,proto3" json:"local_vars,omitempty"`
	IsDefinition  bool                   `protobuf:"varint,6,opt,name=is_definition,json=isDefinition,proto3" json:"is_definition,omitempty"` // 是否是函数定义
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Function) Reset() {
	*x = Function{}
	mi := &file_data_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Function) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Function) ProtoMessage() {}

func (x *Function) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Function.ProtoReflect.Descriptor instead.
func (*Function) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{4}
}

func (x *Function) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Function) GetReturnType() string {
	if x != nil {
		return x.ReturnType
	}
	return ""
}

func (x *Function) GetLocation() *Location {
	if x != nil {
		return x.Location
	}
	return nil
}

func (x *Function) GetParameters() []uint32 {
	if x != nil {
		return x.Parameters
	}
	return nil
}

func (x *Function) GetLocalVars() []uint32 {
	if x != nil {
		return x.LocalVars
	}
	return nil
}

func (x *Function) GetIsDefinition() bool {
	if x != nil {
		return x.IsDefinition
	}
	return false
}

type FunctionCall struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FuncDeclIndex uint32                 `protobuf:"varint,1,opt,name=func_decl_index,json=funcDeclIndex,proto3" json:"func_decl_index,omitempty"` // 引用函数声明的索引
	Name          string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`                                           // 引用字符串表中的函数名
	Location      *Location              `protobuf:"bytes,3,opt,name=location,proto3" json:"location,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FunctionCall) Reset() {
	*x = FunctionCall{}
	mi := &file_data_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FunctionCall) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FunctionCall) ProtoMessage() {}

func (x *FunctionCall) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FunctionCall.ProtoReflect.Descriptor instead.
func (*FunctionCall) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{5}
}

func (x *FunctionCall) GetFuncDeclIndex() uint32 {
	if x != nil {
		return x.FuncDeclIndex
	}
	return 0
}

func (x *FunctionCall) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *FunctionCall) GetLocation() *Location {
	if x != nil {
		return x.Location
	}
	return nil
}

type Variable struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"` // 引用字符串表中的变量名
	Type          string                 `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"` // 引用字符串表中的类型
	Location      *Location              `protobuf:"bytes,3,opt,name=location,proto3" json:"location,omitempty"`
	VarType       Variable_VarType       `protobuf:"varint,4,opt,name=var_type,json=varType,proto3,enum=TopsAstProto.Variable_VarType" json:"var_type,omitempty"` // 变量类型
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Variable) Reset() {
	*x = Variable{}
	mi := &file_data_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Variable) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Variable) ProtoMessage() {}

func (x *Variable) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Variable.ProtoReflect.Descriptor instead.
func (*Variable) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{6}
}

func (x *Variable) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Variable) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *Variable) GetLocation() *Location {
	if x != nil {
		return x.Location
	}
	return nil
}

func (x *Variable) GetVarType() Variable_VarType {
	if x != nil {
		return x.VarType
	}
	return Variable_GLOBAL
}

type DeclRef struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	ReferencedName string                 `protobuf:"bytes,1,opt,name=referenced_name,json=referencedName,proto3" json:"referenced_name,omitempty"`
	RefType        DeclRef_RefType        `protobuf:"varint,2,opt,name=ref_type,json=refType,proto3,enum=TopsAstProto.DeclRef_RefType" json:"ref_type,omitempty"` // 引用类型
	// Types that are valid to be assigned to RefObj:
	//
	//	*DeclRef_Function
	//	*DeclRef_Variable
	RefObj        isDeclRef_RefObj `protobuf_oneof:"ref_obj"`
	Location      *Location        `protobuf:"bytes,5,opt,name=location,proto3" json:"location,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeclRef) Reset() {
	*x = DeclRef{}
	mi := &file_data_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeclRef) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeclRef) ProtoMessage() {}

func (x *DeclRef) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeclRef.ProtoReflect.Descriptor instead.
func (*DeclRef) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{7}
}

func (x *DeclRef) GetReferencedName() string {
	if x != nil {
		return x.ReferencedName
	}
	return ""
}

func (x *DeclRef) GetRefType() DeclRef_RefType {
	if x != nil {
		return x.RefType
	}
	return DeclRef_FUNCTION
}

func (x *DeclRef) GetRefObj() isDeclRef_RefObj {
	if x != nil {
		return x.RefObj
	}
	return nil
}

func (x *DeclRef) GetFunction() uint32 {
	if x != nil {
		if x, ok := x.RefObj.(*DeclRef_Function); ok {
			return x.Function
		}
	}
	return 0
}

func (x *DeclRef) GetVariable() uint32 {
	if x != nil {
		if x, ok := x.RefObj.(*DeclRef_Variable); ok {
			return x.Variable
		}
	}
	return 0
}

func (x *DeclRef) GetLocation() *Location {
	if x != nil {
		return x.Location
	}
	return nil
}

type isDeclRef_RefObj interface {
	isDeclRef_RefObj()
}

type DeclRef_Function struct {
	Function uint32 `protobuf:"varint,3,opt,name=function,proto3,oneof"`
}

type DeclRef_Variable struct {
	Variable uint32 `protobuf:"varint,4,opt,name=variable,proto3,oneof"`
}

func (*DeclRef_Function) isDeclRef_RefObj() {}

func (*DeclRef_Variable) isDeclRef_RefObj() {}

type Position struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Line          uint32                 `protobuf:"varint,1,opt,name=line,proto3" json:"line,omitempty"`
	Character     uint32                 `protobuf:"varint,2,opt,name=character,proto3" json:"character,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Position) Reset() {
	*x = Position{}
	mi := &file_data_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Position) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Position) ProtoMessage() {}

func (x *Position) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Position.ProtoReflect.Descriptor instead.
func (*Position) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{8}
}

func (x *Position) GetLine() uint32 {
	if x != nil {
		return x.Line
	}
	return 0
}

func (x *Position) GetCharacter() uint32 {
	if x != nil {
		return x.Character
	}
	return 0
}

type Range struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Start         *Position              `protobuf:"bytes,1,opt,name=start,proto3" json:"start,omitempty"` // 起始位置
	End           *Position              `protobuf:"bytes,2,opt,name=end,proto3" json:"end,omitempty"`     // 结束位置
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Range) Reset() {
	*x = Range{}
	mi := &file_data_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Range) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Range) ProtoMessage() {}

func (x *Range) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Range.ProtoReflect.Descriptor instead.
func (*Range) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{9}
}

func (x *Range) GetStart() *Position {
	if x != nil {
		return x.Start
	}
	return nil
}

func (x *Range) GetEnd() *Position {
	if x != nil {
		return x.End
	}
	return nil
}

type DiagnosticRelatedInformation struct {
	state         protoimpl.MessageState                 `protogen:"open.v1"`
	Location      *DiagnosticRelatedInformation_Location `protobuf:"bytes,1,opt,name=location,proto3" json:"location,omitempty"` // 错误相关位置
	Message       string                                 `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`   // 错误相关信息
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DiagnosticRelatedInformation) Reset() {
	*x = DiagnosticRelatedInformation{}
	mi := &file_data_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DiagnosticRelatedInformation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DiagnosticRelatedInformation) ProtoMessage() {}

func (x *DiagnosticRelatedInformation) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DiagnosticRelatedInformation.ProtoReflect.Descriptor instead.
func (*DiagnosticRelatedInformation) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{10}
}

func (x *DiagnosticRelatedInformation) GetLocation() *DiagnosticRelatedInformation_Location {
	if x != nil {
		return x.Location
	}
	return nil
}

func (x *DiagnosticRelatedInformation) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type Diagnostic struct {
	state              protoimpl.MessageState          `protogen:"open.v1"`
	Range              *Range                          `protobuf:"bytes,1,opt,name=range,proto3" json:"range,omitempty"`
	Severity           DiagnosticSeverity              `protobuf:"varint,2,opt,name=severity,proto3,enum=TopsAstProto.DiagnosticSeverity" json:"severity,omitempty"` // 错误级别
	Message            string                          `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`                                         // 错误信息
	Source             string                          `protobuf:"bytes,4,opt,name=source,proto3" json:"source,omitempty"`
	RelatedInformation []*DiagnosticRelatedInformation `protobuf:"bytes,5,rep,name=relatedInformation,proto3" json:"relatedInformation,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *Diagnostic) Reset() {
	*x = Diagnostic{}
	mi := &file_data_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Diagnostic) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Diagnostic) ProtoMessage() {}

func (x *Diagnostic) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Diagnostic.ProtoReflect.Descriptor instead.
func (*Diagnostic) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{11}
}

func (x *Diagnostic) GetRange() *Range {
	if x != nil {
		return x.Range
	}
	return nil
}

func (x *Diagnostic) GetSeverity() DiagnosticSeverity {
	if x != nil {
		return x.Severity
	}
	return DiagnosticSeverity_Ignore
}

func (x *Diagnostic) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *Diagnostic) GetSource() string {
	if x != nil {
		return x.Source
	}
	return ""
}

func (x *Diagnostic) GetRelatedInformation() []*DiagnosticRelatedInformation {
	if x != nil {
		return x.RelatedInformation
	}
	return nil
}

type TranslationUnit struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	StringTable     []string               `protobuf:"bytes,1,rep,name=string_table,json=stringTable,proto3" json:"string_table,omitempty"` // 字符串表
	FilePath        string                 `protobuf:"bytes,2,opt,name=file_path,json=filePath,proto3" json:"file_path,omitempty"`          // 引用字符串表中的文件路径
	CompileArgs     string                 `protobuf:"bytes,3,opt,name=compile_args,json=compileArgs,proto3" json:"compile_args,omitempty"`
	IncludedHeaders []*FileInfo            `protobuf:"bytes,4,rep,name=included_headers,json=includedHeaders,proto3" json:"included_headers,omitempty"`
	GlobalVars      []uint32               `protobuf:"varint,7,rep,packed,name=global_vars,json=globalVars,proto3" json:"global_vars,omitempty"`
	DeclRefs        []*DeclRef             `protobuf:"bytes,8,rep,name=decl_refs,json=declRefs,proto3" json:"decl_refs,omitempty"`
	VariableTable   []*Variable            `protobuf:"bytes,10,rep,name=variable_table,json=variableTable,proto3" json:"variable_table,omitempty"` // 变量表
	FunctionTable   []*Function            `protobuf:"bytes,11,rep,name=function_table,json=functionTable,proto3" json:"function_table,omitempty"` // 函数表
	Diagnostics     []*Diagnostic          `protobuf:"bytes,12,rep,name=diagnostics,proto3" json:"diagnostics,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *TranslationUnit) Reset() {
	*x = TranslationUnit{}
	mi := &file_data_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TranslationUnit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TranslationUnit) ProtoMessage() {}

func (x *TranslationUnit) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TranslationUnit.ProtoReflect.Descriptor instead.
func (*TranslationUnit) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{12}
}

func (x *TranslationUnit) GetStringTable() []string {
	if x != nil {
		return x.StringTable
	}
	return nil
}

func (x *TranslationUnit) GetFilePath() string {
	if x != nil {
		return x.FilePath
	}
	return ""
}

func (x *TranslationUnit) GetCompileArgs() string {
	if x != nil {
		return x.CompileArgs
	}
	return ""
}

func (x *TranslationUnit) GetIncludedHeaders() []*FileInfo {
	if x != nil {
		return x.IncludedHeaders
	}
	return nil
}

func (x *TranslationUnit) GetGlobalVars() []uint32 {
	if x != nil {
		return x.GlobalVars
	}
	return nil
}

func (x *TranslationUnit) GetDeclRefs() []*DeclRef {
	if x != nil {
		return x.DeclRefs
	}
	return nil
}

func (x *TranslationUnit) GetVariableTable() []*Variable {
	if x != nil {
		return x.VariableTable
	}
	return nil
}

func (x *TranslationUnit) GetFunctionTable() []*Function {
	if x != nil {
		return x.FunctionTable
	}
	return nil
}

func (x *TranslationUnit) GetDiagnostics() []*Diagnostic {
	if x != nil {
		return x.Diagnostics
	}
	return nil
}

type DiagnosticRelatedInformation_Location struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Uri           string                 `protobuf:"bytes,1,opt,name=uri,proto3" json:"uri,omitempty"`     // 文件 URI
	Range         *Range                 `protobuf:"bytes,2,opt,name=range,proto3" json:"range,omitempty"` // 错误位置范围
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DiagnosticRelatedInformation_Location) Reset() {
	*x = DiagnosticRelatedInformation_Location{}
	mi := &file_data_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DiagnosticRelatedInformation_Location) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DiagnosticRelatedInformation_Location) ProtoMessage() {}

func (x *DiagnosticRelatedInformation_Location) ProtoReflect() protoreflect.Message {
	mi := &file_data_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DiagnosticRelatedInformation_Location.ProtoReflect.Descriptor instead.
func (*DiagnosticRelatedInformation_Location) Descriptor() ([]byte, []int) {
	return file_data_proto_rawDescGZIP(), []int{10, 0}
}

func (x *DiagnosticRelatedInformation_Location) GetUri() string {
	if x != nil {
		return x.Uri
	}
	return ""
}

func (x *DiagnosticRelatedInformation_Location) GetRange() *Range {
	if x != nil {
		return x.Range
	}
	return nil
}

var File_data_proto protoreflect.FileDescriptor

const file_data_proto_rawDesc = "" +
	"\n" +
	"\n" +
	"data.proto\x12\fTopsAstProto\"'\n" +
	"\vStringTable\x12\x18\n" +
	"\aentries\x18\x01 \x03(\tR\aentries\"#\n" +
	"\vStringIndex\x12\x14\n" +
	"\x05index\x18\x01 \x01(\rR\x05index\"\x86\x01\n" +
	"\bLocation\x126\n" +
	"\tfile_name\x18\x01 \x01(\v2\x19.TopsAstProto.StringIndexR\bfileName\x12\x12\n" +
	"\x04line\x18\x02 \x01(\rR\x04line\x12\x16\n" +
	"\x06column\x18\x03 \x01(\rR\x06column\x12\x16\n" +
	"\x06length\x18\x04 \x01(\rR\x06length\"B\n" +
	"\bFileInfo\x126\n" +
	"\tfile_name\x18\x01 \x01(\v2\x19.TopsAstProto.StringIndexR\bfileName\"\xd7\x01\n" +
	"\bFunction\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12\x1f\n" +
	"\vreturn_type\x18\x02 \x01(\tR\n" +
	"returnType\x122\n" +
	"\blocation\x18\x03 \x01(\v2\x16.TopsAstProto.LocationR\blocation\x12\x1e\n" +
	"\n" +
	"parameters\x18\x04 \x03(\rR\n" +
	"parameters\x12\x1d\n" +
	"\n" +
	"local_vars\x18\x05 \x03(\rR\tlocalVars\x12#\n" +
	"\ris_definition\x18\x06 \x01(\bR\fisDefinition\"~\n" +
	"\fFunctionCall\x12&\n" +
	"\x0ffunc_decl_index\x18\x01 \x01(\rR\rfuncDeclIndex\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\x122\n" +
	"\blocation\x18\x03 \x01(\v2\x16.TopsAstProto.LocationR\blocation\"\xd2\x01\n" +
	"\bVariable\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12\x12\n" +
	"\x04type\x18\x02 \x01(\tR\x04type\x122\n" +
	"\blocation\x18\x03 \x01(\v2\x16.TopsAstProto.LocationR\blocation\x129\n" +
	"\bvar_type\x18\x04 \x01(\x0e2\x1e.TopsAstProto.Variable.VarTypeR\avarType\"/\n" +
	"\aVarType\x12\n" +
	"\n" +
	"\x06GLOBAL\x10\x00\x12\t\n" +
	"\x05LOCAL\x10\x01\x12\r\n" +
	"\tPARAMETER\x10\x02\"\x9d\x02\n" +
	"\aDeclRef\x12'\n" +
	"\x0freferenced_name\x18\x01 \x01(\tR\x0ereferencedName\x128\n" +
	"\bref_type\x18\x02 \x01(\x0e2\x1d.TopsAstProto.DeclRef.RefTypeR\arefType\x12\x1c\n" +
	"\bfunction\x18\x03 \x01(\rH\x00R\bfunction\x12\x1c\n" +
	"\bvariable\x18\x04 \x01(\rH\x00R\bvariable\x122\n" +
	"\blocation\x18\x05 \x01(\v2\x16.TopsAstProto.LocationR\blocation\"4\n" +
	"\aRefType\x12\f\n" +
	"\bFUNCTION\x10\x00\x12\f\n" +
	"\bVARIABLE\x10\x01\x12\r\n" +
	"\tPARAMETER\x10\x02B\t\n" +
	"\aref_obj\"<\n" +
	"\bPosition\x12\x12\n" +
	"\x04line\x18\x01 \x01(\rR\x04line\x12\x1c\n" +
	"\tcharacter\x18\x02 \x01(\rR\tcharacter\"_\n" +
	"\x05Range\x12,\n" +
	"\x05start\x18\x01 \x01(\v2\x16.TopsAstProto.PositionR\x05start\x12(\n" +
	"\x03end\x18\x02 \x01(\v2\x16.TopsAstProto.PositionR\x03end\"\xd2\x01\n" +
	"\x1cDiagnosticRelatedInformation\x12O\n" +
	"\blocation\x18\x01 \x01(\v23.TopsAstProto.DiagnosticRelatedInformation.LocationR\blocation\x12\x18\n" +
	"\amessage\x18\x02 \x01(\tR\amessage\x1aG\n" +
	"\bLocation\x12\x10\n" +
	"\x03uri\x18\x01 \x01(\tR\x03uri\x12)\n" +
	"\x05range\x18\x02 \x01(\v2\x13.TopsAstProto.RangeR\x05range\"\x83\x02\n" +
	"\n" +
	"Diagnostic\x12)\n" +
	"\x05range\x18\x01 \x01(\v2\x13.TopsAstProto.RangeR\x05range\x12<\n" +
	"\bseverity\x18\x02 \x01(\x0e2 .TopsAstProto.DiagnosticSeverityR\bseverity\x12\x18\n" +
	"\amessage\x18\x03 \x01(\tR\amessage\x12\x16\n" +
	"\x06source\x18\x04 \x01(\tR\x06source\x12Z\n" +
	"\x12relatedInformation\x18\x05 \x03(\v2*.TopsAstProto.DiagnosticRelatedInformationR\x12relatedInformation\"\xc6\x03\n" +
	"\x0fTranslationUnit\x12!\n" +
	"\fstring_table\x18\x01 \x03(\tR\vstringTable\x12\x1b\n" +
	"\tfile_path\x18\x02 \x01(\tR\bfilePath\x12!\n" +
	"\fcompile_args\x18\x03 \x01(\tR\vcompileArgs\x12A\n" +
	"\x10included_headers\x18\x04 \x03(\v2\x16.TopsAstProto.FileInfoR\x0fincludedHeaders\x12\x1f\n" +
	"\vglobal_vars\x18\a \x03(\rR\n" +
	"globalVars\x122\n" +
	"\tdecl_refs\x18\b \x03(\v2\x15.TopsAstProto.DeclRefR\bdeclRefs\x12=\n" +
	"\x0evariable_table\x18\n" +
	" \x03(\v2\x16.TopsAstProto.VariableR\rvariableTable\x12=\n" +
	"\x0efunction_table\x18\v \x03(\v2\x16.TopsAstProto.FunctionR\rfunctionTable\x12:\n" +
	"\vdiagnostics\x18\f \x03(\v2\x18.TopsAstProto.DiagnosticR\vdiagnostics*S\n" +
	"\x12DiagnosticSeverity\x12\n" +
	"\n" +
	"\x06Ignore\x10\x00\x12\t\n" +
	"\x05Error\x10\x01\x12\v\n" +
	"\aWarning\x10\x02\x12\x0f\n" +
	"\vInformation\x10\x03\x12\b\n" +
	"\x04Hint\x10\x04B\tZ\a./;datab\x06proto3"

var (
	file_data_proto_rawDescOnce sync.Once
	file_data_proto_rawDescData []byte
)

func file_data_proto_rawDescGZIP() []byte {
	file_data_proto_rawDescOnce.Do(func() {
		file_data_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_data_proto_rawDesc), len(file_data_proto_rawDesc)))
	})
	return file_data_proto_rawDescData
}

var file_data_proto_enumTypes = make([]protoimpl.EnumInfo, 3)
var file_data_proto_msgTypes = make([]protoimpl.MessageInfo, 14)
var file_data_proto_goTypes = []any{
	(DiagnosticSeverity)(0),                       // 0: TopsAstProto.DiagnosticSeverity
	(Variable_VarType)(0),                         // 1: TopsAstProto.Variable.VarType
	(DeclRef_RefType)(0),                          // 2: TopsAstProto.DeclRef.RefType
	(*StringTable)(nil),                           // 3: TopsAstProto.StringTable
	(*StringIndex)(nil),                           // 4: TopsAstProto.StringIndex
	(*Location)(nil),                              // 5: TopsAstProto.Location
	(*FileInfo)(nil),                              // 6: TopsAstProto.FileInfo
	(*Function)(nil),                              // 7: TopsAstProto.Function
	(*FunctionCall)(nil),                          // 8: TopsAstProto.FunctionCall
	(*Variable)(nil),                              // 9: TopsAstProto.Variable
	(*DeclRef)(nil),                               // 10: TopsAstProto.DeclRef
	(*Position)(nil),                              // 11: TopsAstProto.Position
	(*Range)(nil),                                 // 12: TopsAstProto.Range
	(*DiagnosticRelatedInformation)(nil),          // 13: TopsAstProto.DiagnosticRelatedInformation
	(*Diagnostic)(nil),                            // 14: TopsAstProto.Diagnostic
	(*TranslationUnit)(nil),                       // 15: TopsAstProto.TranslationUnit
	(*DiagnosticRelatedInformation_Location)(nil), // 16: TopsAstProto.DiagnosticRelatedInformation.Location
}
var file_data_proto_depIdxs = []int32{
	4,  // 0: TopsAstProto.Location.file_name:type_name -> TopsAstProto.StringIndex
	4,  // 1: TopsAstProto.FileInfo.file_name:type_name -> TopsAstProto.StringIndex
	5,  // 2: TopsAstProto.Function.location:type_name -> TopsAstProto.Location
	5,  // 3: TopsAstProto.FunctionCall.location:type_name -> TopsAstProto.Location
	5,  // 4: TopsAstProto.Variable.location:type_name -> TopsAstProto.Location
	1,  // 5: TopsAstProto.Variable.var_type:type_name -> TopsAstProto.Variable.VarType
	2,  // 6: TopsAstProto.DeclRef.ref_type:type_name -> TopsAstProto.DeclRef.RefType
	5,  // 7: TopsAstProto.DeclRef.location:type_name -> TopsAstProto.Location
	11, // 8: TopsAstProto.Range.start:type_name -> TopsAstProto.Position
	11, // 9: TopsAstProto.Range.end:type_name -> TopsAstProto.Position
	16, // 10: TopsAstProto.DiagnosticRelatedInformation.location:type_name -> TopsAstProto.DiagnosticRelatedInformation.Location
	12, // 11: TopsAstProto.Diagnostic.range:type_name -> TopsAstProto.Range
	0,  // 12: TopsAstProto.Diagnostic.severity:type_name -> TopsAstProto.DiagnosticSeverity
	13, // 13: TopsAstProto.Diagnostic.relatedInformation:type_name -> TopsAstProto.DiagnosticRelatedInformation
	6,  // 14: TopsAstProto.TranslationUnit.included_headers:type_name -> TopsAstProto.FileInfo
	10, // 15: TopsAstProto.TranslationUnit.decl_refs:type_name -> TopsAstProto.DeclRef
	9,  // 16: TopsAstProto.TranslationUnit.variable_table:type_name -> TopsAstProto.Variable
	7,  // 17: TopsAstProto.TranslationUnit.function_table:type_name -> TopsAstProto.Function
	14, // 18: TopsAstProto.TranslationUnit.diagnostics:type_name -> TopsAstProto.Diagnostic
	12, // 19: TopsAstProto.DiagnosticRelatedInformation.Location.range:type_name -> TopsAstProto.Range
	20, // [20:20] is the sub-list for method output_type
	20, // [20:20] is the sub-list for method input_type
	20, // [20:20] is the sub-list for extension type_name
	20, // [20:20] is the sub-list for extension extendee
	0,  // [0:20] is the sub-list for field type_name
}

func init() { file_data_proto_init() }
func file_data_proto_init() {
	if File_data_proto != nil {
		return
	}
	file_data_proto_msgTypes[7].OneofWrappers = []any{
		(*DeclRef_Function)(nil),
		(*DeclRef_Variable)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_data_proto_rawDesc), len(file_data_proto_rawDesc)),
			NumEnums:      3,
			NumMessages:   14,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_data_proto_goTypes,
		DependencyIndexes: file_data_proto_depIdxs,
		EnumInfos:         file_data_proto_enumTypes,
		MessageInfos:      file_data_proto_msgTypes,
	}.Build()
	File_data_proto = out.File
	file_data_proto_goTypes = nil
	file_data_proto_depIdxs = nil
}
