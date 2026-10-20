package relay

import (
	"bytes"
	"slices"
	"time"

	"github.com/emersion/go-message"
	"github.com/emersion/go-smtp"
	"nouveauprintemps.org/atmail/filter"
)

func checkFilters(ctx *filter.EvaluationContext, filters []*filter.Tree) *filter.Result {
	for _, f := range filters {
		res, ok := f.Eval(ctx.Clone())
		if ok {
			return res
		}
	}
	return &filter.Result{Action: filter.ActionContinue}
}

func applyResult(s *Session, msg *message.Entity, wait *time.Duration, res *filter.Result) error {
	switch res.Action {
	case filter.ActionAccept:
		err := s.backend.Rspamd.LearnHam(s, msg)
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
		err := s.backend.Rspamd.LearnSpam(s, msg)
		if err != nil {
			return err
		}
	}
	s.To = slices.DeleteFunc(s.To, func(to Rcpt) bool {
		return slices.Contains(res.RemoveRecipient, to.Address)
	})
	for _, to := range res.AddRecipient {
		parsed := ParseAddress(to)
		s.To = append(s.To, Rcpt{
			User:    parsed[0],
			Domain:  parsed[1],
			Address: to,
		})
	}
	if res.Mailbox != nil {
		for i, to := range s.To {
			to.Folder = *res.Mailbox
			s.To[i] = to
		}
	}
	for k, v := range res.ModifyHeader {
		msg.Header.Set(k, v)
	}
	if res.Body != nil {
		msg.Body = bytes.NewBufferString(*res.Body)
	}
	return nil
}
