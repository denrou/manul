.PHONY: build test vet tidy run clean

build:
	go build -o manul ./cmd/manul

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

run: build
	./manul

clean:
	rm -f manul coverage.out
