package sigil

import "log"

// Start initializes the Sigil runtime and starts processing requests using the provided handler.
//
//
// Valid function signatures:
//
//	func ()
//	func (TIn)
//	func () error
//	func (TIn) error
//	func () (TOut, error)
//	func (TIn) (TOut, error)
//	func (context.Context)
//	func (context.Context) error
//	func (context.Context) (TOut, error)
//	func (context.Context, TIn)
//	func (context.Context, TIn) error
//	func (context.Context, TIn) (TOut, error)
func Start(handler any) {
	StartWithOptions(handler)
}

// StartWithOptions is the same as Start after the application of any handler options specified
func StartWithOptions(handler any, options ...Option) {
	start(newHandler(handler, options...))
}

var (
	logFatalf = log.Fatalf
)

func start(handler *handlerOptions) {
	err := startRuntimeGRPCLoop(handler)
	logFatalf("%v", err)
}
