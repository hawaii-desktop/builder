.PHONY: protocol builder builder-cli

all: builder builder-cli

deps:
	@go get -t ./...

protocol:
	@make -C src protocol

builder:
	@make -C src builder

builder-cli:
	@make -C src builder-cli
