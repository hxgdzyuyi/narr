.PHONY: build install test smoke clean

BINARY := narrc
CMD := ./cmd/narrc

build:
	go build -o bin/$(BINARY) $(CMD)

install:
	go install $(CMD)

test:
	go test ./...

smoke:
	go run $(CMD) --version

clean:
	rm -rf bin
