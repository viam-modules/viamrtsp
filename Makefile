
viamrtsp: *.go cmd/module/*.go
	go build -o viamrtsp cmd/module/cmd.go

test:
	go test

lint:
	gofmt -w -s .
