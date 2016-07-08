github-archive
==============

An easy way to archive an entire organisation repos on S3

## Usage github-archive

```
$ export GITHUB_ACCESS_TOKEN=...
$ export AWS_ACCESS_KEY_ID=...
$ export AWS_SECRET_ACCESS_KEY=...
$ export GITHUB_ORG=github
$ export S3_BUCKET=base-bucket/subdirectory
$ make build
$ ./bin/github-archive -org $GITHUB_ORG -bucket S3_BUCKET
```

## Usage mongo-archiver
Add mongo-tools buildpack, required.

heroku buildpacks:add -a <DYNO> https://github.com/zph/heroku-buildpack-mongotools

```
./bin/mongo-archiver ...args
```
