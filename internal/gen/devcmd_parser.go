// Code generated from DevcmdParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen // DevcmdParser
import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr4-go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type DevcmdParser struct {
	*antlr.BaseParser
}

var DevcmdParserParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func devcmdparserParserInit() {
	staticData := &DevcmdParserParserStaticData
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
		"program", "line", "variableDefinition", "variableValue", "commandDefinition",
		"commandBody", "annotatedCommand", "annotation", "simpleCommand", "annotationCommand",
		"blockCommand", "blockStatements", "nonEmptyBlockStatements", "blockStatement",
		"continuationLine", "commandText", "commandTextElement",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 27, 167, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 1, 0, 5, 0, 36, 8, 0, 10, 0, 12, 0, 39, 9, 0, 1, 0, 1, 0,
		1, 1, 1, 1, 1, 1, 3, 1, 46, 8, 1, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1,
		3, 1, 3, 3, 3, 56, 8, 3, 1, 4, 3, 4, 59, 8, 4, 1, 4, 1, 4, 1, 4, 1, 4,
		1, 5, 1, 5, 1, 5, 3, 5, 68, 8, 5, 1, 6, 1, 6, 5, 6, 72, 8, 6, 10, 6, 12,
		6, 75, 9, 6, 1, 6, 1, 6, 3, 6, 79, 8, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6,
		1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 3, 6, 91, 8, 6, 1, 7, 1, 7, 1, 8, 1, 8, 5,
		8, 97, 8, 8, 10, 8, 12, 8, 100, 9, 8, 1, 8, 1, 8, 1, 9, 1, 9, 5, 9, 106,
		8, 9, 10, 9, 12, 9, 109, 9, 9, 1, 10, 1, 10, 3, 10, 113, 8, 10, 1, 10,
		1, 10, 1, 10, 1, 11, 1, 11, 3, 11, 120, 8, 11, 1, 12, 1, 12, 1, 12, 5,
		12, 125, 8, 12, 10, 12, 12, 12, 128, 9, 12, 1, 12, 5, 12, 131, 8, 12, 10,
		12, 12, 12, 134, 9, 12, 1, 12, 3, 12, 137, 8, 12, 1, 12, 5, 12, 140, 8,
		12, 10, 12, 12, 12, 143, 9, 12, 1, 13, 1, 13, 1, 13, 5, 13, 148, 8, 13,
		10, 13, 12, 13, 151, 9, 13, 3, 13, 153, 8, 13, 1, 14, 1, 14, 1, 14, 1,
		14, 1, 15, 5, 15, 160, 8, 15, 10, 15, 12, 15, 163, 9, 15, 1, 16, 1, 16,
		1, 16, 0, 0, 17, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28,
		30, 32, 0, 2, 1, 0, 2, 3, 3, 0, 2, 3, 6, 7, 9, 21, 171, 0, 37, 1, 0, 0,
		0, 2, 45, 1, 0, 0, 0, 4, 47, 1, 0, 0, 0, 6, 55, 1, 0, 0, 0, 8, 58, 1, 0,
		0, 0, 10, 67, 1, 0, 0, 0, 12, 90, 1, 0, 0, 0, 14, 92, 1, 0, 0, 0, 16, 94,
		1, 0, 0, 0, 18, 103, 1, 0, 0, 0, 20, 110, 1, 0, 0, 0, 22, 119, 1, 0, 0,
		0, 24, 121, 1, 0, 0, 0, 26, 152, 1, 0, 0, 0, 28, 154, 1, 0, 0, 0, 30, 161,
		1, 0, 0, 0, 32, 164, 1, 0, 0, 0, 34, 36, 3, 2, 1, 0, 35, 34, 1, 0, 0, 0,
		36, 39, 1, 0, 0, 0, 37, 35, 1, 0, 0, 0, 37, 38, 1, 0, 0, 0, 38, 40, 1,
		0, 0, 0, 39, 37, 1, 0, 0, 0, 40, 41, 5, 0, 0, 1, 41, 1, 1, 0, 0, 0, 42,
		46, 3, 4, 2, 0, 43, 46, 3, 8, 4, 0, 44, 46, 5, 23, 0, 0, 45, 42, 1, 0,
		0, 0, 45, 43, 1, 0, 0, 0, 45, 44, 1, 0, 0, 0, 46, 3, 1, 0, 0, 0, 47, 48,
		5, 1, 0, 0, 48, 49, 5, 18, 0, 0, 49, 50, 5, 6, 0, 0, 50, 51, 3, 6, 3, 0,
		51, 52, 5, 8, 0, 0, 52, 5, 1, 0, 0, 0, 53, 56, 3, 30, 15, 0, 54, 56, 1,
		0, 0, 0, 55, 53, 1, 0, 0, 0, 55, 54, 1, 0, 0, 0, 56, 7, 1, 0, 0, 0, 57,
		59, 7, 0, 0, 0, 58, 57, 1, 0, 0, 0, 58, 59, 1, 0, 0, 0, 59, 60, 1, 0, 0,
		0, 60, 61, 5, 18, 0, 0, 61, 62, 5, 7, 0, 0, 62, 63, 3, 10, 5, 0, 63, 9,
		1, 0, 0, 0, 64, 68, 3, 12, 6, 0, 65, 68, 3, 20, 10, 0, 66, 68, 3, 16, 8,
		0, 67, 64, 1, 0, 0, 0, 67, 65, 1, 0, 0, 0, 67, 66, 1, 0, 0, 0, 68, 11,
		1, 0, 0, 0, 69, 73, 5, 4, 0, 0, 70, 72, 5, 25, 0, 0, 71, 70, 1, 0, 0, 0,
		72, 75, 1, 0, 0, 0, 73, 71, 1, 0, 0, 0, 73, 74, 1, 0, 0, 0, 74, 76, 1,
		0, 0, 0, 75, 73, 1, 0, 0, 0, 76, 78, 5, 26, 0, 0, 77, 79, 5, 8, 0, 0, 78,
		77, 1, 0, 0, 0, 78, 79, 1, 0, 0, 0, 79, 91, 1, 0, 0, 0, 80, 81, 5, 5, 0,
		0, 81, 82, 3, 14, 7, 0, 82, 83, 5, 7, 0, 0, 83, 84, 3, 20, 10, 0, 84, 91,
		1, 0, 0, 0, 85, 86, 5, 5, 0, 0, 86, 87, 3, 14, 7, 0, 87, 88, 5, 7, 0, 0,
		88, 89, 3, 18, 9, 0, 89, 91, 1, 0, 0, 0, 90, 69, 1, 0, 0, 0, 90, 80, 1,
		0, 0, 0, 90, 85, 1, 0, 0, 0, 91, 13, 1, 0, 0, 0, 92, 93, 5, 18, 0, 0, 93,
		15, 1, 0, 0, 0, 94, 98, 3, 30, 15, 0, 95, 97, 3, 28, 14, 0, 96, 95, 1,
		0, 0, 0, 97, 100, 1, 0, 0, 0, 98, 96, 1, 0, 0, 0, 98, 99, 1, 0, 0, 0, 99,
		101, 1, 0, 0, 0, 100, 98, 1, 0, 0, 0, 101, 102, 5, 8, 0, 0, 102, 17, 1,
		0, 0, 0, 103, 107, 3, 30, 15, 0, 104, 106, 3, 28, 14, 0, 105, 104, 1, 0,
		0, 0, 106, 109, 1, 0, 0, 0, 107, 105, 1, 0, 0, 0, 107, 108, 1, 0, 0, 0,
		108, 19, 1, 0, 0, 0, 109, 107, 1, 0, 0, 0, 110, 112, 5, 9, 0, 0, 111, 113,
		5, 23, 0, 0, 112, 111, 1, 0, 0, 0, 112, 113, 1, 0, 0, 0, 113, 114, 1, 0,
		0, 0, 114, 115, 3, 22, 11, 0, 115, 116, 5, 10, 0, 0, 116, 21, 1, 0, 0,
		0, 117, 120, 1, 0, 0, 0, 118, 120, 3, 24, 12, 0, 119, 117, 1, 0, 0, 0,
		119, 118, 1, 0, 0, 0, 120, 23, 1, 0, 0, 0, 121, 132, 3, 26, 13, 0, 122,
		126, 5, 8, 0, 0, 123, 125, 5, 23, 0, 0, 124, 123, 1, 0, 0, 0, 125, 128,
		1, 0, 0, 0, 126, 124, 1, 0, 0, 0, 126, 127, 1, 0, 0, 0, 127, 129, 1, 0,
		0, 0, 128, 126, 1, 0, 0, 0, 129, 131, 3, 26, 13, 0, 130, 122, 1, 0, 0,
		0, 131, 134, 1, 0, 0, 0, 132, 130, 1, 0, 0, 0, 132, 133, 1, 0, 0, 0, 133,
		136, 1, 0, 0, 0, 134, 132, 1, 0, 0, 0, 135, 137, 5, 8, 0, 0, 136, 135,
		1, 0, 0, 0, 136, 137, 1, 0, 0, 0, 137, 141, 1, 0, 0, 0, 138, 140, 5, 23,
		0, 0, 139, 138, 1, 0, 0, 0, 140, 143, 1, 0, 0, 0, 141, 139, 1, 0, 0, 0,
		141, 142, 1, 0, 0, 0, 142, 25, 1, 0, 0, 0, 143, 141, 1, 0, 0, 0, 144, 153,
		3, 12, 6, 0, 145, 149, 3, 30, 15, 0, 146, 148, 3, 28, 14, 0, 147, 146,
		1, 0, 0, 0, 148, 151, 1, 0, 0, 0, 149, 147, 1, 0, 0, 0, 149, 150, 1, 0,
		0, 0, 150, 153, 1, 0, 0, 0, 151, 149, 1, 0, 0, 0, 152, 144, 1, 0, 0, 0,
		152, 145, 1, 0, 0, 0, 153, 27, 1, 0, 0, 0, 154, 155, 5, 13, 0, 0, 155,
		156, 5, 23, 0, 0, 156, 157, 3, 30, 15, 0, 157, 29, 1, 0, 0, 0, 158, 160,
		3, 32, 16, 0, 159, 158, 1, 0, 0, 0, 160, 163, 1, 0, 0, 0, 161, 159, 1,
		0, 0, 0, 161, 162, 1, 0, 0, 0, 162, 31, 1, 0, 0, 0, 163, 161, 1, 0, 0,
		0, 164, 165, 7, 1, 0, 0, 165, 33, 1, 0, 0, 0, 19, 37, 45, 55, 58, 67, 73,
		78, 90, 98, 107, 112, 119, 126, 132, 136, 141, 149, 152, 161,
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

// DevcmdParserInit initializes any static state used to implement DevcmdParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewDevcmdParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func DevcmdParserInit() {
	staticData := &DevcmdParserParserStaticData
	staticData.once.Do(devcmdparserParserInit)
}

// NewDevcmdParser produces a new parser instance for the optional input antlr.TokenStream.
func NewDevcmdParser(input antlr.TokenStream) *DevcmdParser {
	DevcmdParserInit()
	this := new(DevcmdParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &DevcmdParserParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "DevcmdParser.g4"

	return this
}

// DevcmdParser tokens.
const (
	DevcmdParserEOF            = antlr.TokenEOF
	DevcmdParserDEF            = 1
	DevcmdParserWATCH          = 2
	DevcmdParserSTOP           = 3
	DevcmdParserAT_NAME_LPAREN = 4
	DevcmdParserAT             = 5
	DevcmdParserEQUALS         = 6
	DevcmdParserCOLON          = 7
	DevcmdParserSEMICOLON      = 8
	DevcmdParserLBRACE         = 9
	DevcmdParserRBRACE         = 10
	DevcmdParserLPAREN         = 11
	DevcmdParserRPAREN         = 12
	DevcmdParserBACKSLASH      = 13
	DevcmdParserAMPERSAND      = 14
	DevcmdParserVAR_REF        = 15
	DevcmdParserSHELL_VAR      = 16
	DevcmdParserESCAPED_DOLLAR = 17
	DevcmdParserNAME           = 18
	DevcmdParserNUMBER         = 19
	DevcmdParserSTRING         = 20
	DevcmdParserCONTENT        = 21
	DevcmdParserCOMMENT        = 22
	DevcmdParserNEWLINE        = 23
	DevcmdParserWS             = 24
	DevcmdParserRAW_TEXT       = 25
	DevcmdParserRAW_RPAREN     = 26
	DevcmdParserRAW_WS         = 27
)

// DevcmdParser rules.
const (
	DevcmdParserRULE_program                 = 0
	DevcmdParserRULE_line                    = 1
	DevcmdParserRULE_variableDefinition      = 2
	DevcmdParserRULE_variableValue           = 3
	DevcmdParserRULE_commandDefinition       = 4
	DevcmdParserRULE_commandBody             = 5
	DevcmdParserRULE_annotatedCommand        = 6
	DevcmdParserRULE_annotation              = 7
	DevcmdParserRULE_simpleCommand           = 8
	DevcmdParserRULE_annotationCommand       = 9
	DevcmdParserRULE_blockCommand            = 10
	DevcmdParserRULE_blockStatements         = 11
	DevcmdParserRULE_nonEmptyBlockStatements = 12
	DevcmdParserRULE_blockStatement          = 13
	DevcmdParserRULE_continuationLine        = 14
	DevcmdParserRULE_commandText             = 15
	DevcmdParserRULE_commandTextElement      = 16
)

// IProgramContext is an interface to support dynamic dispatch.
type IProgramContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EOF() antlr.TerminalNode
	AllLine() []ILineContext
	Line(i int) ILineContext

	// IsProgramContext differentiates from other interfaces.
	IsProgramContext()
}

type ProgramContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyProgramContext() *ProgramContext {
	var p = new(ProgramContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_program
	return p
}

func InitEmptyProgramContext(p *ProgramContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_program
}

func (*ProgramContext) IsProgramContext() {}

func NewProgramContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ProgramContext {
	var p = new(ProgramContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_program

	return p
}

func (s *ProgramContext) GetParser() antlr.Parser { return s.parser }

func (s *ProgramContext) EOF() antlr.TerminalNode {
	return s.GetToken(DevcmdParserEOF, 0)
}

func (s *ProgramContext) AllLine() []ILineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ILineContext); ok {
			len++
		}
	}

	tst := make([]ILineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ILineContext); ok {
			tst[i] = t.(ILineContext)
			i++
		}
	}

	return tst
}

func (s *ProgramContext) Line(i int) ILineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILineContext)
}

func (s *ProgramContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ProgramContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ProgramContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterProgram(s)
	}
}

func (s *ProgramContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitProgram(s)
	}
}

func (p *DevcmdParser) Program() (localctx IProgramContext) {
	localctx = NewProgramContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, DevcmdParserRULE_program)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(37)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8650766) != 0 {
		{
			p.SetState(34)
			p.Line()
		}

		p.SetState(39)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(40)
		p.Match(DevcmdParserEOF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ILineContext is an interface to support dynamic dispatch.
type ILineContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	VariableDefinition() IVariableDefinitionContext
	CommandDefinition() ICommandDefinitionContext
	NEWLINE() antlr.TerminalNode

	// IsLineContext differentiates from other interfaces.
	IsLineContext()
}

type LineContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLineContext() *LineContext {
	var p = new(LineContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_line
	return p
}

func InitEmptyLineContext(p *LineContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_line
}

func (*LineContext) IsLineContext() {}

func NewLineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LineContext {
	var p = new(LineContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_line

	return p
}

func (s *LineContext) GetParser() antlr.Parser { return s.parser }

func (s *LineContext) VariableDefinition() IVariableDefinitionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariableDefinitionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariableDefinitionContext)
}

func (s *LineContext) CommandDefinition() ICommandDefinitionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandDefinitionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandDefinitionContext)
}

func (s *LineContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNEWLINE, 0)
}

func (s *LineContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LineContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LineContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterLine(s)
	}
}

func (s *LineContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitLine(s)
	}
}

func (p *DevcmdParser) Line() (localctx ILineContext) {
	localctx = NewLineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, DevcmdParserRULE_line)
	p.SetState(45)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case DevcmdParserDEF:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(42)
			p.VariableDefinition()
		}

	case DevcmdParserWATCH, DevcmdParserSTOP, DevcmdParserNAME:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(43)
			p.CommandDefinition()
		}

	case DevcmdParserNEWLINE:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(44)
			p.Match(DevcmdParserNEWLINE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IVariableDefinitionContext is an interface to support dynamic dispatch.
type IVariableDefinitionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	DEF() antlr.TerminalNode
	NAME() antlr.TerminalNode
	EQUALS() antlr.TerminalNode
	VariableValue() IVariableValueContext
	SEMICOLON() antlr.TerminalNode

	// IsVariableDefinitionContext differentiates from other interfaces.
	IsVariableDefinitionContext()
}

type VariableDefinitionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariableDefinitionContext() *VariableDefinitionContext {
	var p = new(VariableDefinitionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_variableDefinition
	return p
}

func InitEmptyVariableDefinitionContext(p *VariableDefinitionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_variableDefinition
}

func (*VariableDefinitionContext) IsVariableDefinitionContext() {}

func NewVariableDefinitionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableDefinitionContext {
	var p = new(VariableDefinitionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_variableDefinition

	return p
}

func (s *VariableDefinitionContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableDefinitionContext) DEF() antlr.TerminalNode {
	return s.GetToken(DevcmdParserDEF, 0)
}

func (s *VariableDefinitionContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *VariableDefinitionContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(DevcmdParserEQUALS, 0)
}

func (s *VariableDefinitionContext) VariableValue() IVariableValueContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariableValueContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariableValueContext)
}

func (s *VariableDefinitionContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSEMICOLON, 0)
}

func (s *VariableDefinitionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableDefinitionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableDefinitionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterVariableDefinition(s)
	}
}

func (s *VariableDefinitionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitVariableDefinition(s)
	}
}

func (p *DevcmdParser) VariableDefinition() (localctx IVariableDefinitionContext) {
	localctx = NewVariableDefinitionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, DevcmdParserRULE_variableDefinition)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(47)
		p.Match(DevcmdParserDEF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(48)
		p.Match(DevcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(49)
		p.Match(DevcmdParserEQUALS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(50)
		p.VariableValue()
	}
	{
		p.SetState(51)
		p.Match(DevcmdParserSEMICOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IVariableValueContext is an interface to support dynamic dispatch.
type IVariableValueContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	CommandText() ICommandTextContext

	// IsVariableValueContext differentiates from other interfaces.
	IsVariableValueContext()
}

type VariableValueContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariableValueContext() *VariableValueContext {
	var p = new(VariableValueContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_variableValue
	return p
}

func InitEmptyVariableValueContext(p *VariableValueContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_variableValue
}

func (*VariableValueContext) IsVariableValueContext() {}

func NewVariableValueContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableValueContext {
	var p = new(VariableValueContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_variableValue

	return p
}

func (s *VariableValueContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableValueContext) CommandText() ICommandTextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextContext)
}

func (s *VariableValueContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableValueContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableValueContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterVariableValue(s)
	}
}

func (s *VariableValueContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitVariableValue(s)
	}
}

func (p *DevcmdParser) VariableValue() (localctx IVariableValueContext) {
	localctx = NewVariableValueContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, DevcmdParserRULE_variableValue)
	p.SetState(55)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 2, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(53)
			p.CommandText()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ICommandDefinitionContext is an interface to support dynamic dispatch.
type ICommandDefinitionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NAME() antlr.TerminalNode
	COLON() antlr.TerminalNode
	CommandBody() ICommandBodyContext
	WATCH() antlr.TerminalNode
	STOP() antlr.TerminalNode

	// IsCommandDefinitionContext differentiates from other interfaces.
	IsCommandDefinitionContext()
}

type CommandDefinitionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCommandDefinitionContext() *CommandDefinitionContext {
	var p = new(CommandDefinitionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandDefinition
	return p
}

func InitEmptyCommandDefinitionContext(p *CommandDefinitionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandDefinition
}

func (*CommandDefinitionContext) IsCommandDefinitionContext() {}

func NewCommandDefinitionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandDefinitionContext {
	var p = new(CommandDefinitionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_commandDefinition

	return p
}

func (s *CommandDefinitionContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandDefinitionContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *CommandDefinitionContext) COLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOLON, 0)
}

func (s *CommandDefinitionContext) CommandBody() ICommandBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandBodyContext)
}

func (s *CommandDefinitionContext) WATCH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserWATCH, 0)
}

func (s *CommandDefinitionContext) STOP() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSTOP, 0)
}

func (s *CommandDefinitionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandDefinitionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandDefinitionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterCommandDefinition(s)
	}
}

func (s *CommandDefinitionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitCommandDefinition(s)
	}
}

func (p *DevcmdParser) CommandDefinition() (localctx ICommandDefinitionContext) {
	localctx = NewCommandDefinitionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, DevcmdParserRULE_commandDefinition)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(58)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == DevcmdParserWATCH || _la == DevcmdParserSTOP {
		{
			p.SetState(57)
			_la = p.GetTokenStream().LA(1)

			if !(_la == DevcmdParserWATCH || _la == DevcmdParserSTOP) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}

	}
	{
		p.SetState(60)
		p.Match(DevcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(61)
		p.Match(DevcmdParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(62)
		p.CommandBody()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ICommandBodyContext is an interface to support dynamic dispatch.
type ICommandBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AnnotatedCommand() IAnnotatedCommandContext
	BlockCommand() IBlockCommandContext
	SimpleCommand() ISimpleCommandContext

	// IsCommandBodyContext differentiates from other interfaces.
	IsCommandBodyContext()
}

type CommandBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCommandBodyContext() *CommandBodyContext {
	var p = new(CommandBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandBody
	return p
}

func InitEmptyCommandBodyContext(p *CommandBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandBody
}

func (*CommandBodyContext) IsCommandBodyContext() {}

func NewCommandBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandBodyContext {
	var p = new(CommandBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_commandBody

	return p
}

func (s *CommandBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandBodyContext) AnnotatedCommand() IAnnotatedCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAnnotatedCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAnnotatedCommandContext)
}

func (s *CommandBodyContext) BlockCommand() IBlockCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockCommandContext)
}

func (s *CommandBodyContext) SimpleCommand() ISimpleCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISimpleCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISimpleCommandContext)
}

func (s *CommandBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterCommandBody(s)
	}
}

func (s *CommandBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitCommandBody(s)
	}
}

func (p *DevcmdParser) CommandBody() (localctx ICommandBodyContext) {
	localctx = NewCommandBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, DevcmdParserRULE_commandBody)
	p.SetState(67)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 4, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(64)
			p.AnnotatedCommand()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(65)
			p.BlockCommand()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(66)
			p.SimpleCommand()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IAnnotatedCommandContext is an interface to support dynamic dispatch.
type IAnnotatedCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsAnnotatedCommandContext differentiates from other interfaces.
	IsAnnotatedCommandContext()
}

type AnnotatedCommandContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAnnotatedCommandContext() *AnnotatedCommandContext {
	var p = new(AnnotatedCommandContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_annotatedCommand
	return p
}

func InitEmptyAnnotatedCommandContext(p *AnnotatedCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_annotatedCommand
}

func (*AnnotatedCommandContext) IsAnnotatedCommandContext() {}

func NewAnnotatedCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AnnotatedCommandContext {
	var p = new(AnnotatedCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_annotatedCommand

	return p
}

func (s *AnnotatedCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *AnnotatedCommandContext) CopyAll(ctx *AnnotatedCommandContext) {
	s.CopyFrom(&ctx.BaseParserRuleContext)
}

func (s *AnnotatedCommandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AnnotatedCommandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type FunctionAnnotContext struct {
	AnnotatedCommandContext
}

func NewFunctionAnnotContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *FunctionAnnotContext {
	var p = new(FunctionAnnotContext)

	InitEmptyAnnotatedCommandContext(&p.AnnotatedCommandContext)
	p.parser = parser
	p.CopyAll(ctx.(*AnnotatedCommandContext))

	return p
}

func (s *FunctionAnnotContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FunctionAnnotContext) AT_NAME_LPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAT_NAME_LPAREN, 0)
}

func (s *FunctionAnnotContext) RAW_RPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRAW_RPAREN, 0)
}

func (s *FunctionAnnotContext) AllRAW_TEXT() []antlr.TerminalNode {
	return s.GetTokens(DevcmdParserRAW_TEXT)
}

func (s *FunctionAnnotContext) RAW_TEXT(i int) antlr.TerminalNode {
	return s.GetToken(DevcmdParserRAW_TEXT, i)
}

func (s *FunctionAnnotContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSEMICOLON, 0)
}

func (s *FunctionAnnotContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterFunctionAnnot(s)
	}
}

func (s *FunctionAnnotContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitFunctionAnnot(s)
	}
}

type SimpleAnnotContext struct {
	AnnotatedCommandContext
}

func NewSimpleAnnotContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *SimpleAnnotContext {
	var p = new(SimpleAnnotContext)

	InitEmptyAnnotatedCommandContext(&p.AnnotatedCommandContext)
	p.parser = parser
	p.CopyAll(ctx.(*AnnotatedCommandContext))

	return p
}

func (s *SimpleAnnotContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SimpleAnnotContext) AT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAT, 0)
}

func (s *SimpleAnnotContext) Annotation() IAnnotationContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAnnotationContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAnnotationContext)
}

func (s *SimpleAnnotContext) COLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOLON, 0)
}

func (s *SimpleAnnotContext) AnnotationCommand() IAnnotationCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAnnotationCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAnnotationCommandContext)
}

func (s *SimpleAnnotContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterSimpleAnnot(s)
	}
}

func (s *SimpleAnnotContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitSimpleAnnot(s)
	}
}

type BlockAnnotContext struct {
	AnnotatedCommandContext
}

func NewBlockAnnotContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *BlockAnnotContext {
	var p = new(BlockAnnotContext)

	InitEmptyAnnotatedCommandContext(&p.AnnotatedCommandContext)
	p.parser = parser
	p.CopyAll(ctx.(*AnnotatedCommandContext))

	return p
}

func (s *BlockAnnotContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockAnnotContext) AT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAT, 0)
}

func (s *BlockAnnotContext) Annotation() IAnnotationContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAnnotationContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAnnotationContext)
}

func (s *BlockAnnotContext) COLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOLON, 0)
}

func (s *BlockAnnotContext) BlockCommand() IBlockCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockCommandContext)
}

func (s *BlockAnnotContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterBlockAnnot(s)
	}
}

func (s *BlockAnnotContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitBlockAnnot(s)
	}
}

func (p *DevcmdParser) AnnotatedCommand() (localctx IAnnotatedCommandContext) {
	localctx = NewAnnotatedCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, DevcmdParserRULE_annotatedCommand)
	var _la int

	p.SetState(90)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 7, p.GetParserRuleContext()) {
	case 1:
		localctx = NewFunctionAnnotContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(69)
			p.Match(DevcmdParserAT_NAME_LPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(73)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == DevcmdParserRAW_TEXT {
			{
				p.SetState(70)
				p.Match(DevcmdParserRAW_TEXT)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

			p.SetState(75)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(76)
			p.Match(DevcmdParserRAW_RPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(78)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 6, p.GetParserRuleContext()) == 1 {
			{
				p.SetState(77)
				p.Match(DevcmdParserSEMICOLON)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		} else if p.HasError() { // JIM
			goto errorExit
		}

	case 2:
		localctx = NewBlockAnnotContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(80)
			p.Match(DevcmdParserAT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(81)
			p.Annotation()
		}
		{
			p.SetState(82)
			p.Match(DevcmdParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(83)
			p.BlockCommand()
		}

	case 3:
		localctx = NewSimpleAnnotContext(p, localctx)
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(85)
			p.Match(DevcmdParserAT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(86)
			p.Annotation()
		}
		{
			p.SetState(87)
			p.Match(DevcmdParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(88)
			p.AnnotationCommand()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IAnnotationContext is an interface to support dynamic dispatch.
type IAnnotationContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NAME() antlr.TerminalNode

	// IsAnnotationContext differentiates from other interfaces.
	IsAnnotationContext()
}

type AnnotationContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAnnotationContext() *AnnotationContext {
	var p = new(AnnotationContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_annotation
	return p
}

func InitEmptyAnnotationContext(p *AnnotationContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_annotation
}

func (*AnnotationContext) IsAnnotationContext() {}

func NewAnnotationContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AnnotationContext {
	var p = new(AnnotationContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_annotation

	return p
}

func (s *AnnotationContext) GetParser() antlr.Parser { return s.parser }

func (s *AnnotationContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *AnnotationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AnnotationContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AnnotationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterAnnotation(s)
	}
}

func (s *AnnotationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitAnnotation(s)
	}
}

func (p *DevcmdParser) Annotation() (localctx IAnnotationContext) {
	localctx = NewAnnotationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, DevcmdParserRULE_annotation)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(92)
		p.Match(DevcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISimpleCommandContext is an interface to support dynamic dispatch.
type ISimpleCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	CommandText() ICommandTextContext
	SEMICOLON() antlr.TerminalNode
	AllContinuationLine() []IContinuationLineContext
	ContinuationLine(i int) IContinuationLineContext

	// IsSimpleCommandContext differentiates from other interfaces.
	IsSimpleCommandContext()
}

type SimpleCommandContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySimpleCommandContext() *SimpleCommandContext {
	var p = new(SimpleCommandContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_simpleCommand
	return p
}

func InitEmptySimpleCommandContext(p *SimpleCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_simpleCommand
}

func (*SimpleCommandContext) IsSimpleCommandContext() {}

func NewSimpleCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SimpleCommandContext {
	var p = new(SimpleCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_simpleCommand

	return p
}

func (s *SimpleCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *SimpleCommandContext) CommandText() ICommandTextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextContext)
}

func (s *SimpleCommandContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSEMICOLON, 0)
}

func (s *SimpleCommandContext) AllContinuationLine() []IContinuationLineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IContinuationLineContext); ok {
			len++
		}
	}

	tst := make([]IContinuationLineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IContinuationLineContext); ok {
			tst[i] = t.(IContinuationLineContext)
			i++
		}
	}

	return tst
}

func (s *SimpleCommandContext) ContinuationLine(i int) IContinuationLineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IContinuationLineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IContinuationLineContext)
}

func (s *SimpleCommandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SimpleCommandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SimpleCommandContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterSimpleCommand(s)
	}
}

func (s *SimpleCommandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitSimpleCommand(s)
	}
}

func (p *DevcmdParser) SimpleCommand() (localctx ISimpleCommandContext) {
	localctx = NewSimpleCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, DevcmdParserRULE_simpleCommand)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(94)
		p.CommandText()
	}
	p.SetState(98)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == DevcmdParserBACKSLASH {
		{
			p.SetState(95)
			p.ContinuationLine()
		}

		p.SetState(100)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(101)
		p.Match(DevcmdParserSEMICOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IAnnotationCommandContext is an interface to support dynamic dispatch.
type IAnnotationCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	CommandText() ICommandTextContext
	AllContinuationLine() []IContinuationLineContext
	ContinuationLine(i int) IContinuationLineContext

	// IsAnnotationCommandContext differentiates from other interfaces.
	IsAnnotationCommandContext()
}

type AnnotationCommandContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAnnotationCommandContext() *AnnotationCommandContext {
	var p = new(AnnotationCommandContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_annotationCommand
	return p
}

func InitEmptyAnnotationCommandContext(p *AnnotationCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_annotationCommand
}

func (*AnnotationCommandContext) IsAnnotationCommandContext() {}

func NewAnnotationCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AnnotationCommandContext {
	var p = new(AnnotationCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_annotationCommand

	return p
}

func (s *AnnotationCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *AnnotationCommandContext) CommandText() ICommandTextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextContext)
}

func (s *AnnotationCommandContext) AllContinuationLine() []IContinuationLineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IContinuationLineContext); ok {
			len++
		}
	}

	tst := make([]IContinuationLineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IContinuationLineContext); ok {
			tst[i] = t.(IContinuationLineContext)
			i++
		}
	}

	return tst
}

func (s *AnnotationCommandContext) ContinuationLine(i int) IContinuationLineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IContinuationLineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IContinuationLineContext)
}

func (s *AnnotationCommandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AnnotationCommandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AnnotationCommandContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterAnnotationCommand(s)
	}
}

func (s *AnnotationCommandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitAnnotationCommand(s)
	}
}

func (p *DevcmdParser) AnnotationCommand() (localctx IAnnotationCommandContext) {
	localctx = NewAnnotationCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, DevcmdParserRULE_annotationCommand)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(103)
		p.CommandText()
	}
	p.SetState(107)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == DevcmdParserBACKSLASH {
		{
			p.SetState(104)
			p.ContinuationLine()
		}

		p.SetState(109)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockCommandContext is an interface to support dynamic dispatch.
type IBlockCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	BlockStatements() IBlockStatementsContext
	RBRACE() antlr.TerminalNode
	NEWLINE() antlr.TerminalNode

	// IsBlockCommandContext differentiates from other interfaces.
	IsBlockCommandContext()
}

type BlockCommandContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockCommandContext() *BlockCommandContext {
	var p = new(BlockCommandContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockCommand
	return p
}

func InitEmptyBlockCommandContext(p *BlockCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockCommand
}

func (*BlockCommandContext) IsBlockCommandContext() {}

func NewBlockCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockCommandContext {
	var p = new(BlockCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_blockCommand

	return p
}

func (s *BlockCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockCommandContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLBRACE, 0)
}

func (s *BlockCommandContext) BlockStatements() IBlockStatementsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockStatementsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockStatementsContext)
}

func (s *BlockCommandContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRBRACE, 0)
}

func (s *BlockCommandContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNEWLINE, 0)
}

func (s *BlockCommandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockCommandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockCommandContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterBlockCommand(s)
	}
}

func (s *BlockCommandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitBlockCommand(s)
	}
}

func (p *DevcmdParser) BlockCommand() (localctx IBlockCommandContext) {
	localctx = NewBlockCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, DevcmdParserRULE_blockCommand)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(110)
		p.Match(DevcmdParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(112)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 10, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(111)
			p.Match(DevcmdParserNEWLINE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	} else if p.HasError() { // JIM
		goto errorExit
	}
	{
		p.SetState(114)
		p.BlockStatements()
	}
	{
		p.SetState(115)
		p.Match(DevcmdParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockStatementsContext is an interface to support dynamic dispatch.
type IBlockStatementsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NonEmptyBlockStatements() INonEmptyBlockStatementsContext

	// IsBlockStatementsContext differentiates from other interfaces.
	IsBlockStatementsContext()
}

type BlockStatementsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockStatementsContext() *BlockStatementsContext {
	var p = new(BlockStatementsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockStatements
	return p
}

func InitEmptyBlockStatementsContext(p *BlockStatementsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockStatements
}

func (*BlockStatementsContext) IsBlockStatementsContext() {}

func NewBlockStatementsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockStatementsContext {
	var p = new(BlockStatementsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_blockStatements

	return p
}

func (s *BlockStatementsContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockStatementsContext) NonEmptyBlockStatements() INonEmptyBlockStatementsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INonEmptyBlockStatementsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INonEmptyBlockStatementsContext)
}

func (s *BlockStatementsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockStatementsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockStatementsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterBlockStatements(s)
	}
}

func (s *BlockStatementsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitBlockStatements(s)
	}
}

func (p *DevcmdParser) BlockStatements() (localctx IBlockStatementsContext) {
	localctx = NewBlockStatementsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, DevcmdParserRULE_blockStatements)
	p.SetState(119)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 11, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(118)
			p.NonEmptyBlockStatements()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// INonEmptyBlockStatementsContext is an interface to support dynamic dispatch.
type INonEmptyBlockStatementsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllBlockStatement() []IBlockStatementContext
	BlockStatement(i int) IBlockStatementContext
	AllSEMICOLON() []antlr.TerminalNode
	SEMICOLON(i int) antlr.TerminalNode
	AllNEWLINE() []antlr.TerminalNode
	NEWLINE(i int) antlr.TerminalNode

	// IsNonEmptyBlockStatementsContext differentiates from other interfaces.
	IsNonEmptyBlockStatementsContext()
}

type NonEmptyBlockStatementsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNonEmptyBlockStatementsContext() *NonEmptyBlockStatementsContext {
	var p = new(NonEmptyBlockStatementsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_nonEmptyBlockStatements
	return p
}

func InitEmptyNonEmptyBlockStatementsContext(p *NonEmptyBlockStatementsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_nonEmptyBlockStatements
}

func (*NonEmptyBlockStatementsContext) IsNonEmptyBlockStatementsContext() {}

func NewNonEmptyBlockStatementsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NonEmptyBlockStatementsContext {
	var p = new(NonEmptyBlockStatementsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_nonEmptyBlockStatements

	return p
}

func (s *NonEmptyBlockStatementsContext) GetParser() antlr.Parser { return s.parser }

func (s *NonEmptyBlockStatementsContext) AllBlockStatement() []IBlockStatementContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IBlockStatementContext); ok {
			len++
		}
	}

	tst := make([]IBlockStatementContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IBlockStatementContext); ok {
			tst[i] = t.(IBlockStatementContext)
			i++
		}
	}

	return tst
}

func (s *NonEmptyBlockStatementsContext) BlockStatement(i int) IBlockStatementContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockStatementContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockStatementContext)
}

func (s *NonEmptyBlockStatementsContext) AllSEMICOLON() []antlr.TerminalNode {
	return s.GetTokens(DevcmdParserSEMICOLON)
}

func (s *NonEmptyBlockStatementsContext) SEMICOLON(i int) antlr.TerminalNode {
	return s.GetToken(DevcmdParserSEMICOLON, i)
}

func (s *NonEmptyBlockStatementsContext) AllNEWLINE() []antlr.TerminalNode {
	return s.GetTokens(DevcmdParserNEWLINE)
}

func (s *NonEmptyBlockStatementsContext) NEWLINE(i int) antlr.TerminalNode {
	return s.GetToken(DevcmdParserNEWLINE, i)
}

func (s *NonEmptyBlockStatementsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NonEmptyBlockStatementsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NonEmptyBlockStatementsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterNonEmptyBlockStatements(s)
	}
}

func (s *NonEmptyBlockStatementsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitNonEmptyBlockStatements(s)
	}
}

func (p *DevcmdParser) NonEmptyBlockStatements() (localctx INonEmptyBlockStatementsContext) {
	localctx = NewNonEmptyBlockStatementsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, DevcmdParserRULE_nonEmptyBlockStatements)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(121)
		p.BlockStatement()
	}
	p.SetState(132)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 13, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(122)
				p.Match(DevcmdParserSEMICOLON)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			p.SetState(126)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
			for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
				if _alt == 1 {
					{
						p.SetState(123)
						p.Match(DevcmdParserNEWLINE)
						if p.HasError() {
							// Recognition error - abort rule
							goto errorExit
						}
					}

				}
				p.SetState(128)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext())
				if p.HasError() {
					goto errorExit
				}
			}
			{
				p.SetState(129)
				p.BlockStatement()
			}

		}
		p.SetState(134)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 13, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(136)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == DevcmdParserSEMICOLON {
		{
			p.SetState(135)
			p.Match(DevcmdParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	p.SetState(141)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == DevcmdParserNEWLINE {
		{
			p.SetState(138)
			p.Match(DevcmdParserNEWLINE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(143)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockStatementContext is an interface to support dynamic dispatch.
type IBlockStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AnnotatedCommand() IAnnotatedCommandContext
	CommandText() ICommandTextContext
	AllContinuationLine() []IContinuationLineContext
	ContinuationLine(i int) IContinuationLineContext

	// IsBlockStatementContext differentiates from other interfaces.
	IsBlockStatementContext()
}

type BlockStatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockStatementContext() *BlockStatementContext {
	var p = new(BlockStatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockStatement
	return p
}

func InitEmptyBlockStatementContext(p *BlockStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_blockStatement
}

func (*BlockStatementContext) IsBlockStatementContext() {}

func NewBlockStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockStatementContext {
	var p = new(BlockStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_blockStatement

	return p
}

func (s *BlockStatementContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockStatementContext) AnnotatedCommand() IAnnotatedCommandContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAnnotatedCommandContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAnnotatedCommandContext)
}

func (s *BlockStatementContext) CommandText() ICommandTextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextContext)
}

func (s *BlockStatementContext) AllContinuationLine() []IContinuationLineContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IContinuationLineContext); ok {
			len++
		}
	}

	tst := make([]IContinuationLineContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IContinuationLineContext); ok {
			tst[i] = t.(IContinuationLineContext)
			i++
		}
	}

	return tst
}

func (s *BlockStatementContext) ContinuationLine(i int) IContinuationLineContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IContinuationLineContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IContinuationLineContext)
}

func (s *BlockStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterBlockStatement(s)
	}
}

func (s *BlockStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitBlockStatement(s)
	}
}

func (p *DevcmdParser) BlockStatement() (localctx IBlockStatementContext) {
	localctx = NewBlockStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 26, DevcmdParserRULE_blockStatement)
	var _la int

	p.SetState(152)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case DevcmdParserAT_NAME_LPAREN, DevcmdParserAT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(144)
			p.AnnotatedCommand()
		}

	case DevcmdParserWATCH, DevcmdParserSTOP, DevcmdParserEQUALS, DevcmdParserCOLON, DevcmdParserSEMICOLON, DevcmdParserLBRACE, DevcmdParserRBRACE, DevcmdParserLPAREN, DevcmdParserRPAREN, DevcmdParserBACKSLASH, DevcmdParserAMPERSAND, DevcmdParserVAR_REF, DevcmdParserSHELL_VAR, DevcmdParserESCAPED_DOLLAR, DevcmdParserNAME, DevcmdParserNUMBER, DevcmdParserSTRING, DevcmdParserCONTENT, DevcmdParserNEWLINE:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(145)
			p.CommandText()
		}
		p.SetState(149)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == DevcmdParserBACKSLASH {
			{
				p.SetState(146)
				p.ContinuationLine()
			}

			p.SetState(151)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IContinuationLineContext is an interface to support dynamic dispatch.
type IContinuationLineContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	BACKSLASH() antlr.TerminalNode
	NEWLINE() antlr.TerminalNode
	CommandText() ICommandTextContext

	// IsContinuationLineContext differentiates from other interfaces.
	IsContinuationLineContext()
}

type ContinuationLineContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyContinuationLineContext() *ContinuationLineContext {
	var p = new(ContinuationLineContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_continuationLine
	return p
}

func InitEmptyContinuationLineContext(p *ContinuationLineContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_continuationLine
}

func (*ContinuationLineContext) IsContinuationLineContext() {}

func NewContinuationLineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ContinuationLineContext {
	var p = new(ContinuationLineContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_continuationLine

	return p
}

func (s *ContinuationLineContext) GetParser() antlr.Parser { return s.parser }

func (s *ContinuationLineContext) BACKSLASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserBACKSLASH, 0)
}

func (s *ContinuationLineContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNEWLINE, 0)
}

func (s *ContinuationLineContext) CommandText() ICommandTextContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextContext)
}

func (s *ContinuationLineContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ContinuationLineContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ContinuationLineContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterContinuationLine(s)
	}
}

func (s *ContinuationLineContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitContinuationLine(s)
	}
}

func (p *DevcmdParser) ContinuationLine() (localctx IContinuationLineContext) {
	localctx = NewContinuationLineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, DevcmdParserRULE_continuationLine)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(154)
		p.Match(DevcmdParserBACKSLASH)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(155)
		p.Match(DevcmdParserNEWLINE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(156)
		p.CommandText()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ICommandTextContext is an interface to support dynamic dispatch.
type ICommandTextContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllCommandTextElement() []ICommandTextElementContext
	CommandTextElement(i int) ICommandTextElementContext

	// IsCommandTextContext differentiates from other interfaces.
	IsCommandTextContext()
}

type CommandTextContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCommandTextContext() *CommandTextContext {
	var p = new(CommandTextContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandText
	return p
}

func InitEmptyCommandTextContext(p *CommandTextContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandText
}

func (*CommandTextContext) IsCommandTextContext() {}

func NewCommandTextContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandTextContext {
	var p = new(CommandTextContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_commandText

	return p
}

func (s *CommandTextContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandTextContext) AllCommandTextElement() []ICommandTextElementContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ICommandTextElementContext); ok {
			len++
		}
	}

	tst := make([]ICommandTextElementContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ICommandTextElementContext); ok {
			tst[i] = t.(ICommandTextElementContext)
			i++
		}
	}

	return tst
}

func (s *CommandTextContext) CommandTextElement(i int) ICommandTextElementContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICommandTextElementContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICommandTextElementContext)
}

func (s *CommandTextContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandTextContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandTextContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterCommandText(s)
	}
}

func (s *CommandTextContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitCommandText(s)
	}
}

func (p *DevcmdParser) CommandText() (localctx ICommandTextContext) {
	localctx = NewCommandTextContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 30, DevcmdParserRULE_commandText)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(161)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 18, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(158)
				p.CommandTextElement()
			}

		}
		p.SetState(163)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 18, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ICommandTextElementContext is an interface to support dynamic dispatch.
type ICommandTextElementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	VAR_REF() antlr.TerminalNode
	SHELL_VAR() antlr.TerminalNode
	ESCAPED_DOLLAR() antlr.TerminalNode
	NAME() antlr.TerminalNode
	NUMBER() antlr.TerminalNode
	STRING() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AMPERSAND() antlr.TerminalNode
	COLON() antlr.TerminalNode
	EQUALS() antlr.TerminalNode
	BACKSLASH() antlr.TerminalNode
	WATCH() antlr.TerminalNode
	STOP() antlr.TerminalNode
	CONTENT() antlr.TerminalNode

	// IsCommandTextElementContext differentiates from other interfaces.
	IsCommandTextElementContext()
}

type CommandTextElementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCommandTextElementContext() *CommandTextElementContext {
	var p = new(CommandTextElementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandTextElement
	return p
}

func InitEmptyCommandTextElementContext(p *CommandTextElementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = DevcmdParserRULE_commandTextElement
}

func (*CommandTextElementContext) IsCommandTextElementContext() {}

func NewCommandTextElementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandTextElementContext {
	var p = new(CommandTextElementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = DevcmdParserRULE_commandTextElement

	return p
}

func (s *CommandTextElementContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandTextElementContext) VAR_REF() antlr.TerminalNode {
	return s.GetToken(DevcmdParserVAR_REF, 0)
}

func (s *CommandTextElementContext) SHELL_VAR() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSHELL_VAR, 0)
}

func (s *CommandTextElementContext) ESCAPED_DOLLAR() antlr.TerminalNode {
	return s.GetToken(DevcmdParserESCAPED_DOLLAR, 0)
}

func (s *CommandTextElementContext) NAME() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNAME, 0)
}

func (s *CommandTextElementContext) NUMBER() antlr.TerminalNode {
	return s.GetToken(DevcmdParserNUMBER, 0)
}

func (s *CommandTextElementContext) STRING() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSTRING, 0)
}

func (s *CommandTextElementContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLPAREN, 0)
}

func (s *CommandTextElementContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRPAREN, 0)
}

func (s *CommandTextElementContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserLBRACE, 0)
}

func (s *CommandTextElementContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(DevcmdParserRBRACE, 0)
}

func (s *CommandTextElementContext) AMPERSAND() antlr.TerminalNode {
	return s.GetToken(DevcmdParserAMPERSAND, 0)
}

func (s *CommandTextElementContext) COLON() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCOLON, 0)
}

func (s *CommandTextElementContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(DevcmdParserEQUALS, 0)
}

func (s *CommandTextElementContext) BACKSLASH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserBACKSLASH, 0)
}

func (s *CommandTextElementContext) WATCH() antlr.TerminalNode {
	return s.GetToken(DevcmdParserWATCH, 0)
}

func (s *CommandTextElementContext) STOP() antlr.TerminalNode {
	return s.GetToken(DevcmdParserSTOP, 0)
}

func (s *CommandTextElementContext) CONTENT() antlr.TerminalNode {
	return s.GetToken(DevcmdParserCONTENT, 0)
}

func (s *CommandTextElementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandTextElementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandTextElementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.EnterCommandTextElement(s)
	}
}

func (s *CommandTextElementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(DevcmdParserListener); ok {
		listenerT.ExitCommandTextElement(s)
	}
}

func (p *DevcmdParser) CommandTextElement() (localctx ICommandTextElementContext) {
	localctx = NewCommandTextElementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 32, DevcmdParserRULE_commandTextElement)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(164)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&4193996) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}
