package sigil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

// Test types
type TestInput struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type TestOutput struct {
	Message string `json:"message"`
	Result  int    `json:"result"`
}

// Test functions for all valid signatures

// func ()
func testFunc1() {}

// func (TIn)
func testFunc2(input TestInput) {}

// func () error
func testFunc3() error {
	return nil
}

// func (TIn) error
func testFunc4(input TestInput) error {
	if input.Value < 0 {
		return errors.New("negative value")
	}
	return nil
}

// func () (TOut, error)
func testFunc5() (TestOutput, error) {
	return TestOutput{Message: "hello", Result: 42}, nil
}

// func (TIn) (TOut, error)
func testFunc6(input TestInput) (TestOutput, error) {
	return TestOutput{
		Message: "processed " + input.Name,
		Result:  input.Value * 2,
	}, nil
}

// func (context.Context)
func testFunc7(ctx context.Context) {}

// func (context.Context) error
func testFunc8(ctx context.Context) error {
	if ctx == nil {
		return errors.New("nil context")
	}
	return nil
}

// func (context.Context) (TOut, error)
func testFunc9(ctx context.Context) (TestOutput, error) {
	return TestOutput{Message: "from context", Result: 1}, nil
}

// func (context.Context, TIn)
func testFunc10(ctx context.Context, input TestInput) {}

// func (context.Context, TIn) error
func testFunc11(ctx context.Context, input TestInput) error {
	if input.Name == "" {
		return errors.New("empty name")
	}
	return nil
}

// func (context.Context, TIn) (TOut, error)
func testFunc12(ctx context.Context, input TestInput) (TestOutput, error) {
	return TestOutput{
		Message: "context + " + input.Name,
		Result:  input.Value + 100,
	}, nil
}

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name        string
		fn          any
		input       TestInput
		expectError bool
		checkOutput bool
		expectedOut TestOutput
	}{
		{
			name:        "func ()",
			fn:          testFunc1,
			expectError: false,
			checkOutput: false,
		},
		{
			name:        "func (TIn)",
			fn:          testFunc2,
			input:       TestInput{Name: "test", Value: 5},
			expectError: false,
			checkOutput: false,
		},
		{
			name:        "func () error",
			fn:          testFunc3,
			expectError: false,
			checkOutput: false,
		},
		{
			name:        "func (TIn) error - success",
			fn:          testFunc4,
			input:       TestInput{Name: "test", Value: 5},
			expectError: false,
			checkOutput: false,
		},
		{
			name:        "func (TIn) error - failure",
			fn:          testFunc4,
			input:       TestInput{Name: "test", Value: -1},
			expectError: true,
		},
		{
			name:        "func () (TOut, error)",
			fn:          testFunc5,
			expectError: false,
			checkOutput: true,
			expectedOut: TestOutput{Message: "hello", Result: 42},
		},
		{
			name:        "func (TIn) (TOut, error)",
			fn:          testFunc6,
			input:       TestInput{Name: "world", Value: 21},
			expectError: false,
			checkOutput: true,
			expectedOut: TestOutput{Message: "processed world", Result: 42},
		},
		{
			name:        "func (context.Context)",
			fn:          testFunc7,
			expectError: false,
			checkOutput: false,
		},
		{
			name:        "func (context.Context) error",
			fn:          testFunc8,
			expectError: false,
			checkOutput: false,
		},
		{
			name:        "func (context.Context) (TOut, error)",
			fn:          testFunc9,
			expectError: false,
			checkOutput: true,
			expectedOut: TestOutput{Message: "from context", Result: 1},
		},
		{
			name:        "func (context.Context, TIn)",
			fn:          testFunc10,
			input:       TestInput{Name: "ctx", Value: 10},
			expectError: false,
			checkOutput: false,
		},
		{
			name:        "func (context.Context, TIn) error",
			fn:          testFunc11,
			input:       TestInput{Name: "ctx", Value: 10},
			expectError: false,
			checkOutput: false,
		},
		{
			name:        "func (context.Context, TIn) error - failure",
			fn:          testFunc11,
			input:       TestInput{Name: "", Value: 10},
			expectError: true,
		},
		{
			name:        "func (context.Context, TIn) (TOut, error)",
			fn:          testFunc12,
			input:       TestInput{Name: "final", Value: 5},
			expectError: false,
			checkOutput: true,
			expectedOut: TestOutput{Message: "context + final", Result: 105},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newHandler(tt.fn)

			var payload []byte
			var err error
			if tt.input != (TestInput{}) {
				payload, err = json.Marshal(tt.input)
				if err != nil {
					t.Fatalf("failed to marshal input: %v", err)
				}
			}

			result, err := handler.Invoke(context.Background(), payload)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.checkOutput && !tt.expectError {
				var output TestOutput
				if err := json.Unmarshal(result, &output); err != nil {
					t.Fatalf("failed to unmarshal output: %v", err)
				}
				if output != tt.expectedOut {
					t.Errorf("expected %+v, got %+v", tt.expectedOut, output)
				}
			}
		})
	}
}

func TestValidSingleParamHandlers(t *testing.T) {
	tests := []struct {
		name string
		fn   any
	}{
		{"func(string)", func(s string) {}},
		{"func(int)", func(i int) {}},
		{"func(struct)", func(input TestInput) {}},
		{"func(string) error", func(s string) error { return nil }},
		{"func(int) (string, error)", func(i int) (string, error) { return "", nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newHandler(tt.fn)

			// Test with empty payload (should work for most cases)
			_, err := handler.Invoke(context.Background(), []byte("{}"))

			// We don't expect an error from the handler creation itself
			// The specific function might return an error, but the handler should be valid
			if err != nil {
				// Check if it's a JSON unmarshal error, which is expected for primitive types
				if !strings.Contains(err.Error(), "decode input") {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestInvalidHandlers(t *testing.T) {
	tests := []struct {
		name string
		fn   any
	}{
		{"nil function", nil},
		{"not a function", "not a function"},
		{"too many params", func(a, b, c, d int) {}},
		{"no error return", func() int { return 0 }},
		{"too many returns", func() (int, string, error) { return 0, "", nil }},
		{"wrong context type", func(s string, input TestInput) error { return nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newHandler(tt.fn)
			_, err := handler.Invoke(context.Background(), nil)
			if err == nil {
				t.Error("expected error for invalid handler")
			}
		})
	}
}

type ctxKey string

func TestWithContext(t *testing.T) {
	const testKey ctxKey = "key"

	customCtx := context.WithValue(context.Background(), testKey, "value")

	handler := newHandlerWithOptions(func(ctx context.Context) (TestOutput, error) {
		value := ctx.Value(testKey)
		if value == nil {
			return TestOutput{}, errors.New("context value not found")
		}
		return TestOutput{Message: value.(string), Result: 1}, nil
	}, WithContext(customCtx))

	result, err := handler.Invoke(customCtx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output TestOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	expected := TestOutput{Message: "value", Result: 1}
	if output != expected {
		t.Errorf("expected %+v, got %+v", expected, output)
	}
}

func TestHandlerReturnsReader(t *testing.T) {
	handler := newHandler(func() (io.Reader, error) {
		return strings.NewReader("direct reader response"), nil
	})

	result, err := handler.Invoke(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != "direct reader response" {
		t.Errorf("expected 'direct reader response', got %q", string(result))
	}
}

func TestJSONUnmarshalError(t *testing.T) {
	handler := newHandler(func(input TestInput) error {
		return nil
	})

	// Invalid JSON
	_, err := handler.Invoke(context.Background(), []byte(`{"invalid": json}`))
	if err == nil {
		t.Error("expected JSON unmarshal error")
	}
}

func TestHandlerInvokeMethod(t *testing.T) {
	// Test that handlerFunc implements Handler interface correctly
	hf := handlerFunc(func(ctx context.Context, payload []byte) (io.Reader, error) {
		return bytes.NewReader([]byte(`{"test": true}`)), nil
	})

	result, err := hf.Invoke(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `{"test": true}`
	if string(result) != expected {
		t.Errorf("expected %q, got %q", expected, string(result))
	}
}

type testCloser struct {
	io.Reader
	closed bool
}

func (tc *testCloser) Close() error {
	tc.closed = true
	return nil
}

func TestCloserInterface(t *testing.T) {

	var tc *testCloser
	handler := newHandler(func() (io.Reader, error) {
		tc = &testCloser{Reader: strings.NewReader("test")}
		return tc, nil
	})

	_, err := handler.Invoke(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !tc.closed {
		t.Error("expected closer to be closed")
	}
}

func TestZeroValueHandling(t *testing.T) {
	tests := []struct {
		name     string
		fn       any
		expected string
	}{
		{
			name:     "no return values",
			fn:       func() {},
			expected: "null",
		},
		{
			name:     "nil return",
			fn:       func() (any, error) { return nil, nil },
			expected: "null\n",
		},
		{
			name:     "empty struct",
			fn:       func() (TestOutput, error) { return TestOutput{}, nil },
			expected: "{\"message\":\"\",\"result\":0}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newHandler(tt.fn)
			result, err := handler.Invoke(context.Background(), nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}

func TestContextPassing(t *testing.T) {
	type testCtxKey string
	const key testCtxKey = "test"

	ctx := context.WithValue(context.Background(), key, "test-value")

	handler := newHandler(func(ctx context.Context, input TestInput) (TestOutput, error) {
		value, ok := ctx.Value(key).(string)
		if !ok {
			return TestOutput{}, errors.New("context value not found")
		}
		return TestOutput{Message: value, Result: input.Value}, nil
	})

	input := TestInput{Name: "test", Value: 42}
	payload, _ := json.Marshal(input)

	result, err := handler.Invoke(ctx, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output TestOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	expected := TestOutput{Message: "test-value", Result: 42}
	if output != expected {
		t.Errorf("expected %+v, got %+v", expected, output)
	}
}
