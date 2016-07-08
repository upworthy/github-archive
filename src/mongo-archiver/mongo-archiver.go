package main

import (
	"flag"
	"fmt"
	"github.com/rlmcpherson/s3gof3r"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	bucketName        = flag.String("bucket", "", "Upload bucket")
	keyPrefix         = flag.String("prefix", "", "S3 key prefix, eg bucket/prefix/output")
	mongodump         = flag.String("mongodump", "mongodump", "Mongodump bin name")
	db                = flag.String("db", "", "db name")
	username          = flag.String("username", "", "user name")
	password          = flag.String("password", "", "password")
	host              = flag.String("host", "", "host:port")
	excludeCollection = flag.String("excludeCollection", "", "collections to exclude")
	pReader, pWriter  = io.Pipe()

	wg sync.WaitGroup

	bucket *s3gof3r.Bucket
	date   string
)

func mustGetEnv(key string) string {
	s := os.Getenv(key)
	if s == "" {
		log.Fatalf("Missing ENV %s", key)
	}
	return s
}

func createBackup() error {
	defer pWriter.Close()
	defer wg.Done()
	wg.Add(1)
	name, err := exec.LookPath(*mongodump)
	if err != nil {
		log.Fatalf("Mongodump cannot be found on path")
	}
	// TODO: test for newness of mongo Archive requires newish >= 3.1 version of mongodump
	// 3.0.5 in homebrew is missing --archive
	// 3.2 is where archive to STDOUT became available
	if *excludeCollection != "" {
		*excludeCollection = "--excludeCollection=" + *excludeCollection
	}
	args := []string{"--archive", "--db=" + *db, "--username=" + *username, "--password=" + *password, "--host=" + *host, *excludeCollection, "--gzip"}
	cmd := exec.Command(name, args...)
	cmd.Stdout = pWriter
	cmd.Stderr = os.Stderr
	log.Printf("CMD: $ %s %s", name, strings.Join(cmd.Args, " "))
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	flags := []string{"bucket", "mongodump", "db", "username", "password", "host"}
	fatal := false
	for _, f := range flags {
		fl := flag.Lookup(f)
		s := fl.Value.String()
		if s == "" {
			fatal = true
			log.Printf("Flag missing -%s which requires %s", fl.Name, fl.Usage)
		}
	}
	if fatal {
		log.Fatal("Exiting because of missing flags.")
	}

	awsAccessKey := mustGetEnv("AWS_ACCESS_KEY_ID")
	awsSecretKey := mustGetEnv("AWS_SECRET_ACCESS_KEY")
	keys := s3gof3r.Keys{
		AccessKey: awsAccessKey,
		SecretKey: awsSecretKey,
	}
	s3 := s3gof3r.New("", keys)
	bucket = s3.Bucket(*bucketName)

	go createBackup()

	now := time.Now().Format("2006-01-02")
	prefix := ""
	if *keyPrefix != "" {
		prefix = *keyPrefix + "/"
	}
	s3Key := fmt.Sprintf("%s%s/%s/%s.tar.gz", prefix, *db, now, *db)
	w, err := bucket.PutWriter(s3Key, nil, nil)
	if err != nil {
		log.Fatalf("Error with bucket (%s/%s) PutWriter: %s", *bucketName, s3Key, err)
	}
	defer func() {
		w.Close()
		log.Printf("Successfully uploaded s3://%s/%s", *bucketName, s3Key)
	}()

	log.Printf("Uploading to s3://%s/%s", *bucketName, s3Key)
	written, err := io.Copy(w, pReader)
	if err != nil {
		log.Printf("Error Uploading to s3://%s/%s, ERROR: %s", *bucketName, s3Key, err)
	}

	wg.Wait()

	log.Printf("Attempting to write %d bytes", written)
}
