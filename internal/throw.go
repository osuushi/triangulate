package internal

import "github.com/pkg/errors"

// Threading errors up and down all the recursive operations during
// trapezoidization and triangulation would add a ton of complexity to the code.
// Instead, we use panics, and the public API recovers to convert to an error.

type TriangulateError error

// Panic with a TriangulateError.
func fatalf(format string, args ...interface{}) {
	panic(errors.Errorf(format, args...))
}

func HandleTriangulatePanicRecover(r interface{}) error {
	if r != nil {
		if triangulateError, ok := r.(TriangulateError); ok {
			return triangulateError
		}
		panic(r)
	}
	return nil
}
