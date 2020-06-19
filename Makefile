.PHONY: clean run deploy build.local build.linux

BINARY        ?= wp
SOURCES       = $(shell find . -name '*.go') tmpl.go
STATICS       = $(shell find tmpl -name '*.*')
BUILD_FLAGS   ?= -v
LDFLAGS       ?= -w -s

default: run

clean:
	rm -rf build

run: build.local
	./build/$(BINARY)

deploy: build.linux
	scp build/linux/$(BINARY) ec2-user@$(PRODUCTION):$(BINARY)/$(BINARY)-next
	ssh ec2-user@$(PRODUCTION) 'mv $(BINARY)/$(BINARY) $(BINARY)/$(BINARY)-old'
	ssh ec2-user@$(PRODUCTION) 'mv $(BINARY)/$(BINARY)-next $(BINARY)/$(BINARY)'
	ssh ec2-user@$(PRODUCTION) 'sudo systemctl restart $(BINARY)'

rollback:
	ssh ec2-user@$(PRODUCTION) 'mv $(BINARY)/$(BINARY)-old $(BINARY)/$(BINARY)'
	ssh ec2-user@$(PRODUCTION) 'sudo systemctl restart $(BINARY)'

tmpl.go: $(STATICS)
	go run cmd/build_tmpl.go $(STATICS)

build.local: build/$(BINARY)
build.linux: build/linux/$(BINARY)

build/$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

build/linux/$(BINARY): $(SOURCES)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/linux/$(BINARY) -ldflags "$(LDFLAGS)" .
