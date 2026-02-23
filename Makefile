BINARY := health
SRC    := ./cmd

.PHONY: build clean

build:
	go build -o $(BINARY) $(SRC)

clean:
	rm -f $(BINARY)
