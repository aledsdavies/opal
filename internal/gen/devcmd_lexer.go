// Code generated from devcmd.g4 by ANTLR 4.13.2. DO NOT EDIT.

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

type devcmdLexer struct {
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
		"DEFAULT_MODE",
	}
	staticData.LiteralNames = []string{
		"", "'def'", "'='", "':'", "'watch'", "'stop'", "'{'", "'}'", "';'",
		"'&'", "'\\'",
	}
	staticData.SymbolicNames = []string{
		"", "DEF", "EQUALS", "COLON", "WATCH", "STOP", "LBRACE", "RBRACE", "SEMICOLON",
		"AMPERSAND", "BACKSLASH", "OUR_VARIABLE_REFERENCE", "SHELL_VARIABLE_REFERENCE",
		"ESCAPED_CHAR", "NAME", "NUMBER", "COMMAND_TEXT", "COMMENT", "NEWLINE",
		"WS",
	}
	staticData.RuleNames = []string{
		"DEF", "EQUALS", "COLON", "WATCH", "STOP", "LBRACE", "RBRACE", "SEMICOLON",
		"AMPERSAND", "BACKSLASH", "OUR_VARIABLE_REFERENCE", "SHELL_VARIABLE_REFERENCE",
		"ESCAPED_CHAR", "NAME", "NUMBER", "COMMAND_TEXT", "COMMENT", "NEWLINE",
		"WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 19, 146, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 1, 0, 1, 0, 1, 0, 1, 0,
		1, 1, 1, 1, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 4, 1, 4,
		1, 4, 1, 4, 1, 4, 1, 5, 1, 5, 1, 6, 1, 6, 1, 7, 1, 7, 1, 8, 1, 8, 1, 9,
		1, 9, 1, 10, 1, 10, 1, 10, 1, 10, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 5,
		11, 78, 8, 11, 10, 11, 12, 11, 81, 9, 11, 1, 12, 1, 12, 1, 12, 1, 12, 1,
		12, 1, 12, 1, 12, 1, 12, 1, 12, 1, 12, 3, 12, 93, 8, 12, 1, 13, 1, 13,
		5, 13, 97, 8, 13, 10, 13, 12, 13, 100, 9, 13, 1, 14, 5, 14, 103, 8, 14,
		10, 14, 12, 14, 106, 9, 14, 1, 14, 1, 14, 4, 14, 110, 8, 14, 11, 14, 12,
		14, 111, 1, 14, 4, 14, 115, 8, 14, 11, 14, 12, 14, 116, 3, 14, 119, 8,
		14, 1, 15, 4, 15, 122, 8, 15, 11, 15, 12, 15, 123, 1, 16, 1, 16, 5, 16,
		128, 8, 16, 10, 16, 12, 16, 131, 9, 16, 1, 16, 1, 16, 1, 17, 3, 17, 136,
		8, 17, 1, 17, 1, 17, 1, 18, 4, 18, 141, 8, 18, 11, 18, 12, 18, 142, 1,
		18, 1, 18, 0, 0, 19, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8,
		17, 9, 19, 10, 21, 11, 23, 12, 25, 13, 27, 14, 29, 15, 31, 16, 33, 17,
		35, 18, 37, 19, 1, 0, 9, 2, 0, 65, 90, 97, 122, 4, 0, 48, 57, 65, 90, 95,
		95, 97, 122, 10, 0, 34, 34, 36, 36, 40, 41, 59, 59, 92, 92, 110, 110, 114,
		114, 116, 116, 123, 123, 125, 125, 3, 0, 48, 57, 65, 70, 97, 102, 5, 0,
		45, 45, 48, 57, 65, 90, 95, 95, 97, 122, 1, 0, 48, 57, 8, 0, 9, 10, 13,
		13, 32, 32, 58, 59, 61, 61, 92, 92, 123, 123, 125, 125, 2, 0, 10, 10, 13,
		13, 2, 0, 9, 9, 32, 32, 157, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5,
		1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13,
		1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0,
		21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0,
		0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 0, 33, 1, 0, 0, 0, 0, 35, 1, 0, 0,
		0, 0, 37, 1, 0, 0, 0, 1, 39, 1, 0, 0, 0, 3, 43, 1, 0, 0, 0, 5, 45, 1, 0,
		0, 0, 7, 47, 1, 0, 0, 0, 9, 53, 1, 0, 0, 0, 11, 58, 1, 0, 0, 0, 13, 60,
		1, 0, 0, 0, 15, 62, 1, 0, 0, 0, 17, 64, 1, 0, 0, 0, 19, 66, 1, 0, 0, 0,
		21, 68, 1, 0, 0, 0, 23, 74, 1, 0, 0, 0, 25, 82, 1, 0, 0, 0, 27, 94, 1,
		0, 0, 0, 29, 118, 1, 0, 0, 0, 31, 121, 1, 0, 0, 0, 33, 125, 1, 0, 0, 0,
		35, 135, 1, 0, 0, 0, 37, 140, 1, 0, 0, 0, 39, 40, 5, 100, 0, 0, 40, 41,
		5, 101, 0, 0, 41, 42, 5, 102, 0, 0, 42, 2, 1, 0, 0, 0, 43, 44, 5, 61, 0,
		0, 44, 4, 1, 0, 0, 0, 45, 46, 5, 58, 0, 0, 46, 6, 1, 0, 0, 0, 47, 48, 5,
		119, 0, 0, 48, 49, 5, 97, 0, 0, 49, 50, 5, 116, 0, 0, 50, 51, 5, 99, 0,
		0, 51, 52, 5, 104, 0, 0, 52, 8, 1, 0, 0, 0, 53, 54, 5, 115, 0, 0, 54, 55,
		5, 116, 0, 0, 55, 56, 5, 111, 0, 0, 56, 57, 5, 112, 0, 0, 57, 10, 1, 0,
		0, 0, 58, 59, 5, 123, 0, 0, 59, 12, 1, 0, 0, 0, 60, 61, 5, 125, 0, 0, 61,
		14, 1, 0, 0, 0, 62, 63, 5, 59, 0, 0, 63, 16, 1, 0, 0, 0, 64, 65, 5, 38,
		0, 0, 65, 18, 1, 0, 0, 0, 66, 67, 5, 92, 0, 0, 67, 20, 1, 0, 0, 0, 68,
		69, 5, 36, 0, 0, 69, 70, 5, 40, 0, 0, 70, 71, 1, 0, 0, 0, 71, 72, 3, 27,
		13, 0, 72, 73, 5, 41, 0, 0, 73, 22, 1, 0, 0, 0, 74, 75, 5, 36, 0, 0, 75,
		79, 7, 0, 0, 0, 76, 78, 7, 1, 0, 0, 77, 76, 1, 0, 0, 0, 78, 81, 1, 0, 0,
		0, 79, 77, 1, 0, 0, 0, 79, 80, 1, 0, 0, 0, 80, 24, 1, 0, 0, 0, 81, 79,
		1, 0, 0, 0, 82, 92, 5, 92, 0, 0, 83, 93, 7, 2, 0, 0, 84, 85, 5, 120, 0,
		0, 85, 86, 7, 3, 0, 0, 86, 93, 7, 3, 0, 0, 87, 88, 5, 117, 0, 0, 88, 89,
		7, 3, 0, 0, 89, 90, 7, 3, 0, 0, 90, 91, 7, 3, 0, 0, 91, 93, 7, 3, 0, 0,
		92, 83, 1, 0, 0, 0, 92, 84, 1, 0, 0, 0, 92, 87, 1, 0, 0, 0, 93, 26, 1,
		0, 0, 0, 94, 98, 7, 0, 0, 0, 95, 97, 7, 4, 0, 0, 96, 95, 1, 0, 0, 0, 97,
		100, 1, 0, 0, 0, 98, 96, 1, 0, 0, 0, 98, 99, 1, 0, 0, 0, 99, 28, 1, 0,
		0, 0, 100, 98, 1, 0, 0, 0, 101, 103, 7, 5, 0, 0, 102, 101, 1, 0, 0, 0,
		103, 106, 1, 0, 0, 0, 104, 102, 1, 0, 0, 0, 104, 105, 1, 0, 0, 0, 105,
		107, 1, 0, 0, 0, 106, 104, 1, 0, 0, 0, 107, 109, 5, 46, 0, 0, 108, 110,
		7, 5, 0, 0, 109, 108, 1, 0, 0, 0, 110, 111, 1, 0, 0, 0, 111, 109, 1, 0,
		0, 0, 111, 112, 1, 0, 0, 0, 112, 119, 1, 0, 0, 0, 113, 115, 7, 5, 0, 0,
		114, 113, 1, 0, 0, 0, 115, 116, 1, 0, 0, 0, 116, 114, 1, 0, 0, 0, 116,
		117, 1, 0, 0, 0, 117, 119, 1, 0, 0, 0, 118, 104, 1, 0, 0, 0, 118, 114,
		1, 0, 0, 0, 119, 30, 1, 0, 0, 0, 120, 122, 8, 6, 0, 0, 121, 120, 1, 0,
		0, 0, 122, 123, 1, 0, 0, 0, 123, 121, 1, 0, 0, 0, 123, 124, 1, 0, 0, 0,
		124, 32, 1, 0, 0, 0, 125, 129, 5, 35, 0, 0, 126, 128, 8, 7, 0, 0, 127,
		126, 1, 0, 0, 0, 128, 131, 1, 0, 0, 0, 129, 127, 1, 0, 0, 0, 129, 130,
		1, 0, 0, 0, 130, 132, 1, 0, 0, 0, 131, 129, 1, 0, 0, 0, 132, 133, 6, 16,
		0, 0, 133, 34, 1, 0, 0, 0, 134, 136, 5, 13, 0, 0, 135, 134, 1, 0, 0, 0,
		135, 136, 1, 0, 0, 0, 136, 137, 1, 0, 0, 0, 137, 138, 5, 10, 0, 0, 138,
		36, 1, 0, 0, 0, 139, 141, 7, 8, 0, 0, 140, 139, 1, 0, 0, 0, 141, 142, 1,
		0, 0, 0, 142, 140, 1, 0, 0, 0, 142, 143, 1, 0, 0, 0, 143, 144, 1, 0, 0,
		0, 144, 145, 6, 18, 0, 0, 145, 38, 1, 0, 0, 0, 12, 0, 79, 92, 98, 104,
		111, 116, 118, 123, 129, 135, 142, 1, 0, 1, 0,
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

// devcmdLexerInit initializes any static state used to implement devcmdLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewdevcmdLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func DevcmdLexerInit() {
	staticData := &DevcmdLexerLexerStaticData
	staticData.once.Do(devcmdlexerLexerInit)
}

// NewdevcmdLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewdevcmdLexer(input antlr.CharStream) *devcmdLexer {
	DevcmdLexerInit()
	l := new(devcmdLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &DevcmdLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "devcmd.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// devcmdLexer tokens.
const (
	devcmdLexerDEF                      = 1
	devcmdLexerEQUALS                   = 2
	devcmdLexerCOLON                    = 3
	devcmdLexerWATCH                    = 4
	devcmdLexerSTOP                     = 5
	devcmdLexerLBRACE                   = 6
	devcmdLexerRBRACE                   = 7
	devcmdLexerSEMICOLON                = 8
	devcmdLexerAMPERSAND                = 9
	devcmdLexerBACKSLASH                = 10
	devcmdLexerOUR_VARIABLE_REFERENCE   = 11
	devcmdLexerSHELL_VARIABLE_REFERENCE = 12
	devcmdLexerESCAPED_CHAR             = 13
	devcmdLexerNAME                     = 14
	devcmdLexerNUMBER                   = 15
	devcmdLexerCOMMAND_TEXT             = 16
	devcmdLexerCOMMENT                  = 17
	devcmdLexerNEWLINE                  = 18
	devcmdLexerWS                       = 19
)
