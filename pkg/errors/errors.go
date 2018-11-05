package errors

import (
	"fmt"
	"strings"
)

type NotEnabledErr struct {
}

func (nee *NotEnabledErr) Error() string {
	return "integration is not enabled"
}

func IsNotEnabledErr(err error) bool {
	_, ok := err.(*NotEnabledErr)
	return ok
}

type AlreadyExistsErr struct {
}

func (ae *AlreadyExistsErr) Error() string {
	return "resource already exists"
}

func IsAlreadyExistsErr(err error) bool {
	_, ok := err.(*AlreadyExistsErr)
	return ok
}

type NotFoundErr struct {
	Resource string
}

func (nfe *NotFoundErr) Error() string {
	return fmt.Sprintf("%s : not found", nfe.Resource)
}

func IsNotFoundErr(err error) bool {
	_, ok := err.(*NotFoundErr)
	return ok
}

type MultiErr struct {
	errs []string
}

func NewMultiErr() *MultiErr {
	return &MultiErr{errs: []string{}}
}

func (me *MultiErr) Add(err error) {
	me.errs = append(me.errs, err.Error())
}

func (me *MultiErr) Error() string {
	fmt.Println(me.errs)
	if me.errs == nil {
		return ""
	}
	return strings.Join(me.errs, " : ")
}
