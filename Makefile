# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)

# Used to populate version variable in main package.
VERSION=$(shell git describe --match 'v[0-9]*' --dirty='.m' --always)

# Allow turning off function inlining and variable registerization
ifeq (${DISABLE_OPTIMIZATION},true)
	GO_GCFLAGS=-gcflags "-N -l"
	VERSION:="$(VERSION)-noopt"
endif

GO_LDFLAGS=-ldflags "-X `go list ./version`.Version=$(VERSION)"

.PHONY: clean all fmt vet lint build test binaries
.DEFAULT: default
#all: AUTHORS clean fmt vet lint build test binaries
all: AUTHORS clean fmt build test binaries

AUTHORS: .mailmap .git/HEAD
	git log --format='%aN <%aE>' | sort -fu > $@

# This only needs to be generated by hand when cutting full releases.
version/version.go:
	./version/version.sh > $@

# This only needs to be generated by hand when the protocol has been changed.
protocol/builder.pb.go: protocol/builder.proto
	protoc --go_out=plugins=grpc:. $<

${PREFIX}/bin/builder-master: version/version.go $(shell find . -type f -name '*.go')
	@echo "+ $@"
	@go build -tags "${BUILDER_BUILDTAGS}" -o $@ ${GO_LDFLAGS}  ${GO_GCFLAGS} ./cmd/master

${PREFIX}/bin/builder-slave: version/version.go $(shell find . -type f -name '*.go')
	@echo "+ $@"
	@go build -tags "${BUILDER_BUILDTAGS}" -o $@ ${GO_LDFLAGS}  ${GO_GCFLAGS} ./cmd/slave

${PREFIX}/bin/builder-cli: version/version.go $(shell find . -type f -name '*.go')
	@echo "+ $@"
	@go build -o $@ ${GO_LDFLAGS} ${GO_GCFLAGS} ./cmd/cli

# Depends on binaries because vet will silently fail if it can't load compiled
# imports
vet: binaries
	@echo "+ $@"
	@go vet ./...

fmt:
	@echo "+ $@"
	@test -z "$$(gofmt -s -l . | grep -v Godeps/_workspace/src/ | tee /dev/stderr)" || \
		echo "+ please format Go code with 'gofmt -s'"

lint:
	@echo "+ $@"
	@test -z "$$(golint ./... | grep -v Godeps/_workspace/src/ | tee /dev/stderr)"

build:
	@echo "+ $@"
	@go build -tags "${BUILDER_BUILDTAGS}" -v ${GO_LDFLAGS} ./...

test:
	@echo "+ $@"
	@go test -test.short -tags "${BUILDER_BUILDTAGS}" ./...

test-full:
	@echo "+ $@"
	@go test ./...

binaries: ${PREFIX}/bin/builder-master ${PREFIX}/bin/builder-slave ${PREFIX}/bin/builder-cli
	@echo "+ $@"

clean:
	@echo "+ $@"
	@rm -rf "${PREFIX}/bin/builder-master" "${PREFIX}/bin/builder-slave" "${PREFIX}/bin/builder-cli"
