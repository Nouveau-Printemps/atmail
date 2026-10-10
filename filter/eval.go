package filter

import (
	"errors"
	"maps"

	"github.com/emersion/go-message"
)

type EvaluationContext struct {
	Variables map[string]*EvaluationVariable
	Actions   map[string]*EvaluationMethod
	Verbose   bool
}

type EvaluationVariable struct {
	Type    EvaluationVariableType
	Value   any
	Methods map[string]*EvaluationMethod
	Fields  map[string]*EvaluationVariable
}

type EvaluationMethod struct {
	Name       string
	ParamsType []EvaluationVariableType
	ReturnType EvaluationVariableType
	Action     func(*EvaluationContext, []*EvaluationVariable) (*EvaluationVariable, error)
}

type EvaluationVariableType uint8

const (
	TypeString EvaluationVariableType = iota
	TypeNumber
	TypeBool
)

func (ctx *EvaluationContext) Clone() *EvaluationContext {
	return &EvaluationContext{
		Variables: maps.Clone(ctx.Variables),
		Actions:   maps.Clone(ctx.Actions),
	}
}

func (m *EvaluationMethod) Eval(ctx *EvaluationContext, params []*EvaluationVariable) (*EvaluationVariable, error) {
	if len(m.ParamsType) != len(params) {
		return nil, errors.New("missing parameters")
	}
	for i, v := range params {
		if m.ParamsType[i] != v.Type {
			return nil, errors.New("invalid parameters type")
		}
	}
	return m.Action(ctx, params)
}

type Rcpt = struct {
	User    string
	Domain  string
	Folder  string
	Address string
}

func rcptVariable(rcpt Rcpt) *EvaluationVariable {
	return &EvaluationVariable{
		Type:  TypeString,
		Value: rcpt.Address,
		Fields: map[string]*EvaluationVariable{
			"user":   {Type: TypeString, Value: rcpt.User},
			"domain": {Type: TypeString, Value: rcpt.Domain},
		},
	}
}

func InitEvalulationContext(from, to Rcpt, header message.Header, body string) *EvaluationContext {
	return &EvaluationContext{
		Variables: map[string]*EvaluationVariable{
			"from": rcptVariable(from),
			"to":   rcptVariable(to),
			"body": &EvaluationVariable{
				Type:  TypeString,
				Value: body,
			},
		},
	}
}
