package sigil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	pb "github.com/Ow1Dev/NoctiFunc/pkgs/api/server"
)

var (
	ErrNoHandler = errors.New("no handler provided") 
	ErrInvalidHandler = errors.New("invalid handler type, must be a function")
	ErrTooManyArguments = errors.New("handler function must accept 0, 1, or 2 arguments")
	ErrTooManyReturns = errors.New("handler function must return 0, 1, or 2 values")
	ErrHandlerSingleMissingError = errors.New("handler returns a single value, but it does not implement error")
	ErrHandlerTwoMissingError = errors.New("handler returns two values, but the second does not implement error")
	ErrHandlerContextMismatch = errors.New("handler's first argument must be context.Context or compatible with it")
	ErrFirstArgNotContext = errors.New("handler takes two arguments, but the first is not context.Context")
	ErrFirstParamMustBeContext = errors.New("first parameter must be context.Context")
	ErrInvalidPayload = errors.New("invalid payload, must be a valid JSON object or empty")
)

type Handler struct {
	handlerValue reflect.Value
	handlerType  reflect.Type
	numIn        int
	numOut       int
}

// NewHandler creates and validates a new handler
func NewHandler(handler any) (*Handler, error) { 
	if handler == nil {
		return nil, ErrNoHandler
	}

	handlerValue := reflect.ValueOf(handler)
	handlerType := handlerValue.Type()

	if handlerType.Kind() != reflect.Func {
		return nil, ErrInvalidHandler
	}

	numIn := handlerType.NumIn()
	if numIn > 2 {
		return nil, ErrTooManyArguments
	}

	numOut := handlerType.NumOut()
	if numOut > 2 {
		return nil, ErrTooManyReturns
	}

	// Validate return types
	if numOut == 1 {
		returnType := handlerType.Out(0)
		errorInterface := reflect.TypeOf((*error)(nil)).Elem()
		if !returnType.Implements(errorInterface) {
			return nil, ErrHandlerSingleMissingError
		}
	} else if numOut == 2 {
		// Second return value must implement error
		secondReturnType := handlerType.Out(1)
		errorInterface := reflect.TypeOf((*error)(nil)).Elem()
		if !secondReturnType.Implements(errorInterface) {
			return nil, ErrHandlerTwoMissingError
		}
	}

	// If handler takes arguments, validate them
	if numIn > 0 {
		firstArgType := handlerType.In(0)
		
		// Check if first argument is context.Context or compatible
		contextInterface := reflect.TypeOf((*context.Context)(nil)).Elem()
		
		if firstArgType.Kind() == reflect.Interface {
			// Check if it's exactly context.Context or has the same methods
			if !isContextCompatible(firstArgType, contextInterface) {
				if numIn == 2 {
					return nil, ErrFirstArgNotContext
				} else {
					return nil, ErrFirstParamMustBeContext
				}
			}
		} else if !firstArgType.Implements(contextInterface) {
			if numIn == 2 {
				return nil, ErrFirstArgNotContext
			} else {
				return nil, ErrFirstParamMustBeContext
			}
		}
	}

	return &Handler{
		handlerValue: handlerValue,
		handlerType:  handlerType,
		numIn:        numIn,
		numOut:       numOut,
	}, nil
}

func (h *Handler) Invoke(ctx context.Context, payload []byte) (*pb.InvokeResult, error) {
	var args []reflect.Value

	if h.numIn == 0 {
	} else if h.numIn == 1 {
		args = append(args, reflect.ValueOf(ctx))
	} else if h.numIn == 2 {
		args = append(args, reflect.ValueOf(ctx))
		argType := h.handlerType.In(1)

		if argType == reflect.TypeOf((*any)(nil)).Elem() {
			var inputValue any
			if len(payload) > 0 {
				err := json.Unmarshal(payload, &inputValue)
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal request body: %w", ErrInvalidPayload)
				}
			}
			args = append(args, reflect.ValueOf(inputValue))
		} else {
			var inputValue reflect.Value
			if argType.Kind() == reflect.Ptr {
				inputValue = reflect.New(argType.Elem()) // *T
			} else {
				inputValue = reflect.New(argType) // *T
			}

			// Unmarshal JSON into inputValue (which is a pointer)
			if len(payload) > 0 {
				err := json.Unmarshal(payload, inputValue.Interface())
				if err != nil {
					return nil, fmt.Errorf("failed to unmarshal request body: %w", ErrInvalidPayload)
				}
			}

			// If handler expects value, dereference pointer
			if argType.Kind() != reflect.Ptr {
				inputValue = inputValue.Elem()
			}

			args = append(args, inputValue)
		}
	}

	results := h.handlerValue.Call(args)

	// Expected returns: 
	// () - no return
	// (error) - single error return
	// (Response, error) - response and error
	if len(results) == 0 {
		// No return values
		return &pb.InvokeResult{Output: "{}"}, nil
	} else if len(results) == 1 {
		// Only error returned
		errInterface := results[0].Interface()
		if errInterface != nil {
			return nil, errInterface.(error)
		}
		// no response
		return &pb.InvokeResult{Output: "{}"}, nil
	} else if len(results) == 2 {
		// First is response, second is error
		errInterface := results[1].Interface()
		if errInterface != nil {
			return nil, errInterface.(error)
		}
		respInterface := results[0].Interface()

		// Marshal the response
		respJSON, err := json.Marshal(respInterface)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}
		return &pb.InvokeResult{Output: string(respJSON)}, nil
	} else {
		return nil, fmt.Errorf("handler returned unexpected number of values: %d", len(results))
	}
}

// isContextCompatible checks if an interface type is compatible with context.Context
func isContextCompatible(interfaceType, contextType reflect.Type) bool {
	if interfaceType.Kind() != reflect.Interface {
		return false
	}

	contextMethods := make(map[string]reflect.Type)
	for i := range contextType.NumMethod() {
		method := contextType.Method(i)
		contextMethods[method.Name] = method.Type
	}

	interfaceMethods := make(map[string]reflect.Type)
	for i := range interfaceType.NumMethod() {
		method := interfaceType.Method(i)
		interfaceMethods[method.Name] = method.Type
	}

	// Check if the interface has exactly the same methods as context.Context
	if len(interfaceMethods) != len(contextMethods) {
		return false
	}

	// For it to be compatible, it should have the same methods as context.Context
	// or be a superset (which we'll reject) or subset (which we'll also reject)
	for name, methodType := range contextMethods {
		if interfaceMethodType, exists := interfaceMethods[name]; !exists || !methodType.AssignableTo(interfaceMethodType) {
			return false
		}
	}

	return true
}
