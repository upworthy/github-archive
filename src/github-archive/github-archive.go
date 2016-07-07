package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"crypto/md5"
	"github.com/google/go-github/github"
	"github.com/rlmcpherson/s3gof3r"
	"golang.org/x/oauth2"
)

var (
	org        = flag.String("org", "", "Organisation")
	bucketName = flag.String("bucket", "", "Upload bucket")

	gh          *github.Client
	bucket      *s3gof3r.Bucket
	date        string
	githubToken string
)

type Repo struct {
	Date     string
	Owner    string
	Name     string
	FullName string
	URL      string
}

func mustGetEnv(key string) string {
	s := os.Getenv(key)
	if s == "" {
		log.Fatalf("Missing ENV %s", key)
	}
	return s
}

func main() {
	flag.Parse()

	var err error
	var goWorkers int
	githubToken = mustGetEnv("GITHUB_ACCESS_TOKEN")
	awsAccessKey := mustGetEnv("AWS_ACCESS_KEY_ID")
	awsSecretKey := mustGetEnv("AWS_SECRET_ACCESS_KEY")
	goWorkers, err = strconv.Atoi(os.Getenv("GO_WORKERS"))
	if err != nil {
		log.Fatal("Cannot parse GO_WORKERS ENV variable into int.")
	}

	if goWorkers == 0 {
		goWorkers = 50
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	gh = github.NewClient(tc)

	keys := s3gof3r.Keys{
		AccessKey: awsAccessKey,
		SecretKey: awsSecretKey,
	}
	s3 := s3gof3r.New("", keys)
	bucket = s3.Bucket(*bucketName)

	repoChan := make(chan Repo)

	wg := new(sync.WaitGroup)
	for i := 0; i < goWorkers; i++ {
		wg.Add(1)
		go worker(repoChan, wg)
	}

	err = uploadReposForOrg(repoChan, *org)
	if err != nil {
		log.Fatal(err)
	}

	close(repoChan)
	wg.Wait()
}

func uploadReposForOrg(repoChan chan Repo, org string) error {
	now := time.Now().Format("20060102150405")

	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 30},
	}
	for {
		repos, resp, err := gh.Repositories.ListByOrg(org, opt)
		if err != nil {
			return err
		}

		for _, repo := range repos {
			r := Repo{
				Date:     now,
				Owner:    *repo.Owner.Login,
				Name:     *repo.Name,
				FullName: *repo.FullName,
				URL:      *repo.SSHURL,
			}
			repoChan <- r
		}

		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}

	return nil
}

func worker(repoChan chan Repo, wg *sync.WaitGroup) {
	defer wg.Done()

	for repo := range repoChan {
		n, err := uploadRepositoryToS3(bucket, repo)
		if err != nil {
			log.Printf("Error while downloading %s: %s", repo.URL, err)
		}
		if n != 0 {
			log.Printf("Successfully uploaded %s (%d bytes)", repo.URL, n)
		}
	}
}

func uploadRepositoryToS3(bucket *s3gof3r.Bucket, repo Repo) (int64, error) {
	tmp, err := ioutil.TempDir("", "gh-archive-")
	if err != nil {
		return 0, err
	}
	defer cleanup(tmp)

	cloneDirectory := fmt.Sprintf("%s-%s-%s", repo.Date, repo.Owner, repo.Name)
	err = cloneRepo(tmp, cloneDirectory, repo)
	if err != nil {
		return 0, err
	}

	archive := cloneDirectory + ".tar.gz"
	err = archiveRepo(tmp, archive, cloneDirectory)
	if err != nil {
		return 0, err
	}

	archivePath := filepath.Join(tmp, archive)
	sum := fileMD5(archivePath)

	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return 0, err
	}
	defer archiveFile.Close()

	s3Key := fmt.Sprintf("%s/%s/%s/%s-%s.tar.gz", repo.Owner, repo.Name, repo.Date, repo.Name, sum)
	w, err := bucket.PutWriter(s3Key, nil, nil)
	if err != nil {
		return 0, err
	}

	n, err := io.Copy(w, archiveFile)
	if err != nil {
		return 0, err
	}

	if err = w.Close(); err != nil {
		return 0, err
	}

	return n, nil
}

type md5sum string

func fileMD5(file string) md5sum {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println(file, err)
	}

	sum := md5.Sum(b)
	return md5sum(fmt.Sprintf("%x", sum))
}

func randomSleep(max int) {
	delay := time.Duration(rand.Intn(max))
	time.Sleep(delay * time.Millisecond)
}

func runCloneCmd(cmdDir, directory string, r Repo) error {
	cmd := exec.Command("git", "clone", httpsRepoWithCredential(r.FullName, githubToken), directory)
	cmd.Dir = cmdDir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return err
}

func cloneRepo(cmdDir, directory string, r Repo) error {
	randomSleep(50)
	err := runCloneCmd(cmdDir, directory, r)
	if err != nil {
		randomSleep(50)
		err := runCloneCmd(cmdDir, directory, r)
		if err != nil {
			return err
		}
	}
	return nil
}

func httpsRepoWithCredential(fullName, token string) string {
	return fmt.Sprintf("https://token:%s@github.com/%s.git", token, fullName)
}

func archiveRepo(cmdDir, archiveFile, directory string) error {
	cmd := exec.Command("tar", "cvzf", archiveFile, directory)
	cmd.Dir = cmdDir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func cleanup(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		log.Fatal(err)
	}
}
