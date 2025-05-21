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
		"ESCAPED_CHAR", "NAME", "COMMAND_TEXT", "COMMENT", "NEWLINE", "WS",
	}
	staticData.RuleNames = []string{
		"DEF", "EQUALS", "COLON", "WATCH", "STOP", "LBRACE", "RBRACE", "SEMICOLON",
		"AMPERSAND", "BACKSLASH", "OUR_VARIABLE_REFERENCE", "SHELL_VARIABLE_REFERENCE",
		"ESCAPED_CHAR", "NAME", "COMMAND_TEXT", "COMMENT", "NEWLINE", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 18, 125, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1,
		1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 4, 1, 4,
		1, 4, 1, 5, 1, 5, 1, 6, 1, 6, 1, 7, 1, 7, 1, 8, 1, 8, 1, 9, 1, 9, 1, 10,
		1, 10, 1, 10, 1, 10, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 5, 11, 76, 8, 11,
		10, 11, 12, 11, 79, 9, 11, 1, 12, 1, 12, 1, 12, 1, 12, 1, 12, 1, 12, 1,
		12, 1, 12, 1, 12, 1, 12, 3, 12, 91, 8, 12, 1, 13, 1, 13, 5, 13, 95, 8,
		13, 10, 13, 12, 13, 98, 9, 13, 1, 14, 4, 14, 101, 8, 14, 11, 14, 12, 14,
		102, 1, 15, 1, 15, 5, 15, 107, 8, 15, 10, 15, 12, 15, 110, 9, 15, 1, 15,
		1, 15, 1, 16, 3, 16, 115, 8, 16, 1, 16, 1, 16, 1, 17, 4, 17, 120, 8, 17,
		11, 17, 12, 17, 121, 1, 17, 1, 17, 0, 0, 18, 1, 1, 3, 2, 5, 3, 7, 4, 9,
		5, 11, 6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25, 13, 27, 14,
		29, 15, 31, 16, 33, 17, 35, 18, 1, 0, 8, 2, 0, 65, 90, 97, 122, 4, 0, 48,
		57, 65, 90, 95, 95, 97, 122, 10, 0, 34, 34, 36, 36, 40, 41, 59, 59, 92,
		92, 110, 110, 114, 114, 116, 116, 123, 123, 125, 125, 3, 0, 48, 57, 65,
		70, 97, 102, 5, 0, 45, 45, 48, 57, 65, 90, 95, 95, 97, 122, 9, 0, 9, 10,
		13, 13, 32, 32, 40, 41, 58, 59, 61, 61, 92, 92, 123, 123, 125, 125, 2,
		0, 10, 10, 13, 13, 2, 0, 9, 9, 32, 32, 132, 0, 1, 1, 0, 0, 0, 0, 3, 1,
		0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1,
		0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19,
		1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0,
		27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 0, 33, 1, 0, 0, 0,
		0, 35, 1, 0, 0, 0, 1, 37, 1, 0, 0, 0, 3, 41, 1, 0, 0, 0, 5, 43, 1, 0, 0,
		0, 7, 45, 1, 0, 0, 0, 9, 51, 1, 0, 0, 0, 11, 56, 1, 0, 0, 0, 13, 58, 1,
		0, 0, 0, 15, 60, 1, 0, 0, 0, 17, 62, 1, 0, 0, 0, 19, 64, 1, 0, 0, 0, 21,
		66, 1, 0, 0, 0, 23, 72, 1, 0, 0, 0, 25, 80, 1, 0, 0, 0, 27, 92, 1, 0, 0,
		0, 29, 100, 1, 0, 0, 0, 31, 104, 1, 0, 0, 0, 33, 114, 1, 0, 0, 0, 35, 119,
		1, 0, 0, 0, 37, 38, 5, 100, 0, 0, 38, 39, 5, 101, 0, 0, 39, 40, 5, 102,
		0, 0, 40, 2, 1, 0, 0, 0, 41, 42, 5, 61, 0, 0, 42, 4, 1, 0, 0, 0, 43, 44,
		5, 58, 0, 0, 44, 6, 1, 0, 0, 0, 45, 46, 5, 119, 0, 0, 46, 47, 5, 97, 0,
		0, 47, 48, 5, 116, 0, 0, 48, 49, 5, 99, 0, 0, 49, 50, 5, 104, 0, 0, 50,
		8, 1, 0, 0, 0, 51, 52, 5, 115, 0, 0, 52, 53, 5, 116, 0, 0, 53, 54, 5, 111,
		0, 0, 54, 55, 5, 112, 0, 0, 55, 10, 1, 0, 0, 0, 56, 57, 5, 123, 0, 0, 57,
		12, 1, 0, 0, 0, 58, 59, 5, 125, 0, 0, 59, 14, 1, 0, 0, 0, 60, 61, 5, 59,
		0, 0, 61, 16, 1, 0, 0, 0, 62, 63, 5, 38, 0, 0, 63, 18, 1, 0, 0, 0, 64,
		65, 5, 92, 0, 0, 65, 20, 1, 0, 0, 0, 66, 67, 5, 36, 0, 0, 67, 68, 5, 40,
		0, 0, 68, 69, 1, 0, 0, 0, 69, 70, 3, 27, 13, 0, 70, 71, 5, 41, 0, 0, 71,
		22, 1, 0, 0, 0, 72, 73, 5, 36, 0, 0, 73, 77, 7, 0, 0, 0, 74, 76, 7, 1,
		0, 0, 75, 74, 1, 0, 0, 0, 76, 79, 1, 0, 0, 0, 77, 75, 1, 0, 0, 0, 77, 78,
		1, 0, 0, 0, 78, 24, 1, 0, 0, 0, 79, 77, 1, 0, 0, 0, 80, 90, 5, 92, 0, 0,
		81, 91, 7, 2, 0, 0, 82, 83, 5, 120, 0, 0, 83, 84, 7, 3, 0, 0, 84, 91, 7,
		3, 0, 0, 85, 86, 5, 117, 0, 0, 86, 87, 7, 3, 0, 0, 87, 88, 7, 3, 0, 0,
		88, 89, 7, 3, 0, 0, 89, 91, 7, 3, 0, 0, 90, 81, 1, 0, 0, 0, 90, 82, 1,
		0, 0, 0, 90, 85, 1, 0, 0, 0, 91, 26, 1, 0, 0, 0, 92, 96, 7, 0, 0, 0, 93,
		95, 7, 4, 0, 0, 94, 93, 1, 0, 0, 0, 95, 98, 1, 0, 0, 0, 96, 94, 1, 0, 0,
		0, 96, 97, 1, 0, 0, 0, 97, 28, 1, 0, 0, 0, 98, 96, 1, 0, 0, 0, 99, 101,
		8, 5, 0, 0, 100, 99, 1, 0, 0, 0, 101, 102, 1, 0, 0, 0, 102, 100, 1, 0,
		0, 0, 102, 103, 1, 0, 0, 0, 103, 30, 1, 0, 0, 0, 104, 108, 5, 35, 0, 0,
		105, 107, 8, 6, 0, 0, 106, 105, 1, 0, 0, 0, 107, 110, 1, 0, 0, 0, 108,
		106, 1, 0, 0, 0, 108, 109, 1, 0, 0, 0, 109, 111, 1, 0, 0, 0, 110, 108,
		1, 0, 0, 0, 111, 112, 6, 15, 0, 0, 112, 32, 1, 0, 0, 0, 113, 115, 5, 13,
		0, 0, 114, 113, 1, 0, 0, 0, 114, 115, 1, 0, 0, 0, 115, 116, 1, 0, 0, 0,
		116, 117, 5, 10, 0, 0, 117, 34, 1, 0, 0, 0, 118, 120, 7, 7, 0, 0, 119,
		118, 1, 0, 0, 0, 120, 121, 1, 0, 0, 0, 121, 119, 1, 0, 0, 0, 121, 122,
		1, 0, 0, 0, 122, 123, 1, 0, 0, 0, 123, 124, 6, 17, 0, 0, 124, 36, 1, 0,
		0, 0, 8, 0, 77, 90, 96, 102, 108, 114, 121, 1, 0, 1, 0,
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
	devcmdLexerCOMMAND_TEXT             = 15
	devcmdLexerCOMMENT                  = 16
	devcmdLexerNEWLINE                  = 17
	devcmdLexerWS                       = 18
)
