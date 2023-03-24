# build file
GO_BUILD=go build -ldflags -s -v

unipay: BIN_BINARY_NAME=unipay
unipay:
	GO111MODULE=on $(GO_BUILD) -o $(BIN_BINARY_NAME) cmd/main.go
	@echo "Build $(BIN_BINARY_NAME) successfully. You can run ./$(BIN_BINARY_NAME) now.If you can't see it soon,wait some seconds"

update:
	go mod tidy

docker:
	docker build --network host -t dotbitteam/unipay:latest .

docker-publish:
	docker image push dotbitteam/unipay:latest
