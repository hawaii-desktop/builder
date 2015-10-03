.PHONY: protocol builder-master builder-slave builder-cli

all: builder-master builder-slave builder-cli

deps:
	@go get -t ./...

protocol:
	@make -C src protocol

builder-master:
	@make -C src builder-master

builder-slave:
	@make -C src builder-slave

builder-cli:
	@make -C src builder-cli
