build:
	@echo "Go version: $$(go version)"
	go build -o chunker.exe main.go
run:
	./chunker.exe serve --http="127.0.0.1:3000"
dev:
	go run . serve --http="127.0.0.1:3000"