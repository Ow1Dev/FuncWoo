# FuncWoo
FuncWoo is a function framework that allow your to build function. The framework is designed to be simple and easy to use, allowing you to focus on building your function without worrying about the underlying infrastructure.

# Development

## setup

To start the we need some folders and files:
```bash
./scripts/setup.sh
```

after that we need to build a exmaple function:
```bash
./scripts/build-func.sh
```

now we are ready to start develop

## Run the ignite server

To run the ignite server, you need to run the following command:
```bash
go run ./cmd/igniterelay/main.go
```

# Register a grpc service
To regenerate the grpc service, you need to run the following command:
```bash
just generate
```
