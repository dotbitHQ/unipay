# build file
GO_BUILD=go build -ldflags -s -v

svr_unipay: BIN_BINARY_NAME=unipay_svr
svr_unipay:
	GO111MODULE=on $(GO_BUILD) -o $(BIN_BINARY_NAME) cmd/main.go
	@echo "Build $(BIN_BINARY_NAME) successfully. You can run ./$(BIN_BINARY_NAME) now.If you can't see it soon,wait some seconds"

svr_refund: BIN_BINARY_NAME=refund_svr
svr_refund:
	GO111MODULE=on $(GO_BUILD) -o $(BIN_BINARY_NAME) cmd/refund/main.go
	@echo "Build $(BIN_BINARY_NAME) successfully. You can run ./$(BIN_BINARY_NAME) now.If you can't see it soon,wait some seconds"

update:
	go mod tidy

docker:
	docker build --network host -t admindid/unipay:latest .

docker-publish:
	docker image push admindid/unipay:latest
