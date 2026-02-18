.PHONY: build test install clean

build:
	go build -o bin/pvault ./cmd/pvault

test:
	go test -v -race ./...

install: build
	install -m 755 bin/pvault /usr/local/bin/pvault

clean:
	rm -rf bin/
