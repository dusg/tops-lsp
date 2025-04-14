// $antlr-format alignTrailingComments true, columnLimit 150, maxEmptyLinesToKeep 1, reflowComments false, useTab false
// $antlr-format allowShortRulesOnASingleLine true, allowShortBlocksOnASingleLine true, minEmptyLines 0, alignSemicolons ownLine
// $antlr-format alignColons trailing, singleLineOverrulesHangingColon true, alignLexerCommands true, alignLabels true, alignTrailers true

grammar Cpp;

IntegerLiteral:
    DecimalLiteral Integersuffix?
    | OctalLiteral Integersuffix?
    | HexadecimalLiteral Integersuffix?
    | BinaryLiteral Integersuffix?
;

CharacterLiteral: ('u' | 'U' | 'L')? '\'' Cchar+ '\'';

FloatingLiteral:
    Fractionalconstant Exponentpart? Floatingsuffix?
    | Digitsequence Exponentpart Floatingsuffix?
;

StringLiteral: Encodingprefix? (Rawstring | '"' Schar* '"');

BooleanLiteral: False_ | True_;

PointerLiteral: Nullptr;

UserDefinedLiteral:
    UserDefinedIntegerLiteral
    | UserDefinedFloatingLiteral
    | UserDefinedStringLiteral
    | UserDefinedCharacterLiteral
;

// MacroKey: '#define' | '#undef' | '#include' | '#pragma' | '#error' | '#line' | '#if' | '#ifdef' | '#ifndef' | '#else' | '#elif' | '#endif';
// MultiLineMacro: '#' (~[\n]*? '\\' '\r'? '\n')+ ~ [\n]+ -> channel (HIDDEN);
MultiLineMacro: '#' (~[\n]*? '\\' '\r'? '\n')+ ~ [\n]+;

// Directive: '#' ~ [\n]* -> channel (HIDDEN);
Directive: '#' ~ [\n]*;
// Directive: MacroKey ~ [\n]*;
/*Keywords*/
// LineDirective: '#';

Alignas: 'alignas';

Alignof: 'alignof';

Asm: 'asm';

Auto: 'auto';

Bool: 'bool';

Break: 'break';

Case: 'case';

Catch: 'catch';

Char: 'char';

Char16: 'char16_t';

Char32: 'char32_t';

Class: 'class';

Const: 'const';

Constexpr: 'constexpr';

Const_cast: 'const_cast';

Continue: 'continue';

Decltype: 'decltype';

Default: 'default';

Delete: 'delete';

Do: 'do';

Double: 'double';

Dynamic_cast: 'dynamic_cast';

Else: 'else';

Enum: 'enum';

Explicit: 'explicit';

Export: 'export';

Extern: 'extern';

//DO NOT RENAME - PYTHON NEEDS True and False
False_: 'false';

Final: 'final';

Float: 'float';

For: 'for';

Friend: 'friend';

Goto: 'goto';

If: 'if';

Inline: 'inline';

Int: 'int';

Long: 'long';

Mutable: 'mutable';

Namespace: 'namespace';

New: 'new';

Noexcept: 'noexcept';

Nullptr: 'nullptr';

Operator: 'operator';

Override: 'override';

Private: 'private';

Protected: 'protected';

Public: 'public';

Register: 'register';

Reinterpret_cast: 'reinterpret_cast';

Return: 'return';

Short: 'short';

Signed: 'signed';

Sizeof: 'sizeof';

Static: 'static';

Static_assert: 'static_assert';

Static_cast: 'static_cast';

Struct: 'struct';

Switch: 'switch';

Template: 'template';

This: 'this';

Thread_local: 'thread_local';

Throw: 'throw';

//DO NOT RENAME - PYTHON NEEDS True and False
True_: 'true';

Try: 'try';

Typedef: 'typedef';

Typeid_: 'typeid';

Typename_: 'typename';

Union: 'union';

Unsigned: 'unsigned';

Using: 'using';

Virtual: 'virtual';

Void: 'void';

Volatile: 'volatile';

Wchar: 'wchar_t';

While: 'while';

Attribute: '__attribute__';
/*Operators*/

LeftParen: '(';

RightParen: ')';

LeftBracket: '[';

RightBracket: ']';

LeftBrace: '{';

RightBrace: '}';

Plus: '+';

Minus: '-';

Star: '*';

Div: '/';

Mod: '%';

Caret: '^';

And: '&';

Or: '|';

Tilde: '~';

Not: '!' | 'not';

Assign: '=';

Less: '<';

Greater: '>';

PlusAssign: '+=';

MinusAssign: '-=';

StarAssign: '*=';

DivAssign: '/=';

ModAssign: '%=';

XorAssign: '^=';

AndAssign: '&=';

OrAssign: '|=';

LeftShiftAssign: '<<=';

RightShiftAssign: '>>=';

Equal: '==';

NotEqual: '!=';

LessEqual: '<=';

GreaterEqual: '>=';

AndAnd: '&&' | 'and';

OrOr: '||' | 'or';

PlusPlus: '++';

MinusMinus: '--';

Comma: ',';

ArrowStar: '->*';

Arrow: '->';

Question: '?';

Colon: ':';

Doublecolon: '::';

Semi: ';';

Dot: '.';

DotStar: '.*';

Ellipsis: '...';

fragment Hexquad: HEXADECIMALDIGIT HEXADECIMALDIGIT HEXADECIMALDIGIT HEXADECIMALDIGIT;

fragment Universalcharactername: '\\u' Hexquad | '\\U' Hexquad Hexquad;

Identifier:
    /*
	 Identifiernondigit | Identifier Identifiernondigit | Identifier DIGIT
	 */ Identifiernondigit (Identifiernondigit | DIGIT)*
;

fragment Identifiernondigit: NONDIGIT | Universalcharactername;

fragment NONDIGIT: [a-zA-Z_];

fragment DIGIT: [0-9];

DecimalLiteral: NONZERODIGIT ('\''? DIGIT)*;

OctalLiteral: '0' ('\''? OCTALDIGIT)*;

HexadecimalLiteral: ('0x' | '0X') HEXADECIMALDIGIT ( '\''? HEXADECIMALDIGIT)*;

BinaryLiteral: ('0b' | '0B') BINARYDIGIT ('\''? BINARYDIGIT)*;

fragment NONZERODIGIT: [1-9];

fragment OCTALDIGIT: [0-7];

fragment HEXADECIMALDIGIT: [0-9a-fA-F];

fragment BINARYDIGIT: [01];

Integersuffix:
    Unsignedsuffix Longsuffix?
    | Unsignedsuffix Longlongsuffix?
    | Longsuffix Unsignedsuffix?
    | Longlongsuffix Unsignedsuffix?
;

fragment Unsignedsuffix: [uU];

fragment Longsuffix: [lL];

fragment Longlongsuffix: 'll' | 'LL';

fragment Cchar: ~ ['\\\r\n] | Escapesequence | Universalcharactername;

fragment Escapesequence: Simpleescapesequence | Octalescapesequence | Hexadecimalescapesequence;

fragment Simpleescapesequence:
    '\\\''
    | '\\"'
    | '\\?'
    | '\\\\'
    | '\\a'
    | '\\b'
    | '\\f'
    | '\\n'
    | '\\r'
    | '\\' ('\r' '\n'? | '\n')
    | '\\t'
    | '\\v'
;

fragment Octalescapesequence:
    '\\' OCTALDIGIT
    | '\\' OCTALDIGIT OCTALDIGIT
    | '\\' OCTALDIGIT OCTALDIGIT OCTALDIGIT
;

fragment Hexadecimalescapesequence: '\\x' HEXADECIMALDIGIT+;

fragment Fractionalconstant: Digitsequence? '.' Digitsequence | Digitsequence '.';

fragment Exponentpart: 'e' SIGN? Digitsequence | 'E' SIGN? Digitsequence;

fragment SIGN: [+-];

fragment Digitsequence: DIGIT ('\''? DIGIT)*;

fragment Floatingsuffix: [flFL];

fragment Encodingprefix: 'u8' | 'u' | 'U' | 'L';

fragment Schar: ~ ["\\\r\n] | Escapesequence | Universalcharactername;

fragment Rawstring: 'R"' ( '\\' ["()] | ~[\r\n (])*? '(' ~[)]*? ')' ( '\\' ["()] | ~[\r\n "])*? '"';

UserDefinedIntegerLiteral:
    DecimalLiteral Udsuffix
    | OctalLiteral Udsuffix
    | HexadecimalLiteral Udsuffix
    | BinaryLiteral Udsuffix
;

UserDefinedFloatingLiteral:
    Fractionalconstant Exponentpart? Udsuffix
    | Digitsequence Exponentpart Udsuffix
;

UserDefinedStringLiteral: StringLiteral Udsuffix;

UserDefinedCharacterLiteral: CharacterLiteral Udsuffix;

fragment Udsuffix: Identifier;

Whitespace: [ \t]+ -> skip;

Newline: ('\r' '\n'? | '\n') -> skip;

BlockComment: '/*' .*? '*/';

LineComment: '//' ~ [\r\n]*;

cpp: stmt* EOF;
stmt:
    Directive
    | MultiLineMacro
    | keyword
    | baseType
    | operator
    | string
    | number
    | identifier
    | comment
    | other
;
number:
    IntegerLiteral
    | FloatingLiteral
    | BooleanLiteral
    | PointerLiteral
    | UserDefinedLiteral
;
string:
     StringLiteral
    | CharacterLiteral
    | UserDefinedStringLiteral
;
identifier: Identifier;
comment: LineComment | BlockComment;
// arrayIdentifier: LeftBrace other+ RightBrace;
// baseVarDecl:
//     baseType Identifier arrayIdentifier? (Assign stmt+)? Semi
//     | baseType Identifier (Comma Identifier)* Semi
//     | baseType Identifier (Comma Identifier)* Assign stmt+ Semi
// ;
// autoVarDecl:
//     Auto Identifier arrayIdentifier? (Assign stmt+)? Semi
//     | Auto Identifier (Comma Identifier)* Semi
//     | Auto Identifier (Comma Identifier)* Assign stmt+ Semi
// ;
other: .;
keyword:
    Typedef
    | Using
    | Namespace
    | Class
    | Struct
    | Union
    | Enum
    | Template
    | Friend
    | Public
    | Private
    | Protected
    | Virtual
    | Override
    | Final
    | Explicit
    | Constexpr
    | Static_assert
    | Return
    | Auto
    | Sizeof
    | Reinterpret_cast
    | Register
    | Const
    | For
;
baseType: Bool | Char | Wchar | Short | Int | Long | Float | Double | Void;
operator:
    Plus
    | Minus
    | Star
    | Div
    | Mod
    | Caret
    | And
    | Or
    | Tilde
    | Not
    | Assign
    | Less
    | Greater
    | PlusAssign
    | MinusAssign
    | StarAssign
    | DivAssign
    | ModAssign
    | XorAssign
    | AndAssign
    | OrAssign
    | LeftShiftAssign
    | RightShiftAssign
    | Equal
    | NotEqual
    | LessEqual
    | GreaterEqual
    | AndAnd
    | OrOr
    | PlusPlus
    | MinusMinus
;
// macro: MacroKey (~ [\n])*;