clean:
	rm -rf ~/tmp/gh-archive-*
run:
	bash -c 'go run src/github-archive/github-archive.go -org $$GITHUB_ORG -bucket $$S3_BUCKET'
build:
	bash -c 'GOPATH=$$(pwd) gb build all'
execute:
	./bin/github-archive -org $$GITHUB_ORG -bucket $$S3_BUCKET
mongo_backup:
	./bin/mongo-archiver -bucket $$S3_BUCKET -prefix $$S3_PREFIX_MONGO -username $$MONGO_USER -password $$MONGO_PASSWORD -host $$MONGO_URL -db $$MONGO_DB -excludeCollection $$MONGO_EXCLUDES
