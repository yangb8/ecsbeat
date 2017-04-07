package ecs

import (
	"fmt"
	"path"
	"reflect"
	"runtime"
	"testing"
)

func areEqual(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	return reflect.DeepEqual(expected, actual)
}

// AssertEqual ...
func AssertEqual(t *testing.T, expected, actual interface{}, message string) {
	if !areEqual(expected, actual) {
		Error(t, fmt.Sprintf("Not equal: %#v (expected) != %#v (actual), extra msg: %s", expected, actual, message))
	}
}

// AssertNotEqual ...
func AssertNotEqual(t *testing.T, expected, actual interface{}, message string) {
	if areEqual(expected, actual) {
		Error(t, fmt.Sprintf("Equal: %#v (expected) == %#v (actual), extra msg: %s", expected, actual, message))
	}
}

// AssertEqualFatal ...
func AssertEqualFatal(t *testing.T, expected, actual interface{}, message string) {
	if !areEqual(expected, actual) {
		Fatal(t, fmt.Sprintf("Not equal: %#v (expected) != %#v (actual), extra msg: %s", expected, actual, message))
	}
}

// AssertNotEqualFatal ...
func AssertNotEqualFatal(t *testing.T, expected, actual interface{}, message string) {
	if areEqual(expected, actual) {
		Fatal(t, fmt.Sprintf("Equal: %#v (expected) == %#v (actual), extra msg: %s", expected, actual, message))
	}
}

// Error ...
func Error(t *testing.T, failureMessage string) {
	_, callerFile, callerLine, _ := runtime.Caller(2)
	t.Errorf("-- %s:%d: error -- %s", path.Base(callerFile), callerLine, failureMessage)
}

// Fatal ...
func Fatal(t *testing.T, failureMessage string) {
	_, callerFile, callerLine, _ := runtime.Caller(2)
	t.Fatalf("-- %s:%d: error -- %s", path.Base(callerFile), callerLine, failureMessage)
}
