coverage:
	rm -rf ./cmd/cligen/.coverdata .coverdata .coverage.raw .coverage.cligen .coverage.out
	go test ./cmd/cligen -v -coverpkg=github.com/lachaloupe/cli/cmd/cligen -coverprofile=.coverage.cligen
	mkdir .coverdata && go tool covdata merge -i ./cmd/cligen/.coverdata -o .coverdata
	go tool covdata textfmt -i .coverdata -o .coverage.raw
	grep -v "example.com/testcli" .coverage.raw > .coverage.out
	tail -n +2 .coverage.cligen >> .coverage.out
	go tool cover -html=.coverage.out
