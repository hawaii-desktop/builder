PROTOC := protoc

protocol:
	@$(PROTOC) --go_out=plugins=grpc:common/protocol builder.proto
