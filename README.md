# CRYPTO-SSS

User oriented mock application that demonstrates the utility of Secret Sharing Scheme (SSS).

Using Go (formerly known as Golang) for building the backbone of the application:

- NoSQL MongoDB storage;
- `gin-gonic` framework for setting up the server;
- cryptographic SSS from scratch;

## Backend

Developed using Go. Hot reload configuration is present in the `.air.toml` file and can be installed using:

```cmd
go install github.com/air-verse/air@latest
```

To run the binary, use the following commands:

```cmd
cd src/backend
air -c .air.toml ...more_args... -- ...your_program_arguments...

# or plainly if the config is already in the project directory
air ...more_args... -- ...your_program_arguments...
```

In order to build/run the binary, use the following commands:

```cmd
cd src/backend
go build -o ./build # builds the binary in the "./build" directory
go run ...executable_name... # runs the binary
```

## Frontend

TODO: frontend description
