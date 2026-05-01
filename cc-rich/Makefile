.PHONY: build install install-shim test clean

build:
	go build -o ./bin/cc-rich ./cmd/cc-rich

install: build
	mkdir -p $$HOME/.local/bin
	cp ./bin/cc-rich $$HOME/.local/bin/cc-rich

install-shim:
	mkdir -p $$HOME/bin
	ln -sf $$HOME/dev/dotfiles/bin/cc-rich-shim $$HOME/bin/cc-rich

test:
	go test ./...

clean:
	rm -rf ./bin
