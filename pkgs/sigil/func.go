package sigil

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"reflect"

	pb "github.com/Ow1Dev/NoctiFunc/pkgs/api/server"
	"google.golang.org/grpc"
)

type serviceServer struct {
	pb.UnimplementedFunctionRunnerServiceServer
	handler any
}

func (s *serviceServer) Invoke(ctx context.Context, req *pb.InvokeRequest) (*pb.InvokeResult, error) {
    handlerValue := reflect.ValueOf(s.handler)
    handlerType := handlerValue.Type()

    // Expect handler to be a func with 1 or 2 inputs: (context.Context, Request)
    if handlerType.Kind() != reflect.Func {
        return nil, fmt.Errorf("handler is not a function")
    }

    // Number of inputs: 1 or 2
    numIn := handlerType.NumIn()
    if numIn != 1 && numIn != 2 {
        return nil, fmt.Errorf("handler function must have 1 or 2 input parameters")
    }

    // First parameter must be context.Context
    if !handlerType.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
        return nil, fmt.Errorf("first parameter must be context.Context")
    }

    var args []reflect.Value
    args = append(args, reflect.ValueOf(ctx))

    if numIn == 2 {
        // Create a new instance of the second argument type
        argType := handlerType.In(1)
        // Pointer or value?
        var inputValue reflect.Value
        if argType.Kind() == reflect.Ptr {
            inputValue = reflect.New(argType.Elem()) // *T
        } else {
            inputValue = reflect.New(argType) // *T
        }

        // Unmarshal JSON into inputValue (which is a pointer)
        err := json.Unmarshal([]byte(req.GetPayload()), inputValue.Interface())
        if err != nil {
            return nil, fmt.Errorf("failed to unmarshal request body: %w", err)
        }

        // If handler expects value, dereference pointer
        if argType.Kind() != reflect.Ptr {
            inputValue = inputValue.Elem()
        }

        args = append(args, inputValue)
    }

    // Call the handler
    results := handlerValue.Call(args)

    // Expected returns: either
    // (Response, error)
    // or (error)
    if len(results) == 2 {
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

    } else if len(results) == 1 {
        // Only error returned
        errInterface := results[0].Interface()
        if errInterface != nil {
            return nil, errInterface.(error)
        }
        // no response
        return &pb.InvokeResult{Output: "{}"}, nil

    } else {
        return nil, fmt.Errorf("handler returned unexpected number of values: %d", len(results))
    }
}

func Start(handler any) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 8080))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
	}

	s := grpc.NewServer()
	pb.RegisterFunctionRunnerServiceServer(s, &serviceServer{
		handler: handler,
	})

	fmt.Printf("Gateway server listening on %s\n", lis.Addr().String())
	if err := s.Serve(lis); err != nil {
		fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
	}
}
