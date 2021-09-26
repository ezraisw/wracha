package wracha

import "fmt"

type cachingError struct {
	message     string
	previousErr error
}

func (e cachingError) Error() string {
	return fmt.Sprintf("%s (%s)", e.message, e.previousErr.Error())
}

func (a cachingError) Unwrap() error {
	return a.previousErr
}
