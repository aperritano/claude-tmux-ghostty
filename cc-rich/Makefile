.PHONY: build install install-shim test clean

# Ad-hoc codesign on macOS: this enterprise Mac silently SIGKILLs unsigned
# arm64 Mach-O binaries on exec (exit 137). `codesign --sign -` is enough
# to satisfy whichever security daemon is policing it. No-op on other OSes.
build:
	go build -o ./bin/cc-rich ./cmd/cc-rich
	@if [ "$$(uname)" = "Darwin" ]; then codesign --force --sign - ./bin/cc-rich >/dev/null 2>&1 || true; fi

install: build
	mkdir -p $$HOME/.local/bin
	cp ./bin/cc-rich $$HOME/.local/bin/cc-rich
	@if [ "$$(uname)" = "Darwin" ]; then codesign --force --sign - $$HOME/.local/bin/cc-rich >/dev/null 2>&1 || true; fi

install-shim:
	mkdir -p $$HOME/bin
	ln -sf $$HOME/dev/dotfiles/bin/cc-rich-shim $$HOME/bin/cc-rich

test:
	go test ./...

clean:
	rm -rf ./bin
