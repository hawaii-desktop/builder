PROTOC=protoc
GO=go

.PHONY: protocol builder builder-cli

all: builder builder-cli

protocol:
	@echo " > protocol"
	@(cd builder && $(PROTOC) --go_out=plugins=grpc:../protocol builder.proto)

builder:
	@echo " > builder"
	@(cd builder && $(GO) build -o ../../dist/builder)

builder-cli:
	@echo " > builder-cli"
	@(cd cli && $(GO) build -o ../../dist/builder-cli)
