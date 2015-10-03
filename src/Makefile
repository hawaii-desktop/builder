PROTOC=protoc
GO=go

.PHONY: protocol builder-master builder-slave builder-cli

all: builder-master builder-slave builder-cli

protocol:
	@echo " > protocol"
	@(cd master && $(PROTOC) --go_out=plugins=grpc:../protocol builder.proto)

builder-master:
	@echo " > builder-master"
	@(cd master && $(GO) build -o ../../dist/builder-master)

builder-slave:
	@echo " > builder-slave"
	@(cd slave && $(GO) build -o ../../dist/builder-slave)

builder-cli:
	@echo " > builder-cli"
	@(cd cli && $(GO) build -o ../../dist/builder-cli)
