clean:
	rm -rf ~/tmp/gh-archive-*
run:
	bash -c 'go run src/github-archive/github-archive.go -org $$GITHUB_ORG -bucket $$S3_BUCKET'
build:
	bash -c 'GOPATH=$$(pwd) gb build all'
execute:
	./bin/github-archive -org $$GITHUB_ORG -bucket $$S3_BUCKET
