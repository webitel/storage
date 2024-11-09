package model

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

const (
	PAGE_DEFAULT     = 0
	PER_PAGE_DEFAULT = 40
	PER_PAGE_MAXIMUM = 10000
)

type ListRequest struct {
	Q       string
	Page    int
	PerPage int
	endList bool

	Fields []string
	Sort   string
}

func (l *ListRequest) RemoveLastElemIfNeed(slicePtr interface{}) {
	s := reflect.ValueOf(slicePtr)
	if s.Kind() != reflect.Ptr || s.Type().Elem().Kind() != reflect.Slice {
		panic(fmt.Errorf("first argument to Remove must be pointer to slice, not %T", slicePtr))
	}
	if s.IsNil() {
		return
	}

	itr := s.Elem()

	l.endList = itr.Len() <= l.PerPage

	if l.endList {
		return
	}

	newSlice := reflect.MakeSlice(itr.Type(), itr.Len()-1, itr.Len()-1)
	reflect.Copy(newSlice.Slice(0, newSlice.Len()), itr.Slice(0, itr.Len()-1))
	s.Elem().Set(newSlice)
}

func (l *ListRequest) EndOfList() bool {
	return l.endList
}

func (l *ListRequest) GetQ() *string {
	if l.Q != "" {
		return NewString(strings.Replace(l.Q, "*", "%", -1))
	}

	return nil
}

func (l *ListRequest) GetLimit() int {
	l.valid()
	return l.PerPage + 1 //FIXME for next page...
}

func (l *ListRequest) GetOffset() int {
	l.valid()
	if l.Page <= 1 {
		return 0
	}
	return l.PerPage * (l.Page - 1)
}

func (l *ListRequest) valid() {
	if l.Page < 0 {
		l.Page = PAGE_DEFAULT
	}

	if l.PerPage < 1 || l.PerPage > PER_PAGE_MAXIMUM {
		l.PerPage = PER_PAGE_DEFAULT
	}
}

type FilterBetween struct {
	From int64
	To   int64
}

func GetBetweenFromTime(src *FilterBetween) *time.Time {
	if src == nil || src.From == 0 {
		return nil
	}
	t := time.Unix(0, src.From*int64(time.Millisecond))
	return &t
}
