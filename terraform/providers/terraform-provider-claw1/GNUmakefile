BINARY   := terraform-provider-claw1
VERSION  := 0.1.0
OS       := $(shell go env GOOS)
ARCH     := $(shell go env GOARCH)
PLUGIN_DIR := $(HOME)/.terraform.d/plugins/local/h9-systems/claw1/$(VERSION)/$(OS)_$(ARCH)

.PHONY: build test install clean

build:
	go build -o $(BINARY) .

test:
	go test ./... -v

install: build
	mkdir -p $(PLUGIN_DIR)
	cp $(BINARY) $(PLUGIN_DIR)/

clean:
	rm -f $(BINARY)
