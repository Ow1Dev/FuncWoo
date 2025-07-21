package sigil

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gotest.tools/assert"
)

func TestInvalidHandlers(t *testing.T) {
	type valuer interface {
		Value(key any) any
	}

	type customContext interface {
		context.Context
		MyCustomMethod()
	}

	type myContext interface {
		Deadline() (deadline time.Time, ok bool)
		Done() <-chan struct{}
		Err() error
		Value(key any) any
	}

	testCases := []struct {
		name     string
		handler  any
		expected string
	}{
		{
			name:     "nil handler",
			expected: "handler function is nil",
			handler:  nil,
		},
		{
			name:     "handler is not a function",
			expected: "expected a function, got struct",
			handler:  struct{}{},
		},
		{
			name: "handler declares too many arguments",
			expected: "handler has too many parameters: 3",
			handler: func(n context.Context, x string, y string) error {
				return nil
			},
		},
		{
			name: "two argument handler does not context as first argument",
			expected: "first argument does not implement context.Context: got string",
			handler: func(a string, x context.Context) error {
				return nil
			},
		},
		{
			name: "handler returns too many values",
			expected: "too many return values: 3",
			handler: func() (error, error, error) {
				return nil, nil, nil
			},
		},
		{
			name: "handler returning two values does not declare error as the second return value",
			expected: "second return must be error, got string",
			handler: func() (error, string) {
				return nil, "hello"
			},
		},
		{
			name: "handler returning a single value does not implement error",
			expected: "single return must be error, got string",
			handler: func() string {
				return "hello"
			},
		},
		{
			name:    "no return value should not result in error",
			expected: "handler must return at least an error",
			handler: func() {
			},
		},
		{
			name:    "the handler takes the empty interface",
			expected: "first argument does not implement context.Context: got interface {}",
			handler: func(v any) error {
				if _, ok := v.(context.Context); ok {
					return errors.New("v should not be a Context")
				}
				return nil
			},
		},
		{
			name:     "the handler takes a same interface with context.Context",
			expected: "",
			handler: func(ctx myContext) error {
				return nil
			},
		},
		{
			name:     "the handler takes a superset of context.Context",
			expected: "cannot be assigned to expected parameter type sigil.customContext",
			handler: func(ctx customContext) error {
				return nil
			},
		},
		{
			name:     "the handler takes two arguments and first argument is a same interface with context.Context",
			expected: "",
			handler: func(ctx myContext, v any) error {
				return nil
			},
		},
		{
			name:     "the handler takes two arguments and first argument is a superset of context.Context",
			expected: "cannot be assigned to expected parameter type sigil.customContext",
			handler: func(ctx customContext, v any) error {
				return nil
			},
		},
	}

	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("testCase[%d] %s", i, testCase.name), func(t *testing.T) {
			t.Helper()
			handler := NewHandler(testCase.handler)
			_, err := handler.Invoke(context.TODO(), []byte("{}"))
			if testCase.expected == "" {
				assert.NilError(t, err)
			} else {
				assert.ErrorContains(t, err, testCase.expected)
			}
		})
	}
}
