package errors

import "fmt"

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
