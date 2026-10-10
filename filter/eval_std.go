package filter

import (
	"strings"
)

var stringMethods = map[string]*EvaluationMethod{
	"len": {
		ReturnType: TypeNumber,
		Action: func(ec *EvaluationContext, parent *EvaluationVariable, _ []*EvaluationVariable) (*EvaluationVariable, error) {
			return newNumber(float64(len(parent.Value.(string)))), nil
		},
	},
	"contains": {
		ReturnType: TypeBool,
		ParamsType: []EvaluationVariableType{TypeString},
		Action: func(ec *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
			return newBool(strings.Contains(parent.Value.(string), params[0].Value.(string))), nil
		},
	},
	"split": {
		ReturnType: TypeCollection,
		ParamsType: []EvaluationVariableType{TypeString},
		Action: func(ec *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
			res := strings.Split(parent.Value.(string), params[0].Value.(string))
			return newCollection(sliceToMap(res)), nil
		},
	},
}

func newString(val string) *EvaluationVariable {
	return &EvaluationVariable{
		Type:    TypeString,
		Value:   val,
		Methods: stringMethods,
	}
}

func newNumber(val float64) *EvaluationVariable {
	return &EvaluationVariable{
		Type:  TypeNumber,
		Value: val,
	}
}

func newBool(val bool) *EvaluationVariable {
	return &EvaluationVariable{
		Type:  TypeBool,
		Value: val,
	}
}

func newCollection[K comparable, V any](val map[K]V) *EvaluationVariable {
	return &EvaluationVariable{
		Type:  TypeCollection,
		Value: val,
	}
}

func sliceToMap[T any](sl []T) map[uint]T {
	res := make(map[uint]T, len(sl))
	for i, v := range sl {
		res[uint(i)] = v
	}
	return res
}
