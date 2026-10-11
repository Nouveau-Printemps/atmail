package filter

import (
	"fmt"
)

var actions = map[string]EvaluationAction{
	"reject":      setResultAction(ActionReject),
	"accept":      setResultAction(ActionAccept),
	"temp_reject": setResultAction(ActionTempFail),
	"discard":     setResultAction(ActionDiscard),
	"quarantine":  setResultAction(ActionQuarantine),
	"forward": func(ctx *EvaluationContext, params []*EvaluationVariable) error {
		if len(params) != 1 || params[0].Type != TypeString {
			return fmt.Errorf("expected one string as argument, not %#v", params)
		}
		ctx.Result.AddRecipient = append(ctx.Result.AddRecipient, params[0].Value.(string))
		return nil
	},
	"mailbox": func(ctx *EvaluationContext, params []*EvaluationVariable) error {
		if len(params) != 1 || params[0].Type != TypeString {
			return fmt.Errorf("expected one string as argument, not %#v", params)
		}
		ctx.Result.Mailbox = new(params[0].Value.(string))
		return nil
	},
	"set_header": func(ctx *EvaluationContext, params []*EvaluationVariable) error {
		if len(params) != 2 || params[0].Type != TypeString || params[1].Type != TypeString {
			return fmt.Errorf("expected two strings as argument, not %#v", params)
		}
		ctx.Result.ModifyHeader[params[0].Value.(string)] = params[1].Value.(string)
		return nil
	},
	"spam": func(ctx *EvaluationContext, params []*EvaluationVariable) error {
		ctx.Result.Mailbox = new("Junk")
		ctx.Result.Action = ActionQuarantine
		return nil
	},
}

func setResultAction(a ResultAction) EvaluationAction {
	return func(ctx *EvaluationContext, _ []*EvaluationVariable) error {
		ctx.Result.Action = a
		return nil
	}
}
