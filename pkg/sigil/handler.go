package sigil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
)

type handler interface {
	Invoke(ctx context.Context, payload []byte) ([]byte, error)
}

type handlerFunc func(context.Context, []byte) (io.Reader, error)

type handlerOptions struct {
	handlerFunc
	baseContext context.Context
}

type Option func(*handlerOptions)

// WithContext sets a custom base context for the handler.
func WithContext(ctx context.Context) Option {
	return func(h *handlerOptions) {
		h.baseContext = ctx
	}
}

func newHandlerWithOptions(handlerFunc any, options ...Option) handler {
	return newHandler(handlerFunc, options...)
}

// newHandler constructs a handlerOptions object and wraps a function as a handler.
func newHandler(fn any, opts ...Option) *handlerOptions {
	if h, ok := fn.(*handlerOptions); ok {
		return h
	}

	h := &handlerOptions{
		baseContext: context.Background(),
	}

	for _, opt := range opts {
		opt(h)
	}

	h.handlerFunc = wrapHandler(fn)
	return h
}

func (h handlerFunc) Invoke(ctx context.Context, payload []byte) ([]byte, error) {
	resp, err := h(ctx, payload)
	if err != nil {
		return nil, err
	}

	// Cleanup resources if response implements io.Closer
	if closer, ok := resp.(io.Closer); ok {
		defer func() {
			if err := closer.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "error closing response: %v\n", err)
			}
		}()
	}

	// Fast-path if it's already a bytes.Buffer or jsonOutBuffer
	switch b := resp.(type) {
	case *jsonOutBuffer:
		return b.Bytes(), nil
	case *bytes.Buffer:
		return b.Bytes(), nil
	default:
		return io.ReadAll(resp)
	}
}

func errorHandler(err error) handlerFunc {
	return func(_ context.Context, _ []byte) (io.Reader, error) {
		return nil, err
	}
}

func handlerTakesContext(t reflect.Type) (bool, error) {
	if t.NumIn() == 0 {
		return false, nil
	}

	if t.NumIn() > 2 {
		return false, fmt.Errorf("handler has too many parameters: %d", t.NumIn())
	}

	// Check first parameter
	arg := t.In(0)
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()

	if arg.Implements(ctxType) {
		// First parameter is context.Context
		return true, nil
	}

	// If first parameter is not context.Context, it must be the input type
	// and we can only have 1 parameter total
	if t.NumIn() > 1 {
		return false, fmt.Errorf("when first argument is not context.Context, handler can only have 1 parameter: got %d parameters, first is %v", t.NumIn(), arg)
	}

	// Single parameter that is not context.Context - this is valid (input type)
	return false, nil
}

func validateReturnTypes(t reflect.Type) error {
	errType := reflect.TypeOf((*error)(nil)).Elem()

	switch t.NumOut() {
	case 0:
		// func() or func(TIn) - allowed
		return nil
	case 1:
		// Must be error: func() error or func(TIn) error
		if !t.Out(0).Implements(errType) {
			return fmt.Errorf("single return must be error, got %v", t.Out(0))
		}
	case 2:
		// Must be (TOut, error)
		if !t.Out(1).Implements(errType) {
			return fmt.Errorf("second return must be error, got %v", t.Out(1))
		}
	default:
		return fmt.Errorf("too many return values: %d", t.NumOut())
	}
	return nil
}

// jsonOutBuffer is used to avoid reallocation for JSON-encoded responses.
// It's pooled for better memory efficiency.
type jsonOutBuffer struct {
	*bytes.Buffer
}

var bufferPool = sync.Pool{
	New: func() any {
		return &jsonOutBuffer{Buffer: new(bytes.Buffer)}
	},
}

func getBuffer() *jsonOutBuffer {
	buf := bufferPool.Get().(*jsonOutBuffer)
	buf.Reset()
	return buf
}

func putBuffer(buf *jsonOutBuffer) {
	if buf.Cap() < 64*1024 { // Don't pool very large buffers
		bufferPool.Put(buf)
	}
}

// wrapHandler converts a user-defined function to a handlerFunc.
func wrapHandler(fn any) handlerFunc {
	if fn == nil {
		return errorHandler(errors.New("handler function is nil"))
	}

	val := reflect.ValueOf(fn)
	typ := reflect.TypeOf(fn)

	if typ.Kind() != reflect.Func {
		return errorHandler(fmt.Errorf("expected a function, got %v", typ.Kind()))
	}

	takesCtx, err := handlerTakesContext(typ)
	if err != nil {
		return errorHandler(err)
	}

	if err := validateReturnTypes(typ); err != nil {
		return errorHandler(err)
	}

	return func(ctx context.Context, payload []byte) (io.Reader, error) {
		// Prepare arguments
		args := make([]reflect.Value, 0, 2)
		paramIndex := 0

		// Add context if required
		if takesCtx {
			expectedCtxType := typ.In(0)
			if !reflect.TypeOf(ctx).AssignableTo(expectedCtxType) {
				return nil, fmt.Errorf("provided context of type %T cannot be assigned to expected parameter type %v", ctx, expectedCtxType)
			}
			args = append(args, reflect.ValueOf(ctx))
			paramIndex = 1
		}

		// Add input parameter if required
		if paramIndex < typ.NumIn() {
			eventType := typ.In(paramIndex)
			event := reflect.New(eventType)

			if len(payload) > 0 {
				if err := json.Unmarshal(payload, event.Interface()); err != nil {
					return nil, fmt.Errorf("failed to decode input: %w", err)
				}
			}
			args = append(args, event.Elem())
		}

		// Invoke the function
		results := val.Call(args)

		// Handle return values
		if len(results) == 0 {
			// No return values - return null
			return bytes.NewReader([]byte("null")), nil
		}

		// Check for error (always last return value if present)
		if len(results) >= 1 {
			if errVal := results[len(results)-1]; !errVal.IsNil() {
				if err, ok := errVal.Interface().(error); ok {
					return nil, err
				}
			}
		}

		// Handle response value
		var resp any
		if len(results) == 2 {
			// (TOut, error) case
			resp = results[0].Interface()
		} else if len(results) == 1 {
			// Check if single return is error or value
			if results[0].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				// Single error return - no output value
				return bytes.NewReader([]byte("null")), nil
			} else {
				// Single value return (shouldn't happen with current validation, but handle gracefully)
				resp = results[0].Interface()
			}
		}

		// Handle direct io.Reader responses
		if reader, ok := resp.(io.Reader); ok {
			return reader, nil
		}

		// JSON encode the response
		buf := getBuffer()
		encoder := json.NewEncoder(buf)
		if err := encoder.Encode(resp); err != nil {
			putBuffer(buf)
			return nil, fmt.Errorf("response encoding failed: %w", err)
		}

		// Return buffer but don't put it back in pool yet (caller will read from it)
		return &jsonOutBufferReader{buf}, nil
	}
}

// jsonOutBufferReader wraps jsonOutBuffer to handle proper cleanup
type jsonOutBufferReader struct {
	*jsonOutBuffer
}

func (r *jsonOutBufferReader) Read(p []byte) (n int, err error) {
	return r.jsonOutBuffer.Read(p)
}

func (r *jsonOutBufferReader) Close() error {
	putBuffer(r.jsonOutBuffer)
	return nil
}

// Ensure jsonOutBufferReader implements io.ReadCloser
var _ io.ReadCloser = (*jsonOutBufferReader)(nil)
