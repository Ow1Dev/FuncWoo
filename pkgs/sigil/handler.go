package sigil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
)

type Handler interface {
	Invoke(ctx context.Context, payload []byte) ([]byte, error)
}

type handlerFunc func(context.Context, []byte) (io.Reader, error)

type handlerOptions struct {
	handlerFunc
	baseContext  context.Context
}

type Option func(*handlerOptions)

// WithContext sets a custom base context for the handler.
func WithContext(ctx context.Context) Option {
	return func(h *handlerOptions) {
		h.baseContext = ctx
	}
}

func NewHandler(handlerFunc any) Handler {
	return NewHandlerWithOptions(handlerFunc)
}

func NewHandlerWithOptions(handlerFunc any, options ...Option) Handler {
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
		defer closer.Close()
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
	switch t.NumIn() {
	case 0:
		return false, nil
	case 1, 2:
		arg := t.In(0)
		ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
		if !arg.Implements(ctxType) {
			return false, fmt.Errorf("first argument does not implement context.Context: got %v", arg)
		}
		return true, nil
	default:
		return false, fmt.Errorf("handler has too many parameters: %d", t.NumIn())
	}
}

func validateReturnTypes(t reflect.Type) error {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	switch t.NumOut() {
	case 1:
		if !t.Out(0).Implements(errType) {
			return fmt.Errorf("single return must be error, got %v", t.Out(0))
		}
	case 2:
		if !t.Out(1).Implements(errType) {
			return fmt.Errorf("second return must be error, got %v", t.Out(1))
		}
	case 0:
		return fmt.Errorf("handler must return at least an error")
	default:
		return fmt.Errorf("too many return values: %d", t.NumOut())
	}
	return nil
}

// jsonOutBuffer is used to avoid reallocation for JSON-encoded responses.
type jsonOutBuffer struct {
	*bytes.Buffer
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

	out := &jsonOutBuffer{Buffer: new(bytes.Buffer)}

	return func(ctx context.Context, payload []byte) (io.Reader, error) {
		out.Reset()

		var args []reflect.Value
		if takesCtx {
			expectedCtxType := typ.In(0)

			// Check if ctx can be assigned to expectedCtxType
			if !reflect.TypeOf(ctx).AssignableTo(expectedCtxType) {
				return nil, fmt.Errorf("provided context of type %T cannot be assigned to expected parameter type %v", ctx, expectedCtxType)
			}

			args = append(args, reflect.ValueOf(ctx))
		}

		// Prepare input arguments
		paramIndex := 0
		if takesCtx {
			paramIndex = 1
		}

		if paramIndex < typ.NumIn() {
			eventType := typ.In(paramIndex)
			event := reflect.New(eventType).Interface()

			if err := json.Unmarshal(payload, event); err != nil {
				return nil, fmt.Errorf("failed to decode input: %w", err)
			}
			args = append(args, reflect.ValueOf(event).Elem())
		}

		// Invoke the function
		results := val.Call(args)

		// Handle errors
		last := results[len(results)-1].Interface()
		if err, ok := last.(error); ok && err != nil {
			return nil, err
		}

		var resp any
		if len(results) > 1 {
			resp = results[0].Interface()
		}

		// Encode result to JSON
		encoder := json.NewEncoder(out)
		if err := encoder.Encode(resp); err != nil {
			// Allow returning io.Reader directly
			if reader, ok := resp.(io.Reader); ok {
				return reader, nil
			}
			return nil, fmt.Errorf("response encoding failed: %w", err)
		}

		return out, nil
	}
}

