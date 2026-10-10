package filter

import (
	"fmt"
	"reflect"
	"strings"
)

func stringMethods() map[string]*EvaluationMethod {
	return map[string]*EvaluationMethod{
		"len": {
			Action: func(ec *EvaluationContext, parent *EvaluationVariable, _ []*EvaluationVariable) (*EvaluationVariable, error) {
				return newNumber(float64(len(parent.Value.(string)))), nil
			},
		},
		"contains": {
			ParamsType: []EvaluationVariableType{TypeString},
			Action: func(ec *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
				return newBool(strings.Contains(parent.Value.(string), params[0].Value.(string))), nil
			},
		},
		"split": {
			ParamsType: []EvaluationVariableType{TypeString},
			Action: func(ec *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
				res := strings.Split(parent.Value.(string), params[0].Value.(string))
				return newCollection[float64, string](sliceToMap(res, newString)), nil
			},
		},
		"get": {
			ParamsType: []EvaluationVariableType{TypeNumber},
			Action: func(ec *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
				s := parent.Value.(string)
				k := int(params[0].Value.(float64))
				if k >= len(s) {
					return nil, fmt.Errorf("out of range: %d of string size %d", k, len(s))
				}
				return newString(string(s[int(k)])), nil
			},
		},
		"sub": {
			ParamsType: []EvaluationVariableType{TypeNumber, TypeNumber},
			Action: func(ec *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
				s := parent.Value.(string)
				k1 := int(params[0].Value.(float64))
				k2 := int(params[1].Value.(float64))
				if k1 < 0 || k2 < 0 {
					return nil, fmt.Errorf("invalid range: must be positive or null")
				}
				if k1 > k2 {
					return nil, fmt.Errorf("invalid range: %d is bigger than %d", k1, k2)
				}
				if k1 > len(s) {
					return newString(""), nil
				}
				k2 = min(k2, len(s))
				return newString(s[k1:k2]), nil
			},
		},
	}
}

func newString(val string) *EvaluationVariable {
	return &EvaluationVariable{
		Type:    TypeString,
		Value:   val,
		Methods: stringMethods(),
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
		Methods: map[string]*EvaluationMethod{
			"not": {
				Action: func(ec *EvaluationContext, parent *EvaluationVariable, _ []*EvaluationVariable) (*EvaluationVariable, error) {
					return newBool(!parent.Value.(bool)), nil
				},
			},
		},
	}
}

func collectionMethods[K comparable, V any]() map[string]*EvaluationMethod {
	return map[string]*EvaluationMethod{
		"get": {
			ParamsType: []EvaluationVariableType{typeOf[K]()},
			Action: func(ec *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
				k := params[0].Value.(K)
				val, ok := parent.Value.(map[K]*EvaluationVariable)[k]
				if !ok {
					return nil, fmt.Errorf("%#v not found", k)
				}
				return val, nil
			},
		},
		"len": {
			Action: func(ec *EvaluationContext, parent *EvaluationVariable, _ []*EvaluationVariable) (*EvaluationVariable, error) {
				return newNumber(float64(len(parent.Value.(map[K]*EvaluationVariable)))), nil
			},
		},
		"set": {
			ParamsType: []EvaluationVariableType{typeOf[K](), typeOf[V]()},
			Action: func(ec *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
				parent = parent.Clone()
				parent.Value.(map[K]*EvaluationVariable)[params[0].Value.(K)] = params[1]
				return parent, nil
			},
		},
		"has_value": {
			ParamsType: []EvaluationVariableType{typeOf[V]()},
			Action: func(ec *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
				for _, v := range parent.Value.(map[K]*EvaluationVariable) {
					if reflect.DeepEqual(v.Value, params[0].Value) {
						return newBool(true), nil
					}
				}
				return newBool(false), nil
			},
		},
		"has_key": {
			ParamsType: []EvaluationVariableType{typeOf[K]()},
			Action: func(ec *EvaluationContext, parent *EvaluationVariable, params []*EvaluationVariable) (*EvaluationVariable, error) {
				_, ok := parent.Value.(map[K]*EvaluationVariable)[params[0].Value.(K)]
				return newBool(ok), nil
			},
		},
	}
}

func newCollection[K comparable, V any](val map[K]*EvaluationVariable) *EvaluationVariable {
	return &EvaluationVariable{
		Type:    TypeCollection,
		Value:   val,
		Methods: collectionMethods[K, V](),
	}
}

func sliceToMap[T any](sl []T, conv func(T) *EvaluationVariable) map[float64]*EvaluationVariable {
	res := make(map[float64]*EvaluationVariable, len(sl))
	for i, v := range sl {
		res[float64(i)] = conv(v)
	}
	return res
}

func typeOf[V any]() EvaluationVariableType {
	var v V
	tpe := reflect.TypeOf(v)
	switch tpe.Kind() {
	case reflect.Bool:
		return TypeBool
	case reflect.Float64:
		return TypeNumber
	case reflect.String:
		return TypeString
	case reflect.Slice, reflect.Map:
		return TypeCollection
	default:
		panic("internal error: unsupported kind")
	}
}
