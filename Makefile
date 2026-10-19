BINARY := bin/revenger-clientless

.PHONY: build run test vet fmt clean

build:
	go build -trimpath -ldflags "-s -w" -o $(BINARY) ./cmd/revenger-clientless

run:
	go run ./cmd/revenger-clientless

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -rf bin
