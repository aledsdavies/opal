// Code generated from devcmd.g4 by ANTLR 4.13.2. DO NOT EDIT.

package gen // devcmd
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

type devcmdParser struct {
	*antlr.BaseParser
}

var DevcmdParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func devcmdParserInit() {
	staticData := &DevcmdParserStaticData
	staticData.LiteralNames = []string{
		"", "'def'", "'='", "':'", "'watch'", "'stop'", "'{'", "'}'", "';'",
		"'&'", "'\\'", "'$('", "')'",
	}
	staticData.SymbolicNames = []string{
		"", "DEF", "EQUALS", "COLON", "WATCH", "STOP", "LBRACE", "RBRACE", "SEMICOLON",
		"AMPERSAND", "BACKSLASH", "VAR_START", "VAR_END", "INCOMPLETE_VARIABLE_REFERENCE",
		"ESCAPED_CHAR", "NAME", "COMMAND_TEXT", "COMMENT", "NEWLINE", "WS",
	}
	staticData.RuleNames = []string{
		"program", "line", "variableDefinition", "commandDefinition", "simpleCommand",
		"blockCommand", "blockStatements", "nonEmptyBlockStatements", "blockStatement",
		"continuationLine", "commandText", "variableReference",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 19, 118, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 1, 0, 5, 0, 26, 8, 0, 10, 0, 12, 0, 29, 9, 0, 1, 0, 1,
		0, 1, 1, 1, 1, 1, 1, 3, 1, 36, 8, 1, 1, 2, 1, 2, 1, 2, 1, 2, 3, 2, 42,
		8, 2, 3, 2, 44, 8, 2, 1, 2, 1, 2, 1, 3, 3, 3, 49, 8, 3, 1, 3, 1, 3, 1,
		3, 1, 3, 3, 3, 55, 8, 3, 1, 4, 1, 4, 5, 4, 59, 8, 4, 10, 4, 12, 4, 62,
		9, 4, 3, 4, 64, 8, 4, 1, 4, 1, 4, 1, 5, 1, 5, 3, 5, 70, 8, 5, 1, 5, 1,
		5, 1, 5, 3, 5, 75, 8, 5, 1, 6, 1, 6, 3, 6, 79, 8, 6, 1, 7, 1, 7, 1, 7,
		3, 7, 84, 8, 7, 1, 7, 5, 7, 87, 8, 7, 10, 7, 12, 7, 90, 9, 7, 1, 7, 3,
		7, 93, 8, 7, 1, 8, 1, 8, 3, 8, 97, 8, 8, 1, 9, 1, 9, 1, 9, 1, 9, 1, 10,
		1, 10, 1, 10, 1, 10, 1, 10, 1, 10, 1, 10, 4, 10, 110, 8, 10, 11, 10, 12,
		10, 111, 1, 11, 1, 11, 1, 11, 1, 11, 1, 11, 0, 0, 12, 0, 2, 4, 6, 8, 10,
		12, 14, 16, 18, 20, 22, 0, 2, 1, 1, 18, 18, 1, 0, 4, 5, 128, 0, 27, 1,
		0, 0, 0, 2, 35, 1, 0, 0, 0, 4, 37, 1, 0, 0, 0, 6, 48, 1, 0, 0, 0, 8, 63,
		1, 0, 0, 0, 10, 67, 1, 0, 0, 0, 12, 78, 1, 0, 0, 0, 14, 80, 1, 0, 0, 0,
		16, 94, 1, 0, 0, 0, 18, 98, 1, 0, 0, 0, 20, 109, 1, 0, 0, 0, 22, 113, 1,
		0, 0, 0, 24, 26, 3, 2, 1, 0, 25, 24, 1, 0, 0, 0, 26, 29, 1, 0, 0, 0, 27,
		25, 1, 0, 0, 0, 27, 28, 1, 0, 0, 0, 28, 30, 1, 0, 0, 0, 29, 27, 1, 0, 0,
		0, 30, 31, 5, 0, 0, 1, 31, 1, 1, 0, 0, 0, 32, 36, 3, 4, 2, 0, 33, 36, 3,
		6, 3, 0, 34, 36, 5, 18, 0, 0, 35, 32, 1, 0, 0, 0, 35, 33, 1, 0, 0, 0, 35,
		34, 1, 0, 0, 0, 36, 3, 1, 0, 0, 0, 37, 38, 5, 1, 0, 0, 38, 43, 5, 15, 0,
		0, 39, 41, 5, 2, 0, 0, 40, 42, 3, 20, 10, 0, 41, 40, 1, 0, 0, 0, 41, 42,
		1, 0, 0, 0, 42, 44, 1, 0, 0, 0, 43, 39, 1, 0, 0, 0, 43, 44, 1, 0, 0, 0,
		44, 45, 1, 0, 0, 0, 45, 46, 7, 0, 0, 0, 46, 5, 1, 0, 0, 0, 47, 49, 7, 1,
		0, 0, 48, 47, 1, 0, 0, 0, 48, 49, 1, 0, 0, 0, 49, 50, 1, 0, 0, 0, 50, 51,
		5, 15, 0, 0, 51, 54, 5, 3, 0, 0, 52, 55, 3, 8, 4, 0, 53, 55, 3, 10, 5,
		0, 54, 52, 1, 0, 0, 0, 54, 53, 1, 0, 0, 0, 55, 7, 1, 0, 0, 0, 56, 60, 3,
		20, 10, 0, 57, 59, 3, 18, 9, 0, 58, 57, 1, 0, 0, 0, 59, 62, 1, 0, 0, 0,
		60, 58, 1, 0, 0, 0, 60, 61, 1, 0, 0, 0, 61, 64, 1, 0, 0, 0, 62, 60, 1,
		0, 0, 0, 63, 56, 1, 0, 0, 0, 63, 64, 1, 0, 0, 0, 64, 65, 1, 0, 0, 0, 65,
		66, 7, 0, 0, 0, 66, 9, 1, 0, 0, 0, 67, 69, 5, 6, 0, 0, 68, 70, 5, 18, 0,
		0, 69, 68, 1, 0, 0, 0, 69, 70, 1, 0, 0, 0, 70, 71, 1, 0, 0, 0, 71, 72,
		3, 12, 6, 0, 72, 74, 5, 7, 0, 0, 73, 75, 7, 0, 0, 0, 74, 73, 1, 0, 0, 0,
		74, 75, 1, 0, 0, 0, 75, 11, 1, 0, 0, 0, 76, 79, 1, 0, 0, 0, 77, 79, 3,
		14, 7, 0, 78, 76, 1, 0, 0, 0, 78, 77, 1, 0, 0, 0, 79, 13, 1, 0, 0, 0, 80,
		88, 3, 16, 8, 0, 81, 83, 5, 8, 0, 0, 82, 84, 5, 18, 0, 0, 83, 82, 1, 0,
		0, 0, 83, 84, 1, 0, 0, 0, 84, 85, 1, 0, 0, 0, 85, 87, 3, 16, 8, 0, 86,
		81, 1, 0, 0, 0, 87, 90, 1, 0, 0, 0, 88, 86, 1, 0, 0, 0, 88, 89, 1, 0, 0,
		0, 89, 92, 1, 0, 0, 0, 90, 88, 1, 0, 0, 0, 91, 93, 5, 8, 0, 0, 92, 91,
		1, 0, 0, 0, 92, 93, 1, 0, 0, 0, 93, 15, 1, 0, 0, 0, 94, 96, 3, 20, 10,
		0, 95, 97, 5, 9, 0, 0, 96, 95, 1, 0, 0, 0, 96, 97, 1, 0, 0, 0, 97, 17,
		1, 0, 0, 0, 98, 99, 5, 10, 0, 0, 99, 100, 5, 18, 0, 0, 100, 101, 3, 20,
		10, 0, 101, 19, 1, 0, 0, 0, 102, 110, 5, 14, 0, 0, 103, 110, 3, 22, 11,
		0, 104, 110, 5, 13, 0, 0, 105, 110, 5, 3, 0, 0, 106, 110, 5, 2, 0, 0, 107,
		110, 5, 16, 0, 0, 108, 110, 5, 15, 0, 0, 109, 102, 1, 0, 0, 0, 109, 103,
		1, 0, 0, 0, 109, 104, 1, 0, 0, 0, 109, 105, 1, 0, 0, 0, 109, 106, 1, 0,
		0, 0, 109, 107, 1, 0, 0, 0, 109, 108, 1, 0, 0, 0, 110, 111, 1, 0, 0, 0,
		111, 109, 1, 0, 0, 0, 111, 112, 1, 0, 0, 0, 112, 21, 1, 0, 0, 0, 113, 114,
		5, 11, 0, 0, 114, 115, 5, 15, 0, 0, 115, 116, 5, 12, 0, 0, 116, 23, 1,
		0, 0, 0, 17, 27, 35, 41, 43, 48, 54, 60, 63, 69, 74, 78, 83, 88, 92, 96,
		109, 111,
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

// devcmdParserInit initializes any static state used to implement devcmdParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewdevcmdParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func DevcmdParserInit() {
	staticData := &DevcmdParserStaticData
	staticData.once.Do(devcmdParserInit)
}

// NewdevcmdParser produces a new parser instance for the optional input antlr.TokenStream.
func NewdevcmdParser(input antlr.TokenStream) *devcmdParser {
	DevcmdParserInit()
	this := new(devcmdParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &DevcmdParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "devcmd.g4"

	return this
}

// devcmdParser tokens.
const (
	devcmdParserEOF                           = antlr.TokenEOF
	devcmdParserDEF                           = 1
	devcmdParserEQUALS                        = 2
	devcmdParserCOLON                         = 3
	devcmdParserWATCH                         = 4
	devcmdParserSTOP                          = 5
	devcmdParserLBRACE                        = 6
	devcmdParserRBRACE                        = 7
	devcmdParserSEMICOLON                     = 8
	devcmdParserAMPERSAND                     = 9
	devcmdParserBACKSLASH                     = 10
	devcmdParserVAR_START                     = 11
	devcmdParserVAR_END                       = 12
	devcmdParserINCOMPLETE_VARIABLE_REFERENCE = 13
	devcmdParserESCAPED_CHAR                  = 14
	devcmdParserNAME                          = 15
	devcmdParserCOMMAND_TEXT                  = 16
	devcmdParserCOMMENT                       = 17
	devcmdParserNEWLINE                       = 18
	devcmdParserWS                            = 19
)

// devcmdParser rules.
const (
	devcmdParserRULE_program                 = 0
	devcmdParserRULE_line                    = 1
	devcmdParserRULE_variableDefinition      = 2
	devcmdParserRULE_commandDefinition       = 3
	devcmdParserRULE_simpleCommand           = 4
	devcmdParserRULE_blockCommand            = 5
	devcmdParserRULE_blockStatements         = 6
	devcmdParserRULE_nonEmptyBlockStatements = 7
	devcmdParserRULE_blockStatement          = 8
	devcmdParserRULE_continuationLine        = 9
	devcmdParserRULE_commandText             = 10
	devcmdParserRULE_variableReference       = 11
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
	p.RuleIndex = devcmdParserRULE_program
	return p
}

func InitEmptyProgramContext(p *ProgramContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_program
}

func (*ProgramContext) IsProgramContext() {}

func NewProgramContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ProgramContext {
	var p = new(ProgramContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_program

	return p
}

func (s *ProgramContext) GetParser() antlr.Parser { return s.parser }

func (s *ProgramContext) EOF() antlr.TerminalNode {
	return s.GetToken(devcmdParserEOF, 0)
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
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterProgram(s)
	}
}

func (s *ProgramContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitProgram(s)
	}
}

func (p *devcmdParser) Program() (localctx IProgramContext) {
	localctx = NewProgramContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, devcmdParserRULE_program)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(27)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&294962) != 0 {
		{
			p.SetState(24)
			p.Line()
		}

		p.SetState(29)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(30)
		p.Match(devcmdParserEOF)
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
	p.RuleIndex = devcmdParserRULE_line
	return p
}

func InitEmptyLineContext(p *LineContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_line
}

func (*LineContext) IsLineContext() {}

func NewLineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LineContext {
	var p = new(LineContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_line

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
	return s.GetToken(devcmdParserNEWLINE, 0)
}

func (s *LineContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LineContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LineContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterLine(s)
	}
}

func (s *LineContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitLine(s)
	}
}

func (p *devcmdParser) Line() (localctx ILineContext) {
	localctx = NewLineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, devcmdParserRULE_line)
	p.SetState(35)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case devcmdParserDEF:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(32)
			p.VariableDefinition()
		}

	case devcmdParserWATCH, devcmdParserSTOP, devcmdParserNAME:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(33)
			p.CommandDefinition()
		}

	case devcmdParserNEWLINE:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(34)
			p.Match(devcmdParserNEWLINE)
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
	NEWLINE() antlr.TerminalNode
	EOF() antlr.TerminalNode
	EQUALS() antlr.TerminalNode
	CommandText() ICommandTextContext

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
	p.RuleIndex = devcmdParserRULE_variableDefinition
	return p
}

func InitEmptyVariableDefinitionContext(p *VariableDefinitionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_variableDefinition
}

func (*VariableDefinitionContext) IsVariableDefinitionContext() {}

func NewVariableDefinitionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableDefinitionContext {
	var p = new(VariableDefinitionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_variableDefinition

	return p
}

func (s *VariableDefinitionContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableDefinitionContext) DEF() antlr.TerminalNode {
	return s.GetToken(devcmdParserDEF, 0)
}

func (s *VariableDefinitionContext) NAME() antlr.TerminalNode {
	return s.GetToken(devcmdParserNAME, 0)
}

func (s *VariableDefinitionContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(devcmdParserNEWLINE, 0)
}

func (s *VariableDefinitionContext) EOF() antlr.TerminalNode {
	return s.GetToken(devcmdParserEOF, 0)
}

func (s *VariableDefinitionContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(devcmdParserEQUALS, 0)
}

func (s *VariableDefinitionContext) CommandText() ICommandTextContext {
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

func (s *VariableDefinitionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableDefinitionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableDefinitionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterVariableDefinition(s)
	}
}

func (s *VariableDefinitionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitVariableDefinition(s)
	}
}

func (p *devcmdParser) VariableDefinition() (localctx IVariableDefinitionContext) {
	localctx = NewVariableDefinitionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, devcmdParserRULE_variableDefinition)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(37)
		p.Match(devcmdParserDEF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(38)
		p.Match(devcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(43)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == devcmdParserEQUALS {
		{
			p.SetState(39)
			p.Match(devcmdParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(41)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&124940) != 0 {
			{
				p.SetState(40)
				p.CommandText()
			}

		}

	}
	{
		p.SetState(45)
		_la = p.GetTokenStream().LA(1)

		if !(_la == devcmdParserEOF || _la == devcmdParserNEWLINE) {
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

// ICommandDefinitionContext is an interface to support dynamic dispatch.
type ICommandDefinitionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NAME() antlr.TerminalNode
	COLON() antlr.TerminalNode
	SimpleCommand() ISimpleCommandContext
	BlockCommand() IBlockCommandContext
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
	p.RuleIndex = devcmdParserRULE_commandDefinition
	return p
}

func InitEmptyCommandDefinitionContext(p *CommandDefinitionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_commandDefinition
}

func (*CommandDefinitionContext) IsCommandDefinitionContext() {}

func NewCommandDefinitionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandDefinitionContext {
	var p = new(CommandDefinitionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_commandDefinition

	return p
}

func (s *CommandDefinitionContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandDefinitionContext) NAME() antlr.TerminalNode {
	return s.GetToken(devcmdParserNAME, 0)
}

func (s *CommandDefinitionContext) COLON() antlr.TerminalNode {
	return s.GetToken(devcmdParserCOLON, 0)
}

func (s *CommandDefinitionContext) SimpleCommand() ISimpleCommandContext {
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

func (s *CommandDefinitionContext) BlockCommand() IBlockCommandContext {
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

func (s *CommandDefinitionContext) WATCH() antlr.TerminalNode {
	return s.GetToken(devcmdParserWATCH, 0)
}

func (s *CommandDefinitionContext) STOP() antlr.TerminalNode {
	return s.GetToken(devcmdParserSTOP, 0)
}

func (s *CommandDefinitionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandDefinitionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandDefinitionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterCommandDefinition(s)
	}
}

func (s *CommandDefinitionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitCommandDefinition(s)
	}
}

func (p *devcmdParser) CommandDefinition() (localctx ICommandDefinitionContext) {
	localctx = NewCommandDefinitionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, devcmdParserRULE_commandDefinition)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(48)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == devcmdParserWATCH || _la == devcmdParserSTOP {
		{
			p.SetState(47)
			_la = p.GetTokenStream().LA(1)

			if !(_la == devcmdParserWATCH || _la == devcmdParserSTOP) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}

	}
	{
		p.SetState(50)
		p.Match(devcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(51)
		p.Match(devcmdParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(54)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case devcmdParserEOF, devcmdParserEQUALS, devcmdParserCOLON, devcmdParserVAR_START, devcmdParserINCOMPLETE_VARIABLE_REFERENCE, devcmdParserESCAPED_CHAR, devcmdParserNAME, devcmdParserCOMMAND_TEXT, devcmdParserNEWLINE:
		{
			p.SetState(52)
			p.SimpleCommand()
		}

	case devcmdParserLBRACE:
		{
			p.SetState(53)
			p.BlockCommand()
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

// ISimpleCommandContext is an interface to support dynamic dispatch.
type ISimpleCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NEWLINE() antlr.TerminalNode
	EOF() antlr.TerminalNode
	CommandText() ICommandTextContext
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
	p.RuleIndex = devcmdParserRULE_simpleCommand
	return p
}

func InitEmptySimpleCommandContext(p *SimpleCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_simpleCommand
}

func (*SimpleCommandContext) IsSimpleCommandContext() {}

func NewSimpleCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SimpleCommandContext {
	var p = new(SimpleCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_simpleCommand

	return p
}

func (s *SimpleCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *SimpleCommandContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(devcmdParserNEWLINE, 0)
}

func (s *SimpleCommandContext) EOF() antlr.TerminalNode {
	return s.GetToken(devcmdParserEOF, 0)
}

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
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterSimpleCommand(s)
	}
}

func (s *SimpleCommandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitSimpleCommand(s)
	}
}

func (p *devcmdParser) SimpleCommand() (localctx ISimpleCommandContext) {
	localctx = NewSimpleCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, devcmdParserRULE_simpleCommand)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(63)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&124940) != 0 {
		{
			p.SetState(56)
			p.CommandText()
		}
		p.SetState(60)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == devcmdParserBACKSLASH {
			{
				p.SetState(57)
				p.ContinuationLine()
			}

			p.SetState(62)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

	}
	{
		p.SetState(65)
		_la = p.GetTokenStream().LA(1)

		if !(_la == devcmdParserEOF || _la == devcmdParserNEWLINE) {
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

// IBlockCommandContext is an interface to support dynamic dispatch.
type IBlockCommandContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	BlockStatements() IBlockStatementsContext
	RBRACE() antlr.TerminalNode
	AllNEWLINE() []antlr.TerminalNode
	NEWLINE(i int) antlr.TerminalNode
	EOF() antlr.TerminalNode

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
	p.RuleIndex = devcmdParserRULE_blockCommand
	return p
}

func InitEmptyBlockCommandContext(p *BlockCommandContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_blockCommand
}

func (*BlockCommandContext) IsBlockCommandContext() {}

func NewBlockCommandContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockCommandContext {
	var p = new(BlockCommandContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_blockCommand

	return p
}

func (s *BlockCommandContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockCommandContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(devcmdParserLBRACE, 0)
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
	return s.GetToken(devcmdParserRBRACE, 0)
}

func (s *BlockCommandContext) AllNEWLINE() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserNEWLINE)
}

func (s *BlockCommandContext) NEWLINE(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserNEWLINE, i)
}

func (s *BlockCommandContext) EOF() antlr.TerminalNode {
	return s.GetToken(devcmdParserEOF, 0)
}

func (s *BlockCommandContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockCommandContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockCommandContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterBlockCommand(s)
	}
}

func (s *BlockCommandContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitBlockCommand(s)
	}
}

func (p *devcmdParser) BlockCommand() (localctx IBlockCommandContext) {
	localctx = NewBlockCommandContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, devcmdParserRULE_blockCommand)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(67)
		p.Match(devcmdParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(69)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == devcmdParserNEWLINE {
		{
			p.SetState(68)
			p.Match(devcmdParserNEWLINE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(71)
		p.BlockStatements()
	}
	{
		p.SetState(72)
		p.Match(devcmdParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(74)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 9, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(73)
			_la = p.GetTokenStream().LA(1)

			if !(_la == devcmdParserEOF || _la == devcmdParserNEWLINE) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}

	} else if p.HasError() { // JIM
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
	p.RuleIndex = devcmdParserRULE_blockStatements
	return p
}

func InitEmptyBlockStatementsContext(p *BlockStatementsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_blockStatements
}

func (*BlockStatementsContext) IsBlockStatementsContext() {}

func NewBlockStatementsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockStatementsContext {
	var p = new(BlockStatementsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_blockStatements

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
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterBlockStatements(s)
	}
}

func (s *BlockStatementsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitBlockStatements(s)
	}
}

func (p *devcmdParser) BlockStatements() (localctx IBlockStatementsContext) {
	localctx = NewBlockStatementsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, devcmdParserRULE_blockStatements)
	p.SetState(78)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case devcmdParserRBRACE:
		p.EnterOuterAlt(localctx, 1)

	case devcmdParserEQUALS, devcmdParserCOLON, devcmdParserVAR_START, devcmdParserINCOMPLETE_VARIABLE_REFERENCE, devcmdParserESCAPED_CHAR, devcmdParserNAME, devcmdParserCOMMAND_TEXT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(77)
			p.NonEmptyBlockStatements()
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
	p.RuleIndex = devcmdParserRULE_nonEmptyBlockStatements
	return p
}

func InitEmptyNonEmptyBlockStatementsContext(p *NonEmptyBlockStatementsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_nonEmptyBlockStatements
}

func (*NonEmptyBlockStatementsContext) IsNonEmptyBlockStatementsContext() {}

func NewNonEmptyBlockStatementsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NonEmptyBlockStatementsContext {
	var p = new(NonEmptyBlockStatementsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_nonEmptyBlockStatements

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
	return s.GetTokens(devcmdParserSEMICOLON)
}

func (s *NonEmptyBlockStatementsContext) SEMICOLON(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserSEMICOLON, i)
}

func (s *NonEmptyBlockStatementsContext) AllNEWLINE() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserNEWLINE)
}

func (s *NonEmptyBlockStatementsContext) NEWLINE(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserNEWLINE, i)
}

func (s *NonEmptyBlockStatementsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NonEmptyBlockStatementsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NonEmptyBlockStatementsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterNonEmptyBlockStatements(s)
	}
}

func (s *NonEmptyBlockStatementsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitNonEmptyBlockStatements(s)
	}
}

func (p *devcmdParser) NonEmptyBlockStatements() (localctx INonEmptyBlockStatementsContext) {
	localctx = NewNonEmptyBlockStatementsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, devcmdParserRULE_nonEmptyBlockStatements)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(80)
		p.BlockStatement()
	}
	p.SetState(88)
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
				p.SetState(81)
				p.Match(devcmdParserSEMICOLON)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			p.SetState(83)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)

			if _la == devcmdParserNEWLINE {
				{
					p.SetState(82)
					p.Match(devcmdParserNEWLINE)
					if p.HasError() {
						// Recognition error - abort rule
						goto errorExit
					}
				}

			}
			{
				p.SetState(85)
				p.BlockStatement()
			}

		}
		p.SetState(90)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(92)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == devcmdParserSEMICOLON {
		{
			p.SetState(91)
			p.Match(devcmdParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
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

// IBlockStatementContext is an interface to support dynamic dispatch.
type IBlockStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	CommandText() ICommandTextContext
	AMPERSAND() antlr.TerminalNode

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
	p.RuleIndex = devcmdParserRULE_blockStatement
	return p
}

func InitEmptyBlockStatementContext(p *BlockStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_blockStatement
}

func (*BlockStatementContext) IsBlockStatementContext() {}

func NewBlockStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockStatementContext {
	var p = new(BlockStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_blockStatement

	return p
}

func (s *BlockStatementContext) GetParser() antlr.Parser { return s.parser }

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

func (s *BlockStatementContext) AMPERSAND() antlr.TerminalNode {
	return s.GetToken(devcmdParserAMPERSAND, 0)
}

func (s *BlockStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterBlockStatement(s)
	}
}

func (s *BlockStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitBlockStatement(s)
	}
}

func (p *devcmdParser) BlockStatement() (localctx IBlockStatementContext) {
	localctx = NewBlockStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, devcmdParserRULE_blockStatement)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(94)
		p.CommandText()
	}
	p.SetState(96)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == devcmdParserAMPERSAND {
		{
			p.SetState(95)
			p.Match(devcmdParserAMPERSAND)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
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
	p.RuleIndex = devcmdParserRULE_continuationLine
	return p
}

func InitEmptyContinuationLineContext(p *ContinuationLineContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_continuationLine
}

func (*ContinuationLineContext) IsContinuationLineContext() {}

func NewContinuationLineContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ContinuationLineContext {
	var p = new(ContinuationLineContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_continuationLine

	return p
}

func (s *ContinuationLineContext) GetParser() antlr.Parser { return s.parser }

func (s *ContinuationLineContext) BACKSLASH() antlr.TerminalNode {
	return s.GetToken(devcmdParserBACKSLASH, 0)
}

func (s *ContinuationLineContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(devcmdParserNEWLINE, 0)
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
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterContinuationLine(s)
	}
}

func (s *ContinuationLineContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitContinuationLine(s)
	}
}

func (p *devcmdParser) ContinuationLine() (localctx IContinuationLineContext) {
	localctx = NewContinuationLineContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, devcmdParserRULE_continuationLine)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(98)
		p.Match(devcmdParserBACKSLASH)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(99)
		p.Match(devcmdParserNEWLINE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(100)
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
	AllESCAPED_CHAR() []antlr.TerminalNode
	ESCAPED_CHAR(i int) antlr.TerminalNode
	AllVariableReference() []IVariableReferenceContext
	VariableReference(i int) IVariableReferenceContext
	AllINCOMPLETE_VARIABLE_REFERENCE() []antlr.TerminalNode
	INCOMPLETE_VARIABLE_REFERENCE(i int) antlr.TerminalNode
	AllCOLON() []antlr.TerminalNode
	COLON(i int) antlr.TerminalNode
	AllEQUALS() []antlr.TerminalNode
	EQUALS(i int) antlr.TerminalNode
	AllCOMMAND_TEXT() []antlr.TerminalNode
	COMMAND_TEXT(i int) antlr.TerminalNode
	AllNAME() []antlr.TerminalNode
	NAME(i int) antlr.TerminalNode

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
	p.RuleIndex = devcmdParserRULE_commandText
	return p
}

func InitEmptyCommandTextContext(p *CommandTextContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_commandText
}

func (*CommandTextContext) IsCommandTextContext() {}

func NewCommandTextContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CommandTextContext {
	var p = new(CommandTextContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_commandText

	return p
}

func (s *CommandTextContext) GetParser() antlr.Parser { return s.parser }

func (s *CommandTextContext) AllESCAPED_CHAR() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserESCAPED_CHAR)
}

func (s *CommandTextContext) ESCAPED_CHAR(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserESCAPED_CHAR, i)
}

func (s *CommandTextContext) AllVariableReference() []IVariableReferenceContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IVariableReferenceContext); ok {
			len++
		}
	}

	tst := make([]IVariableReferenceContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IVariableReferenceContext); ok {
			tst[i] = t.(IVariableReferenceContext)
			i++
		}
	}

	return tst
}

func (s *CommandTextContext) VariableReference(i int) IVariableReferenceContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariableReferenceContext); ok {
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

	return t.(IVariableReferenceContext)
}

func (s *CommandTextContext) AllINCOMPLETE_VARIABLE_REFERENCE() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserINCOMPLETE_VARIABLE_REFERENCE)
}

func (s *CommandTextContext) INCOMPLETE_VARIABLE_REFERENCE(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserINCOMPLETE_VARIABLE_REFERENCE, i)
}

func (s *CommandTextContext) AllCOLON() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserCOLON)
}

func (s *CommandTextContext) COLON(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserCOLON, i)
}

func (s *CommandTextContext) AllEQUALS() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserEQUALS)
}

func (s *CommandTextContext) EQUALS(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserEQUALS, i)
}

func (s *CommandTextContext) AllCOMMAND_TEXT() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserCOMMAND_TEXT)
}

func (s *CommandTextContext) COMMAND_TEXT(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserCOMMAND_TEXT, i)
}

func (s *CommandTextContext) AllNAME() []antlr.TerminalNode {
	return s.GetTokens(devcmdParserNAME)
}

func (s *CommandTextContext) NAME(i int) antlr.TerminalNode {
	return s.GetToken(devcmdParserNAME, i)
}

func (s *CommandTextContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CommandTextContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CommandTextContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterCommandText(s)
	}
}

func (s *CommandTextContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitCommandText(s)
	}
}

func (p *devcmdParser) CommandText() (localctx ICommandTextContext) {
	localctx = NewCommandTextContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, devcmdParserRULE_commandText)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(109)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = ((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&124940) != 0) {
		p.SetState(109)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case devcmdParserESCAPED_CHAR:
			{
				p.SetState(102)
				p.Match(devcmdParserESCAPED_CHAR)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		case devcmdParserVAR_START:
			{
				p.SetState(103)
				p.VariableReference()
			}

		case devcmdParserINCOMPLETE_VARIABLE_REFERENCE:
			{
				p.SetState(104)
				p.Match(devcmdParserINCOMPLETE_VARIABLE_REFERENCE)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		case devcmdParserCOLON:
			{
				p.SetState(105)
				p.Match(devcmdParserCOLON)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		case devcmdParserEQUALS:
			{
				p.SetState(106)
				p.Match(devcmdParserEQUALS)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		case devcmdParserCOMMAND_TEXT:
			{
				p.SetState(107)
				p.Match(devcmdParserCOMMAND_TEXT)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		case devcmdParserNAME:
			{
				p.SetState(108)
				p.Match(devcmdParserNAME)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

		p.SetState(111)
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

// IVariableReferenceContext is an interface to support dynamic dispatch.
type IVariableReferenceContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	VAR_START() antlr.TerminalNode
	NAME() antlr.TerminalNode
	VAR_END() antlr.TerminalNode

	// IsVariableReferenceContext differentiates from other interfaces.
	IsVariableReferenceContext()
}

type VariableReferenceContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariableReferenceContext() *VariableReferenceContext {
	var p = new(VariableReferenceContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_variableReference
	return p
}

func InitEmptyVariableReferenceContext(p *VariableReferenceContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = devcmdParserRULE_variableReference
}

func (*VariableReferenceContext) IsVariableReferenceContext() {}

func NewVariableReferenceContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableReferenceContext {
	var p = new(VariableReferenceContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = devcmdParserRULE_variableReference

	return p
}

func (s *VariableReferenceContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableReferenceContext) VAR_START() antlr.TerminalNode {
	return s.GetToken(devcmdParserVAR_START, 0)
}

func (s *VariableReferenceContext) NAME() antlr.TerminalNode {
	return s.GetToken(devcmdParserNAME, 0)
}

func (s *VariableReferenceContext) VAR_END() antlr.TerminalNode {
	return s.GetToken(devcmdParserVAR_END, 0)
}

func (s *VariableReferenceContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableReferenceContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableReferenceContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.EnterVariableReference(s)
	}
}

func (s *VariableReferenceContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(devcmdListener); ok {
		listenerT.ExitVariableReference(s)
	}
}

func (p *devcmdParser) VariableReference() (localctx IVariableReferenceContext) {
	localctx = NewVariableReferenceContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, devcmdParserRULE_variableReference)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(113)
		p.Match(devcmdParserVAR_START)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(114)
		p.Match(devcmdParserNAME)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(115)
		p.Match(devcmdParserVAR_END)
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
