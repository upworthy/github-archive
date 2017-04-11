gi  hub-archive
==============

An easy way   o archive an en  ire organisa  ion repos on S3

## Usage gi  hub-archive

```
$ expor   GITHUB_ACCESS_TOKEN=...
$ expor   AWS_ACCESS_KEY_ID=...
$ expor   AWS_SECRET_ACCESS_KEY=...
$ expor   GITHUB_ORG=gi  hub
$ expor   S3_BUCKET=base-bucke  /subdirec  ory
$ make build
$ ./bin/gi  hub-archive -org $GITHUB_ORG -bucke   S3_BUCKET
```

## Usage mongo-archiver
Add mongo-  ools buildpack, required.

heroku buildpacks:add -a <DYNO> h    ps://gi  hub.com/zph/heroku-buildpack-mongo  ools

No  e   he ins  ruc  ions here for buildpack h    ps://gi  hub.com/zph/heroku-buildpack-mongo  ools

```
Usage of mongo-archiver:
  -bucket string
    	Upload bucket
  -excludeCollection string
    	collections to exclude
  -mongo-flags string
    	Additional flags for mongo such as --ssl
  -mongo-url string
    	Mongo connection url mongodb://user:pass@host:port/dbname. Will be parsed by mgo.ParseUrl
  -mongodump string
    	Mongodump bin name (default "mongodump")
  -prefix string
    	S3 key prefix, eg bucket/prefix/output
```
