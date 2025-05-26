// Code generated from DevcmdLexer.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"sync"
	"unicode"
)

// Suppress unused import error
var _ = fmt.Printf
var _ = sync.Once{}
var _ = unicode.IsLetter

type DevcmdLexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var DevcmdLexerLexerStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	ChannelNames           []string
	ModeNames              []string
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func devcmdlexerLexerInit() {
	staticData := &DevcmdLexerLexerStaticData
	staticData.ChannelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.ModeNames = []string{
		"DEFAULT_MODE", "RAW_CONTENT",
	}
	staticData.LiteralNames = []string{
		"", "'def'", "'watch'", "'stop'", "", "'@'", "'='", "':'", "';'", "'{'",
		"'}'", "'('", "", "'\\'", "'&'", "", "", "'\\$'",
	}
	staticData.SymbolicNames = []string{
		"", "DEF", "WATCH", "STOP", "AT_NAME_LPAREN", "AT", "EQUALS", "COLON",
		"SEMICOLON", "LBRACE", "RBRACE", "LPAREN", "RPAREN", "BACKSLASH", "AMPERSAND",
		"VAR_REF", "SHELL_VAR", "ESCAPED_DOLLAR", "NAME", "NUMBER", "STRING",
		"CONTENT", "COMMENT", "NEWLINE", "WS", "RAW_TEXT", "RAW_RPAREN", "RAW_WS",
	}
	staticData.RuleNames = []string{
		"DEF", "WATCH", "STOP", "AT_NAME_LPAREN", "AT", "EQUALS", "COLON", "SEMICOLON",
		"LBRACE", "RBRACE", "LPAREN", "RPAREN", "BACKSLASH", "AMPERSAND", "VAR_REF",
		"SHELL_VAR", "ESCAPED_DOLLAR", "NAME", "NUMBER", "STRING", "CONTENT",
		"COMMENT", "NEWLINE", "WS", "RAW_TEXT", "RAW_RPAREN", "RAW_WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 27, 207, 6, -1, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3,
		7, 3, 2, 4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9,
		7, 9, 2, 10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7,
		14, 2, 15, 7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19,
		2, 20, 7, 20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2,
		25, 7, 25, 2, 26, 7, 26, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 5, 3, 75, 8,
		3, 10, 3, 12, 3, 78, 9, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 5, 1,
		5, 1, 6, 1, 6, 1, 7, 1, 7, 1, 8, 1, 8, 1, 9, 1, 9, 1, 10, 1, 10, 1, 11,
		1, 11, 1, 12, 1, 12, 1, 13, 1, 13, 1, 14, 1, 14, 1, 14, 1, 14, 1, 14, 5,
		14, 109, 8, 14, 10, 14, 12, 14, 112, 9, 14, 1, 14, 1, 14, 1, 15, 1, 15,
		1, 15, 5, 15, 119, 8, 15, 10, 15, 12, 15, 122, 9, 15, 1, 16, 1, 16, 1,
		16, 1, 17, 1, 17, 5, 17, 129, 8, 17, 10, 17, 12, 17, 132, 9, 17, 1, 18,
		4, 18, 135, 8, 18, 11, 18, 12, 18, 136, 1, 18, 1, 18, 4, 18, 141, 8, 18,
		11, 18, 12, 18, 142, 3, 18, 145, 8, 18, 1, 18, 1, 18, 4, 18, 149, 8, 18,
		11, 18, 12, 18, 150, 3, 18, 153, 8, 18, 1, 19, 1, 19, 1, 19, 1, 19, 5,
		19, 159, 8, 19, 10, 19, 12, 19, 162, 9, 19, 1, 19, 1, 19, 1, 20, 4, 20,
		167, 8, 20, 11, 20, 12, 20, 168, 1, 21, 1, 21, 5, 21, 173, 8, 21, 10, 21,
		12, 21, 176, 9, 21, 1, 21, 1, 21, 1, 22, 3, 22, 181, 8, 22, 1, 22, 1, 22,
		1, 23, 4, 23, 186, 8, 23, 11, 23, 12, 23, 187, 1, 23, 1, 23, 1, 24, 4,
		24, 193, 8, 24, 11, 24, 12, 24, 194, 1, 25, 1, 25, 1, 25, 1, 25, 1, 26,
		4, 26, 202, 8, 26, 11, 26, 12, 26, 203, 1, 26, 1, 26, 0, 0, 27, 2, 1, 4,
		2, 6, 3, 8, 4, 10, 5, 12, 6, 14, 7, 16, 8, 18, 9, 20, 10, 22, 11, 24, 12,
		26, 13, 28, 14, 30, 15, 32, 16, 34, 17, 36, 18, 38, 19, 40, 20, 42, 21,
		44, 22, 46, 23, 48, 24, 50, 25, 52, 26, 54, 27, 2, 0, 1, 10, 2, 0, 65,
		90, 97, 122, 5, 0, 45, 45, 48, 57, 65, 90, 95, 95, 97, 122, 4, 0, 48, 57,
		65, 90, 95, 95, 97, 122, 1, 0, 48, 57, 3, 0, 10, 10, 13, 13, 34, 34, 11,
		0, 9, 10, 13, 13, 32, 32, 36, 36, 40, 41, 58, 59, 61, 61, 64, 64, 92, 92,
		123, 123, 125, 125, 2, 0, 10, 10, 13, 13, 2, 0, 9, 9, 32, 32, 1, 0, 41,
		41, 3, 0, 9, 10, 13, 13, 32, 32, 222, 0, 2, 1, 0, 0, 0, 0, 4, 1, 0, 0,
		0, 0, 6, 1, 0, 0, 0, 0, 8, 1, 0, 0, 0, 0, 10, 1, 0, 0, 0, 0, 12, 1, 0,
		0, 0, 0, 14, 1, 0, 0, 0, 0, 16, 1, 0, 0, 0, 0, 18, 1, 0, 0, 0, 0, 20, 1,
		0, 0, 0, 0, 22, 1, 0, 0, 0, 0, 24, 1, 0, 0, 0, 0, 26, 1, 0, 0, 0, 0, 28,
		1, 0, 0, 0, 0, 30, 1, 0, 0, 0, 0, 32, 1, 0, 0, 0, 0, 34, 1, 0, 0, 0, 0,
		36, 1, 0, 0, 0, 0, 38, 1, 0, 0, 0, 0, 40, 1, 0, 0, 0, 0, 42, 1, 0, 0, 0,
		0, 44, 1, 0, 0, 0, 0, 46, 1, 0, 0, 0, 0, 48, 1, 0, 0, 0, 1, 50, 1, 0, 0,
		0, 1, 52, 1, 0, 0, 0, 1, 54, 1, 0, 0, 0, 2, 56, 1, 0, 0, 0, 4, 60, 1, 0,
		0, 0, 6, 66, 1, 0, 0, 0, 8, 71, 1, 0, 0, 0, 10, 83, 1, 0, 0, 0, 12, 85,
		1, 0, 0, 0, 14, 87, 1, 0, 0, 0, 16, 89, 1, 0, 0, 0, 18, 91, 1, 0, 0, 0,
		20, 93, 1, 0, 0, 0, 22, 95, 1, 0, 0, 0, 24, 97, 1, 0, 0, 0, 26, 99, 1,
		0, 0, 0, 28, 101, 1, 0, 0, 0, 30, 103, 1, 0, 0, 0, 32, 115, 1, 0, 0, 0,
		34, 123, 1, 0, 0, 0, 36, 126, 1, 0, 0, 0, 38, 152, 1, 0, 0, 0, 40, 154,
		1, 0, 0, 0, 42, 166, 1, 0, 0, 0, 44, 170, 1, 0, 0, 0, 46, 180, 1, 0, 0,
		0, 48, 185, 1, 0, 0, 0, 50, 192, 1, 0, 0, 0, 52, 196, 1, 0, 0, 0, 54, 201,
		1, 0, 0, 0, 56, 57, 5, 100, 0, 0, 57, 58, 5, 101, 0, 0, 58, 59, 5, 102,
		0, 0, 59, 3, 1, 0, 0, 0, 60, 61, 5, 119, 0, 0, 61, 62, 5, 97, 0, 0, 62,
		63, 5, 116, 0, 0, 63, 64, 5, 99, 0, 0, 64, 65, 5, 104, 0, 0, 65, 5, 1,
		0, 0, 0, 66, 67, 5, 115, 0, 0, 67, 68, 5, 116, 0, 0, 68, 69, 5, 111, 0,
		0, 69, 70, 5, 112, 0, 0, 70, 7, 1, 0, 0, 0, 71, 72, 5, 64, 0, 0, 72, 76,
		7, 0, 0, 0, 73, 75, 7, 1, 0, 0, 74, 73, 1, 0, 0, 0, 75, 78, 1, 0, 0, 0,
		76, 74, 1, 0, 0, 0, 76, 77, 1, 0, 0, 0, 77, 79, 1, 0, 0, 0, 78, 76, 1,
		0, 0, 0, 79, 80, 5, 40, 0, 0, 80, 81, 1, 0, 0, 0, 81, 82, 6, 3, 0, 0, 82,
		9, 1, 0, 0, 0, 83, 84, 5, 64, 0, 0, 84, 11, 1, 0, 0, 0, 85, 86, 5, 61,
		0, 0, 86, 13, 1, 0, 0, 0, 87, 88, 5, 58, 0, 0, 88, 15, 1, 0, 0, 0, 89,
		90, 5, 59, 0, 0, 90, 17, 1, 0, 0, 0, 91, 92, 5, 123, 0, 0, 92, 19, 1, 0,
		0, 0, 93, 94, 5, 125, 0, 0, 94, 21, 1, 0, 0, 0, 95, 96, 5, 40, 0, 0, 96,
		23, 1, 0, 0, 0, 97, 98, 5, 41, 0, 0, 98, 25, 1, 0, 0, 0, 99, 100, 5, 92,
		0, 0, 100, 27, 1, 0, 0, 0, 101, 102, 5, 38, 0, 0, 102, 29, 1, 0, 0, 0,
		103, 104, 5, 36, 0, 0, 104, 105, 5, 40, 0, 0, 105, 106, 1, 0, 0, 0, 106,
		110, 7, 0, 0, 0, 107, 109, 7, 1, 0, 0, 108, 107, 1, 0, 0, 0, 109, 112,
		1, 0, 0, 0, 110, 108, 1, 0, 0, 0, 110, 111, 1, 0, 0, 0, 111, 113, 1, 0,
		0, 0, 112, 110, 1, 0, 0, 0, 113, 114, 5, 41, 0, 0, 114, 31, 1, 0, 0, 0,
		115, 116, 5, 36, 0, 0, 116, 120, 7, 0, 0, 0, 117, 119, 7, 2, 0, 0, 118,
		117, 1, 0, 0, 0, 119, 122, 1, 0, 0, 0, 120, 118, 1, 0, 0, 0, 120, 121,
		1, 0, 0, 0, 121, 33, 1, 0, 0, 0, 122, 120, 1, 0, 0, 0, 123, 124, 5, 92,
		0, 0, 124, 125, 5, 36, 0, 0, 125, 35, 1, 0, 0, 0, 126, 130, 7, 0, 0, 0,
		127, 129, 7, 1, 0, 0, 128, 127, 1, 0, 0, 0, 129, 132, 1, 0, 0, 0, 130,
		128, 1, 0, 0, 0, 130, 131, 1, 0, 0, 0, 131, 37, 1, 0, 0, 0, 132, 130, 1,
		0, 0, 0, 133, 135, 7, 3, 0, 0, 134, 133, 1, 0, 0, 0, 135, 136, 1, 0, 0,
		0, 136, 134, 1, 0, 0, 0, 136, 137, 1, 0, 0, 0, 137, 144, 1, 0, 0, 0, 138,
		140, 5, 46, 0, 0, 139, 141, 7, 3, 0, 0, 140, 139, 1, 0, 0, 0, 141, 142,
		1, 0, 0, 0, 142, 140, 1, 0, 0, 0, 142, 143, 1, 0, 0, 0, 143, 145, 1, 0,
		0, 0, 144, 138, 1, 0, 0, 0, 144, 145, 1, 0, 0, 0, 145, 153, 1, 0, 0, 0,
		146, 148, 5, 46, 0, 0, 147, 149, 7, 3, 0, 0, 148, 147, 1, 0, 0, 0, 149,
		150, 1, 0, 0, 0, 150, 148, 1, 0, 0, 0, 150, 151, 1, 0, 0, 0, 151, 153,
		1, 0, 0, 0, 152, 134, 1, 0, 0, 0, 152, 146, 1, 0, 0, 0, 153, 39, 1, 0,
		0, 0, 154, 160, 5, 34, 0, 0, 155, 159, 8, 4, 0, 0, 156, 157, 5, 92, 0,
		0, 157, 159, 5, 34, 0, 0, 158, 155, 1, 0, 0, 0, 158, 156, 1, 0, 0, 0, 159,
		162, 1, 0, 0, 0, 160, 158, 1, 0, 0, 0, 160, 161, 1, 0, 0, 0, 161, 163,
		1, 0, 0, 0, 162, 160, 1, 0, 0, 0, 163, 164, 5, 34, 0, 0, 164, 41, 1, 0,
		0, 0, 165, 167, 8, 5, 0, 0, 166, 165, 1, 0, 0, 0, 167, 168, 1, 0, 0, 0,
		168, 166, 1, 0, 0, 0, 168, 169, 1, 0, 0, 0, 169, 43, 1, 0, 0, 0, 170, 174,
		5, 35, 0, 0, 171, 173, 8, 6, 0, 0, 172, 171, 1, 0, 0, 0, 173, 176, 1, 0,
		0, 0, 174, 172, 1, 0, 0, 0, 174, 175, 1, 0, 0, 0, 175, 177, 1, 0, 0, 0,
		176, 174, 1, 0, 0, 0, 177, 178, 6, 21, 1, 0, 178, 45, 1, 0, 0, 0, 179,
		181, 5, 13, 0, 0, 180, 179, 1, 0, 0, 0, 180, 181, 1, 0, 0, 0, 181, 182,
		1, 0, 0, 0, 182, 183, 5, 10, 0, 0, 183, 47, 1, 0, 0, 0, 184, 186, 7, 7,
		0, 0, 185, 184, 1, 0, 0, 0, 186, 187, 1, 0, 0, 0, 187, 185, 1, 0, 0, 0,
		187, 188, 1, 0, 0, 0, 188, 189, 1, 0, 0, 0, 189, 190, 6, 23, 1, 0, 190,
		49, 1, 0, 0, 0, 191, 193, 8, 8, 0, 0, 192, 191, 1, 0, 0, 0, 193, 194, 1,
		0, 0, 0, 194, 192, 1, 0, 0, 0, 194, 195, 1, 0, 0, 0, 195, 51, 1, 0, 0,
		0, 196, 197, 5, 41, 0, 0, 197, 198, 1, 0, 0, 0, 198, 199, 6, 25, 2, 0,
		199, 53, 1, 0, 0, 0, 200, 202, 7, 9, 0, 0, 201, 200, 1, 0, 0, 0, 202, 203,
		1, 0, 0, 0, 203, 201, 1, 0, 0, 0, 203, 204, 1, 0, 0, 0, 204, 205, 1, 0,
		0, 0, 205, 206, 6, 26, 1, 0, 206, 55, 1, 0, 0, 0, 19, 0, 1, 76, 110, 120,
		130, 136, 142, 144, 150, 152, 158, 160, 168, 174, 180, 187, 194, 203, 3,
		5, 1, 0, 0, 1, 0, 4, 0, 0,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// DevcmdLexerInit initializes any static state used to implement DevcmdLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewDevcmdLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func DevcmdLexerInit() {
	staticData := &DevcmdLexerLexerStaticData
	staticData.once.Do(devcmdlexerLexerInit)
}

// NewDevcmdLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewDevcmdLexer(input antlr.CharStream) *DevcmdLexer {
	DevcmdLexerInit()
	l := new(DevcmdLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &DevcmdLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "DevcmdLexer.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// DevcmdLexer tokens.
const (
	DevcmdLexerDEF            = 1
	DevcmdLexerWATCH          = 2
	DevcmdLexerSTOP           = 3
	DevcmdLexerAT_NAME_LPAREN = 4
	DevcmdLexerAT             = 5
	DevcmdLexerEQUALS         = 6
	DevcmdLexerCOLON          = 7
	DevcmdLexerSEMICOLON      = 8
	DevcmdLexerLBRACE         = 9
	DevcmdLexerRBRACE         = 10
	DevcmdLexerLPAREN         = 11
	DevcmdLexerRPAREN         = 12
	DevcmdLexerBACKSLASH      = 13
	DevcmdLexerAMPERSAND      = 14
	DevcmdLexerVAR_REF        = 15
	DevcmdLexerSHELL_VAR      = 16
	DevcmdLexerESCAPED_DOLLAR = 17
	DevcmdLexerNAME           = 18
	DevcmdLexerNUMBER         = 19
	DevcmdLexerSTRING         = 20
	DevcmdLexerCONTENT        = 21
	DevcmdLexerCOMMENT        = 22
	DevcmdLexerNEWLINE        = 23
	DevcmdLexerWS             = 24
	DevcmdLexerRAW_TEXT       = 25
	DevcmdLexerRAW_RPAREN     = 26
	DevcmdLexerRAW_WS         = 27
)

// DevcmdLexerRAW_CONTENT is the DevcmdLexer mode.
const DevcmdLexerRAW_CONTENT = 1
