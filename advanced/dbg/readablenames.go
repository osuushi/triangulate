package dbg

import (
	"fmt"
	"reflect"
	"strings"

	petname "github.com/dustinkirkland/golang-petname"
)

// This converts arbitrary strings into random readable names. It flagrantly
// leaks memory but generates the names lazily, so it's not a problem unless
// you're actually using it. This is helpful for turning pointer strings into
// something more easily distinguishable when debugging.

var memo map[interface{}]string

func init() {
	memo = make(map[interface{}]string)
	// Since the ids are generated in order of demand, we make them
	// nondetemrinistic to remind the user that the same name doesn't refer to the
	// same thing between runs.
	petname.NonDeterministicMode()
}

func Name(obj interface{}) string {
	if reflect.ValueOf(obj).IsNil() {
		return "Ø"
	}

	if r, ok := memo[obj]; ok {
		return r
	}
	r := fmt.Sprintf("%s%s", strings.Title(petname.Adjective()), strings.Title(petname.Name()))
	memo[obj] = r
	return r
}
