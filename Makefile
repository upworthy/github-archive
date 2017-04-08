clean:
	rm -rf ~/tmp/gh-archive-*
run:
	bash -c 'go run src/github-archive/github-archive.go -org $$GITHUB_ORG -bucket $$S3_BUCKET'
build: src/mongo-archiver/mongo-archiver.go src/github-archive/github-archive.go
	@ go build -o bin/mongo-archiver ./src/mongo-archiver/mongo-archiver.go
	@ go build -o bin/github-archiver ./src/github-archive/github-archive.go
execute:
	./bin/github-archive -org $$GITHUB_ORG -bucket $$S3_BUCKET
mongo_backup:
	@ ./bin/mongo-archiver -bucket $$S3_BUCKET -prefix $$S3_PREFIX_MONGO -mongo-url $$MONGO_URL -mongo-flags "--ssl --sslAllowInvalidCertificates"
