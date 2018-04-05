package helpers

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// WriteFile test helper
func WriteFile(t *testing.T, filepath string, bytes []byte) {
	t.Helper()
	if err := ioutil.WriteFile(filepath, bytes, 0644); err != nil {
		t.Fatalf("failed writing file: %s", err)
	}
}

// ReadFile test helper
func ReadFile(t *testing.T, filepath string) []byte {
	t.Helper()
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("failed reading file: %s", err)
	}
	return b
}

// FixtureGoldenFileName returns the golden file name on fixtures path
func FixtureGoldenFileName(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("fixtures", name+".golden")
}

func vetExtras(extras []interface{}) (bool, string) {
	for i, extra := range extras {
		if extra != nil {
			zeroValue := reflect.Zero(reflect.TypeOf(extra)).Interface()
			if !reflect.DeepEqual(zeroValue, extra) {
				message := fmt.Sprintf("unexpected non-nil/non-zero extra argument at index %d:\n\t<%T>: %#v", i+1, extra, extra)
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

// ShouldEventuallyReceive should asserts that eventually channel c receives a value
func ShouldEventuallyReceive(t *testing.T, c interface{}, timeouts ...time.Duration) interface{} {
	t.Helper()
	if !isChan(c) {
		t.Fatal("ShouldEventuallyReceive c argument should be a channel")
	}
	v := reflect.ValueOf(c)

	timeout := time.After(100 * time.Millisecond)

	if len(timeouts) > 0 {
		timeout = time.After(timeouts[0])
	}

	recvChan := make(chan reflect.Value)

	go func() {
		v, ok := v.Recv()
		if ok {
			recvChan <- v
		}
	}()

	select {
	case <-timeout:
		t.Fatal(errors.New("timed out waiting for channel to receive"))
	case a := <-recvChan:
		return a.Interface()
	}

	return nil
}

// ShouldEventuallyReturn asserts that eventually the return of f should be v, timeouts: 0 - evaluation interval, 1 - timeout
func ShouldEventuallyReturn(t *testing.T, f interface{}, v interface{}, timeouts ...time.Duration) {
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
	defer ticker.Stop()

	if isFunction(f) {
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
					return
				}
			}
		}
	} else {
		t.Fatal("ShouldEventuallyEqual should receive a function with no args and more than 0 outs")
		return
	}
}
