package storage

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"nouveauprintemps.org/atmail/utils"
)

type searchDataSmall struct {
	id       int64
	filename string
	offset   int64
}

func Search(
	ctx context.Context,
	user string,
	mailbox int64,
	kind imapserver.NumKind,
	criteria *imap.SearchCriteria,
	options *imap.SearchOptions,
) (*imap.SearchData, error) {
	if options.ReturnSave {
		return nil, &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCode("NOTSAVED"),
			Text: "Server doesn't implement save",
		}
	}
	var where strings.Builder
	check := parseCriteria(criteria, &where)
	meta, err := Cache.DB(ctx, user)
	if err != nil {
		return nil, err
	}
	var rows *sql.Rows
	if where.Len() > 0 {
		rows, err = meta.db.QueryContext(ctx, `SELECT id, filename, offset FROM emails WHERE `+where.String())
	} else {
		rows, err = meta.db.QueryContext(ctx, `SELECT id, filename, offset FROM emails`)
	}
	if err != nil {
		return nil, err
	}
	var rawIds []searchDataSmall
	for rows.Next() {
		var id int64
		var filename string
		var offset int64
		err = rows.Scan(&id, &filename, &offset)
		if err != nil {
			return nil, err
		}
		v := searchDataSmall{
			id:       id,
			filename: filename,
			offset:   offset,
		}
		if !check(v) {
			continue
		}
		rawIds = append(rawIds, v)
	}
	ids := utils.Map(rawIds, func(id searchDataSmall) imap.UID {
		return imap.UID(id.id)
	})
	slices.SortFunc(rawIds, func(a, b searchDataSmall) int {
		return int(a.id - b.id)
	})
	var res imap.SearchData
	res.All = imap.UIDSetNum(ids...)
	if options.ReturnMax && len(ids) > 0 {
		res.Max = uint32(ids[len(ids)-1])
	}
	if options.ReturnMin && len(ids) > 0 {
		res.Min = uint32(ids[0])
	}
	if options.ReturnCount {
		res.Count = uint32(len(ids))
	}
	return &res, nil
}

func parseCriteria(criteria *imap.SearchCriteria, sb *strings.Builder) func(v searchDataSmall) bool {
	ids := make([]int64, 0, len(criteria.SeqNum)+len(criteria.UID))
	for _, set := range criteria.SeqNum {
		nums, _ := set.Nums()
		conv := utils.Map(nums, func(v uint32) int64 { return int64(v) })
		ids = append(ids, conv...)
	}
	for _, set := range criteria.UID {
		nums, _ := set.Nums()
		conv := utils.Map(nums, func(v imap.UID) int64 { return int64(v) })
		ids = append(ids, conv...)
	}
	needAnd := false
	if len(ids) > 0 {
		var idsb strings.Builder
		idsb.Grow(len(ids) + 2)
		idsb.WriteRune('(')
		idsb.WriteRune(')')
		for _, id := range ids {
			idsb.WriteString(strconv.Itoa(int(id)))
			idsb.WriteRune(',')
		}
		idss := idsb.String()
		idss = idss[:len(idss)-1]
		sb.WriteString("id IN ")
		sb.WriteString(idss)
		needAnd = true
	}
	if !criteria.Before.IsZero() {
		if needAnd {
			sb.WriteString(" AND ")
		}
		sb.WriteString("internal_date < ")
		sb.WriteString(strconv.Itoa(int(criteria.Before.Unix())))
		needAnd = true
	}
	if !criteria.Since.IsZero() {
		if needAnd {
			sb.WriteString(" AND ")
		}
		sb.WriteString("internal_date > ")
		sb.WriteString(strconv.Itoa(int(criteria.Since.Unix())))
		needAnd = true
	}
	if criteria.Larger != 0 {
		if needAnd {
			sb.WriteString(" AND ")
		}
		sb.WriteString("id > ")
		sb.WriteString(strconv.Itoa(int(criteria.Larger)))
		needAnd = true
	}
	if criteria.Smaller != 0 {
		if needAnd {
			sb.WriteString(" AND ")
		}
		sb.WriteString("id < ")
		sb.WriteString(strconv.Itoa(int(criteria.Smaller)))
		needAnd = true
	}
	for _, flag := range criteria.Flag {
		if needAnd {
			sb.WriteString(" AND ")
		}
		sb.WriteString(`EXISTS (
				SELECT * FROM emails_flags WHERE email_id = id and flag_id = (
					SELECT id FROM flags WHERE name = '`)
		sb.WriteString(strings.ReplaceAll(string(flag), `'`, `\'`))
		sb.WriteString(`'))`)
		needAnd = true
	}
	for _, flag := range criteria.NotFlag {
		if needAnd {
			sb.WriteString(" AND ")
		}
		sb.WriteString(`NOT EXISTS (
				SELECT * FROM emails_flags WHERE email_id = id and flag_id = (
					SELECT id FROM flags WHERE name = '`)
		sb.WriteString(strings.ReplaceAll(string(flag), `'`, `\'`))
		sb.WriteString(`'))`)
		needAnd = true

	}
	var notFns []func(v searchDataSmall) bool
	for _, not := range criteria.Not {
		if needAnd {
			sb.WriteString(" AND ")
		}
		sb.WriteString("NOT (")
		notFns = append(notFns, parseCriteria(&not, sb))
		sb.WriteRune(')')
		needAnd = true
	}
	var orFns [][2]func(v searchDataSmall) bool
	for _, or := range criteria.Or {
		if needAnd {
			sb.WriteString(" AND ")
		}
		sb.WriteRune('(')
		or1 := parseCriteria(&or[0], sb)
		sb.WriteString(") OR (")
		or2 := parseCriteria(&or[1], sb)
		sb.WriteRune(')')
		orFns = append(orFns, [2]func(v searchDataSmall) bool{or1, or2})
		needAnd = true
	}
	return func(v searchDataSmall) bool {
		base := !utils.Any(utils.Map(notFns, func(fn func(v searchDataSmall) bool) bool {
			return fn(v)
		})) && utils.All(utils.Map(orFns, func(fns [2]func(v searchDataSmall) bool) bool {
			return fns[0](v) || fns[1](v)
		}))
		if !base {
			return false
		}
		//TODO: handle Header, Body and Text
		return true
	}
}
