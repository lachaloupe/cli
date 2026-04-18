coverage:
	rm -rf ./cmd/cligen/.coverdata .coverdata .coverage.raw .coverage.out
	go test ./cmd/cligen -v
	mkdir .coverdata && go tool covdata merge -i ./cmd/cligen/.coverdata -o .coverdata
	go tool covdata textfmt -i .coverdata -o .coverage.raw
	grep -v "example.com/testcli" .coverage.raw > .coverage.out
	go tool cover -html=.coverage.out
