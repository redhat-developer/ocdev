package binding

import (
	"errors"
	"fmt"
)

type ErrInvalidAnnotationPrefix string

func (e ErrInvalidAnnotationPrefix) Error() string {
	return fmt.Sprintf("invalid annotation prefix: %s", string(e))
}

func IsErrInvalidAnnotationPrefix(err error) bool {
	_, ok := err.(ErrInvalidAnnotationPrefix)
	return ok
}

var ErrInvalidAnnotationName = errors.New("invalid annotation name")

type ErrEmptyAnnotationName string

func (e ErrEmptyAnnotationName) Error() string {
	return fmt.Sprintf("empty annotation name: %s", string(e))
}

func IsErrEmptyAnnotationName(err error) bool {
	_, ok := err.(ErrEmptyAnnotationName)
	return ok
}
