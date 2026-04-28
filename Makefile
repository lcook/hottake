.POSIX:
PROG=		hottake

CONFIG?=	./config.example.yaml

HASH!=		git rev-parse --short HEAD 2>/dev/null
VERSION:=	0.1.0-$(HASH)

GO_MODULE=	github.com/lcook/${PROG}
GO_FLAGS=  	-v -ldflags "-s -w -X ${GO_MODULE}/internal/version.Build=${VERSION}"

default: build
build:
	go build $(GO_FLAGS) -o $(PROG) cmd/$(PROG)/$(PROG).go && strip -s $(PROG)

clean:
	rm -f $(PROG)
	go clean

container:
	podman build -t $(PROG) .

run-container:
	podman run -v $(CONFIG):/app/config.yaml localhost/$(PROG) /app/$(PROG) -V 2

update:
	go get -u -v ./...
	go mod tidy -v

lint:
	golangci-lint run

.PHONY: build clean container run-container update
