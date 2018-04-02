package helpers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"
)

// WriteFile test helper
func WriteFile(t *testing.T, filepath string, bytes []byte) {
	t.Helper()
	if err := ioutil.WriteFile(filepath, bytes, 0644); err != nil {
		t.Fatalf("failed writing .golden: %s", err)
	}
}

// ReadFile test helper
func ReadFile(t *testing.T, filepath string) []byte {
	t.Helper()
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}
	return b
}

func isAFunction(f interface{}) bool {
	actual := reflect.TypeOf(f)
	return actual.Kind() == reflect.Func && actual.NumIn() == 0 && actual.NumOut() > 0
}

func vetExtras(extras []interface{}) (bool, string) {
	for i, extra := range extras {
		if extra != nil {
			zeroValue := reflect.Zero(reflect.TypeOf(extra)).Interface()
			if !reflect.DeepEqual(zeroValue, extra) {
				message := fmt.Sprintf("Unexpected non-nil/non-zero extra argument at index %d:\n\t<%T>: %#v", i+1, extra, extra)
				return false, message
			}
		}
	}
	return true, ""
}

func pollFuncReturn(f interface{}) (interface{}, error) {
	values := reflect.ValueOf(f).Call([]reflect.Value{})

	extras := []interface{}{}
	for _, value := range values[1:] {
		extras = append(extras, value.Interface())
	}

	success, message := vetExtras(extras)

	if !success {
		return nil, errors.New(message)
	}

	return values[0].Interface(), nil
}

// ShouldEventuallyReturn asserts that eventually the return of f should be v, timeouts: 0 - evaluation interval, 1 - timeout
func ShouldEventuallyReturn(t *testing.T, f interface{}, v interface{}, timeouts ...time.Duration) bool {
	t.Helper()
	interval := 10 * time.Millisecond
	timeout := time.After(50 * time.Millisecond)
	switch len(timeouts) {
	case 1:
		interval = timeouts[0]
		break
	case 2:
		interval = timeouts[0]
		timeout = time.After(timeouts[1])
	}
	ticker := time.NewTicker(interval)
	if isAFunction(f) {
		for {
			select {
			case <-timeout:
				t.Fatalf("function f never returned value %s", v)
			case <-ticker.C:
				val, err := pollFuncReturn(f)
				if err != nil {
					t.Fatal(err)
				}
				if v == val {
					return true
				}
			}
		}
	} else {
		t.Fatal("ShouldEventuallyEqual should receive a function with no args and more than 0 outs")
		return false
	}
}
