APP=synapse
CMD=./
BIN=./bin

# Go packages we own — excludes frontend/node_modules, which contains a stray
# third-party Go package that `./...` would otherwise pick up.
PKGS=. ./configs/... ./internal/... ./tests/...

.PHONY: run build test fmt vet tidy clean

build:
	go build -o $(BIN)/$(APP) $(CMD)

run: build
	$(BIN)/$(APP)

test:
	go test $(PKGS) -v

fmt:
	go fmt $(PKGS)

vet:
	go vet $(PKGS)

tidy:
	go mod tidy

clean:
ifeq ($(OS),Windows_NT)
	if exist bin rmdir /s /q bin
else
	rm -rf bin
endif