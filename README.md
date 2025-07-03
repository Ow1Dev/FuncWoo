# FuncWoo

FuncWoo is a lightweight function framework for building scalable, distributed functions. It is designed to be developer-friendly, allowing you to focus on writing your function logic without worrying about infrastructure details.

---

## Core Components

### Ignite — Distributed Function Server

Ignite manages the lifecycle of your functions, including starting, stopping, and scaling. It handles distributed execution and communication using gRPC.

### Prism — API Gateway

Prism exposes your functions as RESTful APIs. It routes HTTP requests to the appropriate functions and returns responses to clients.

### Sigil — Function Definition Toolkit

Sigil is a Go package for defining and authoring functions with a simple interface, integrated seamlessly with the FuncWoo runtime.

> See the ['examples/'](./examples) directory for sample functions.

---

## Getting Started


### 1. Set Up Project Files

### Prerequisites

Ensure you have the following installed:
- Go (version 1.18 or later)
- Just (a command runner for Go projects)
- Protobuf compiler (protoc)
- gRPC plugin for Go
- docker

Initialize the required folders and files:

```bash
./scripts/setup.sh
```

### 2. Build Your Function

Compile the function and generate the necessary runtime files:

```bash
./scripts/build-func.sh
```

---

## Running Locally

### Run Both
To run both Ignite and Prism, you can use the provided `just` commands:

```bash
just run
```

### Starting Ignite and Prism Separately

#### Start Ignite

```bash
go run ./cmd/igniterelay/main.go
```

#### Start Prism

```bash
go run ./cmd/prism/main.go
```

---

## Regenerate gRPC Services

To regenerate the gRPC service files, run:

```bash
just generate
```

---

## Contributing

Contributions are welcome! Feel free to open issues or submit pull requests.

---

## License

Apache License © Ow1Dev

