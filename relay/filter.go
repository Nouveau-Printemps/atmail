package relay

import (
	"bytes"
	"maps"
	"time"

	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/filter"
	"nouveauprintemps.org/atmail/utils"
)

func checkFilters(ctx *filter.EvaluationContext, filters map[string]*filter.Tree) *filter.Result {
	for name, f := range filters {
		ctx.LoggerKey = name
		res, ok := f.Eval(ctx.Clone())
		if ok {
			if ctx.Verbose {
				ctx.Logger().Debug("result obtained", "res", res.Action)
			}
			return res
		}
	}
	return &filter.Result{Action: filter.ActionContinue}
}

func applyResult(s *Session, msg *message.Entity, wait *time.Duration, res *filter.Result) error {
	switch res.Action {
	case filter.ActionAccept:
		err := s.backend.Rspamd.LearnHam(msg)
		if err != nil {
			return err
		}
	case filter.ActionContinue:
	case filter.ActionDiscard:
		return nil
	case filter.ActionQuarantine:
		*wait = max(*wait, 15*time.Minute)
	case filter.ActionReject:
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: [3]int{5, 7, 1},
			Message:      "Your message is unwanted",
		}
	case filter.ActionTempFail:
		return &smtp.SMTPError{
			Code:         450,
			EnhancedCode: [3]int{4, 7, 1},
			Message:      "Your message is temporarily unwanted, retry later",
		}
	default:
		panic("action not implemented")
	}
	if res.Spam {
		err := s.backend.Rspamd.LearnSpam(msg)
		if err != nil {
			return err
		}
	}
	for rcpt := range res.RemoveRecipient {
		maps.DeleteFunc(s.To, func(k Rcpt, v struct{}) bool {
			return k.Address == rcpt
		})
	}
	for to := range res.AddRecipient {
		parsed := ParseAddress(to)
		utils.SetInsert(s.To, Rcpt{
			User:    parsed[0],
			Domain:  parsed[1],
			Address: to,
		})
	}
	if res.Mailbox != nil {
		s.To = utils.SetMap(s.To, func(to Rcpt) Rcpt {
			to.Folder = *res.Mailbox
			return to
		})
	}
	for k, v := range res.ModifyHeader {
		msg.Header.Set(k, v)
	}
	if res.Body != nil {
		msg.Body = bytes.NewBufferString(*res.Body)
	}
	return nil
}
