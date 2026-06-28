APP=synapse
CMD=./
BIN=./bin

.PHONY: run build test fmt vet tidy clean

build:
	go build -o $(BIN)/$(APP) $(CMD)

run: build
	$(BIN)/$(APP)

test:
	go test ./... -v

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
ifeq ($(OS),Windows_NT)
	if exist bin rmdir /s /q bin
else
	rm -rf bin
endif