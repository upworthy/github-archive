clean:
	rm -rf ~/tmp/gh-archive-*
run:
	bash -c 'go run github-archive.go -org $$GITHUB_ORG -bucket $$S3_BUCKET'
