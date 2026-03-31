coverage:
	go test -v -coverprofile=.coverage ./... ./cmd/cligen
	go tool cover -html=.coverage
