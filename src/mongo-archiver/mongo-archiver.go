package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rlmcpherson/s3gof3r"
	"gopkg.in/mgo.v2"
)

var (
	bucketName        = flag.String("bucket", "", "Upload bucket")
	keyPrefix         = flag.String("prefix", "", "S3 key prefix, eg bucket/prefix/output")
	mongodump         = flag.String("mongodump", "mongodump", "Mongodump bin name")
	mongoUrl          = flag.String("mongo-url", "", "Mongo connection url mongodb://user:pass@host:port/dbname. Will be parsed by mgo.ParseUrl")
	excludeCollection = flag.String("excludeCollection", "", "collections to exclude")
	mongoFlags        = flag.String("mongo-flags", "", "Additional flags for mongo such as --ssl")
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

func createBackup(dialInfo *mgo.DialInfo) error {
	defer pWriter.Close()
	defer wg.Done()
	wg.Add(1)
	name, err := exec.LookPath(*mongodump)
	if err != nil {
		log.Fatalf("Mongodump cannot be found on path")
	}
	db := &dialInfo.Database
	username := &dialInfo.Username
	password := &dialInfo.Password
	host := strings.Join(dialInfo.Addrs, ",")
	args := []string{
		"--archive",
		"--gzip",
		"--db=" + *db,
		"--username=" + *username,
		"--password=" + *password,
		"--host=" + host}
	flags := strings.Split(*mongoFlags, " ")
	args = append(flags, args...)
	// TODO: test for newness of mongo Archive requires newish >= 3.1 version of mongodump
	// 3.0.5 in homebrew is missing --archive
	// 3.2 is where archive to STDOUT became available
	if *excludeCollection != "" {
		*excludeCollection = "--excludeCollection=" + *excludeCollection
		args = append(args, *excludeCollection)
	}
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

func pseudo_uuid() (uuid string) {
	// Credit: http://stackoverflow.com/a/25736155
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	uuid = fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return
}

func setupFlags() {
	flag.Parse()
	flags := []string{"bucket", "mongodump", "mongo-url"}
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
}

func setupS3() *s3gof3r.Bucket {
	awsAccessKey := mustGetEnv("AWS_ACCESS_KEY_ID")
	awsSecretKey := mustGetEnv("AWS_SECRET_ACCESS_KEY")
	keys := s3gof3r.Keys{
		AccessKey: awsAccessKey,
		SecretKey: awsSecretKey,
	}
	s3 := s3gof3r.New("", keys)
	return s3.Bucket(*bucketName)
}

func generateS3Key(db *string) string {
	now := time.Now().Format("2006-01-02/15")
	prefix := ""
	if *keyPrefix != "" {
		prefix = *keyPrefix + "/"
	}
	uuid := pseudo_uuid()
	return fmt.Sprintf("%s%s/%s/%s.tar.gz", prefix, *db, now, uuid)
}

func main() {
	setupFlags()
	bucket := setupS3()
	dialInfo, err := mgo.ParseURL(*mongoUrl)
	if err != nil {
		panic("Unable to parse mongo uri")
	}

	go createBackup(dialInfo)

	s3Key := generateS3Key(&dialInfo.Database)
	output := fmt.Sprintf("s3://%s/%s", *bucketName, s3Key)
	w, err := bucket.PutWriter(s3Key, nil, nil)
	if err != nil {
		log.Fatalf("Error with bucket (%s/%s) PutWriter: %s", *bucketName, s3Key, err)
	}
	defer func() {
		w.Close()
		log.Printf("Successfully uploaded %s", output)
	}()

	log.Printf("Uploading to %s", output)
	written, err := io.Copy(w, pReader)
	if err != nil {
		log.Printf("Error Uploading to %s, ERROR: %s", output, err)
	}

	wg.Wait()

	log.Printf("Attempting to write %d bytes", written)
}
