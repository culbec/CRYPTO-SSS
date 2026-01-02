#- Makefile directory. Used to prevent redundant passing of the Makefile directory as an argument to the make command.
MakefileDir := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

#- Commands
DOCKERCOMPOSE ?= docker compose

.PHONY: all build-backend run-backend

all: build-backend

build-backend:
	@echo "Building the backend..."
	$(DOCKERCOMPOSE) -f $(MakefileDir)/deployments/docker-compose.yml up --build

run-backend:
	@echo "Running the backend..."
	$(DOCKERCOMPOSE) -f $(MakefileDir)/deployments/docker-compose.yml up