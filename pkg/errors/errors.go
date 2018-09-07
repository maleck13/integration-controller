package errors

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
