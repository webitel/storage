package sqlstore

import (
	"github.com/lib/pq"
	"github.com/webitel/storage/model"
)

type ConstraintKind string

const (
	ConstraintFK     ConstraintKind = "23503"
	ConstraintUnique ConstraintKind = "23505"
	ConstraintCheck  ConstraintKind = "23514"
)

type DBConstraint struct {
	Name  string
	Table string
	Kind  ConstraintKind
}

func NewConstraintFromError(err error) *DBConstraint {
	pqErr, ok := err.(*pq.Error)
	if !ok {
		return nil
	}

	switch pqErr.Code {
	case pq.ErrorCode(ConstraintFK), pq.ErrorCode(ConstraintUnique), pq.ErrorCode(ConstraintCheck):
	default:
		return nil
	}

	return &DBConstraint{
		Name:  pqErr.Constraint,
		Table: pqErr.Table,
		Kind:  ConstraintKind(pqErr.Code),
	}
}

type ConstraintHandler func(c *DBConstraint) model.AppError

type BaseConstraintMapper struct {
	rules    map[string]ConstraintHandler
	fallback model.AppError
}

func NewConstraintMapper(fallback model.AppError) *BaseConstraintMapper {
	return &BaseConstraintMapper{
		rules:    make(map[string]ConstraintHandler),
		fallback: fallback,
	}
}

func (m *BaseConstraintMapper) RegisterStatic(constraintName string, appErr model.AppError) *BaseConstraintMapper {
	m.rules[constraintName] = func(c *DBConstraint) model.AppError {
		return appErr
	}
	return m
}

func (m *BaseConstraintMapper) RegisterFunc(constraintName string, handler ConstraintHandler) *BaseConstraintMapper {
	m.rules[constraintName] = handler
	return m
}

func (m *BaseConstraintMapper) Map(err error) model.AppError {
	c := NewConstraintFromError(err)
	if c == nil {
		return nil
	}

	if handler, exists := m.rules[c.Name]; exists {
		return handler(c)
	}

	if m.fallback != nil {
		return m.fallback
	}

	return model.NewCustomCodeError("store.sql.constraint_violation", c.Name, 409)
}

var MediaFileConstraints = NewConstraintMapper(
	model.NewCustomCodeError("media.delete.is_used", "Cannot delete file because it is currently in use within the system", 409),
).RegisterStatic(
	"cc_agent_media_files_id_fk",
	model.NewCustomCodeError("media.delete.is_used.agent", "Cannot delete file because it is assigned to an agent as greeting", 409),
).RegisterStatic(
	"cc_queue_media_files_id_fk",
	model.NewCustomCodeError("media.delete.is_used.queue", "Cannot delete file because it is configured in queue settings as ringtone", 409),
)
