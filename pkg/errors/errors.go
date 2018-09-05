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
