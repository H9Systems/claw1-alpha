BINARY  := claw1
INSTALL := /usr/local/bin/$(BINARY)
VERSION ?= v$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build install clean

build:
	go build $(LDFLAGS) -o $(BINARY) .

install: build
	sudo cp $(BINARY) $(INSTALL)
	@echo "Installed to $(INSTALL)"

clean:
	rm -f $(BINARY)
