package filter

import (
	"errors"
	"maps"
	"reflect"

	"github.com/emersion/go-message"
)

type EvaluationContext struct {
	Variables map[string]*EvaluationVariable
	Actions   map[string]EvaluationAction
	Result    Result
	Verbose   bool
}

type EvaluationAction = func(*EvaluationContext, []*EvaluationVariable) error

type EvaluationVariable struct {
	Type    EvaluationVariableType
	Value   any
	Methods map[string]*EvaluationMethod
	Fields  map[string]*EvaluationVariable
}

type EvaluationMethod struct {
	ParamsType []EvaluationVariableType
	Action     func(*EvaluationContext, *EvaluationVariable, []*EvaluationVariable) (*EvaluationVariable, error)
}

type EvaluationVariableType uint8

const (
	TypeString EvaluationVariableType = iota
	TypeNumber
	TypeBool
	TypeCollection
	TypeAny
)

func (ctx *EvaluationContext) Clone() *EvaluationContext {
	return &EvaluationContext{
		Variables: maps.Clone(ctx.Variables),
		Actions:   maps.Clone(ctx.Actions),
		Result:    ctx.Result,
		Verbose:   ctx.Verbose,
	}
}

func (m *EvaluationMethod) Eval(ctx *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
	if len(m.ParamsType) != len(params) {
		return nil, errors.New("missing parameters")
	}
	for i, v := range params {
		if m.ParamsType[i] != TypeAny && m.ParamsType[i] != v.Type {
			return nil, errors.New("invalid parameters type")
		}
	}
	return m.Action(ctx, parent, params)
}

func (v *EvaluationVariable) Clone() *EvaluationVariable {
	var res EvaluationVariable
	if v.Fields != nil {
		res.Fields = maps.Clone(v.Fields)
	}
	if v.Methods != nil {
		res.Methods = maps.Clone(v.Methods)
	}
	switch reflect.TypeOf(v.Value).Kind() {
	case reflect.Map:
		val := reflect.ValueOf(v.Value)
		n := reflect.MakeMap(val.Type())
		for k, v := range val.Seq2() {
			n.SetMapIndex(k, v)
		}
		res.Value = n.Interface()
	default:
		res.Value = v.Value
	}
	return &res
}

type Rcpt = struct {
	User    string
	Domain  string
	Folder  string
	Address string
}

func rcptVariable(rcpt Rcpt) *EvaluationVariable {
	val := newString(rcpt.Address)
	val.Fields = map[string]*EvaluationVariable{
		"user":   newString(rcpt.User),
		"domain": newString(rcpt.Domain),
	}
	return val
}

func headerVariable(header message.Header) *EvaluationVariable {
	res := make(map[string]*EvaluationVariable, header.Len())
	for k, v := range header.Map() {
		if len(v) == 1 {
			res[k] = newString(v[0])
		} else {
			res[k] = newCollection[float64, string](sliceToMap(v, newString))
		}
	}
	return newCollection[string, any](res)
}

func InitEvalulationContext(from, to Rcpt, header message.Header, body string) *EvaluationContext {
	return &EvaluationContext{
		Result: Result{ModifyHeader: make(map[string]string)},
		Variables: map[string]*EvaluationVariable{
			"true":   newBool(true),
			"false":  newBool(false),
			"from":   rcptVariable(from),
			"to":     rcptVariable(to),
			"header": headerVariable(header),
			"body": &EvaluationVariable{
				Type:  TypeString,
				Value: body,
			},
		},
		Actions: actions,
	}
}

type ResultAction uint8

// Default actions in the milter protocol
const (
	ActionContinue ResultAction = iota
	ActionAccept
	ActionReject
	ActionTempFail
	ActionDiscard
	ActionConnectionFail
	ActionQuarantine
)

type Result struct {
	Action          ResultAction
	AddRecipient    []string
	RemoveRecipient []string
	SenderAddress   *Rcpt
	Mailbox         *string
	Body            *string
	ModifyHeader    map[string]string
}
