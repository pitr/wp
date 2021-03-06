.PHONY: clean test check run build.local build.linux build.osx build.docker deploy

BINARY        ?= wp
SOURCES       = $(shell find . -name '*.go') tmpl.go
STATICS       = $(shell find tmpl -name '*.*')
VERSION       ?= $(shell git describe --tags --always)
IMAGE         ?= deploy.glv.one/pitr/$(BINARY)
TAG           ?= $(VERSION)
DOCKERFILE    ?= Dockerfile
BUILD_FLAGS   ?= -v
LDFLAGS       ?= -w -s -X main.version=$(VERSION)

default: run

clean:
	rm -rf build

run: build.local
	./build/$(BINARY)

tmpl.go: $(STATICS)
	go run cmd/build_tmpl.go $(STATICS)

build.local: build/$(BINARY)
build.linux: build/linux/$(BINARY)

build/$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

build/linux/$(BINARY): $(SOURCES)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/linux/$(BINARY) -ldflags "$(LDFLAGS)" .

build.docker: build.linux
	docker build --rm -t "$(IMAGE):$(TAG)" -f $(DOCKERFILE) .

deploy: build.docker
	docker push "$(IMAGE):$(TAG)"
