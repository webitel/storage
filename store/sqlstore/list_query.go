package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/webitel/storage/model"
)

type Filter map[string][]interface{}

type Entity interface {
	AllowFields() []string
	DefaultFields() []string
	EntityName() string
	DefaultOrder() string
}

func GetFields(f []string, e Entity) []string {

	if f == nil || len(f) < 1 {
		//TODO add cache
		f = e.DefaultFields()
	}

	res := make([]string, 0, len(f))

	for _, v := range f {
		if containsString(e.AllowFields(), v) {
			res = append(res, pq.QuoteIdentifier(v))
		}
	}

	return res
}

func QuoteIdentifier(name string) string {
	return pq.QuoteIdentifier(name)
}

func QuoteLiteral(name string) string {
	return pq.QuoteLiteral(name)
}

func GetOrderBy(t, s string) string {
	if s != "" {
		sort := ""
		field := ""
		if s[0] == '+' || s[0] == 32 {
			sort = "asc"
			field = s[1:]
		} else if s[0] == '-' {
			sort = "desc"
			field = s[1:]
		} else {
			field = s
		}

		return fmt.Sprintf(`order by case when call_center.cc_is_lookup(%s, %s) then (%s::text)::json->>'name' end %s,
         case when not call_center.cc_is_lookup(%s, %s) then %s end %s`, QuoteLiteral(t), QuoteLiteral(field), QuoteIdentifier(field),
			sort, QuoteLiteral(t), QuoteLiteral(field), QuoteIdentifier(field), sort)

	}

	return "" //TODO
}

// TODO filter
func Build(req *model.ListRequest, schema string, where string, e Entity, args map[string]interface{}) string {
	s := GetFields(req.Fields, e)
	sort := ""

	if req.Sort != "" {
		sort = req.Sort
	} else if e.DefaultOrder() != "" {
		sort = e.DefaultOrder()
	}

	args["Offset"] = req.GetOffset()
	args["Limit"] = req.GetLimit()

	t := pq.QuoteIdentifier(e.EntityName())

	if schema != "" {
		t = pq.QuoteIdentifier(schema) + "." + t
	}

	query := fmt.Sprintf(`select %s
	from %s as t
	where %s
	%s
	offset :Offset
	limit :Limit`, strings.Join(s, ", "), t, where, GetOrderBy(e.EntityName(), sort))

	return query
}

func (s *SqlSupplier) ListQuery(out interface{}, req model.ListRequest, where string, e Entity, params map[string]interface{}) error {
	q := Build(&req, "storage", where, e, params)
	_, err := s.GetReplica().Select(out, q, params)
	if err != nil {
		return err
	}

	return nil
}

func (s *SqlSupplier) ListQueryCtx(ctx context.Context, out interface{}, req model.ListRequest, where string, e Entity, params map[string]interface{}) error {
	q := Build(&req, "storage", where, e, params)
	_, err := s.GetReplica().WithContext(ctx).Select(out, q, params)
	if err != nil {
		return err
	}

	return nil
}

// todo
func (s *SqlSupplier) ListQueryFromSchema(out interface{}, schema string, req model.ListRequest, where string, e Entity, params map[string]interface{}) error {
	q := Build(&req, schema, where, e, params)
	_, err := s.GetReplica().Select(out, q, params)
	if err != nil {
		return err
	}

	return nil
}

func containsString(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}
