package advanced

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleTriangulatePanicRecover(t *testing.T) {
	testFn := func(shouldThrow bool, shouldPanic bool) (err error) {
		defer func() {
			recoveredErr := HandleTriangulatePanicRecover(recover())
			if recoveredErr != nil {
				err = recoveredErr
			}
		}()

		if shouldThrow {
			fatalf("kaboom!")
		}

		if shouldPanic {
			panic("true panic")
		}

		return nil
	}

	t.Run("with throw", func(t *testing.T) {
		err := testFn(true, false)
		assert.EqualError(t, err, "kaboom!")
	})

	t.Run("with real panic", func(t *testing.T) {
		assert.Panics(t, func() {
			testFn(false, true)
		})
	})

	t.Run("no error", func(t *testing.T) {
		err := testFn(false, false)
		assert.NoError(t, err)
	})
}
