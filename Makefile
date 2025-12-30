DOCKER ?= docker
DOCKERCOMPOSE ?= docker compose

#- Adjust this such that the context represents the root of the repository
DOCKERCONTEXT ?= .

.PHONY: all build-backend run-backend

all: build-backend

build-backend:
	@echo "Building the backend..."
	$(DOCKERCOMPOSE) -f $(DOCKERCONTEXT)/deployments/docker-compose.yml up --build

run-backend:
	@echo "Running the backend..."
	$(DOCKERCOMPOSE) -f $(DOCKERCONTEXT)/deployments/docker-compose.yml up