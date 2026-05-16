BINARY  := claw1
INSTALL := /usr/local/bin/$(BINARY)

.PHONY: build install clean

build:
	go build -o $(BINARY) .

install: build
	sudo cp $(BINARY) $(INSTALL)
	@echo "Installed to $(INSTALL)"

clean:
	rm -f $(BINARY)
