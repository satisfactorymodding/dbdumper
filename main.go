package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var emailRegex = regexp.MustCompile(`'[a-z0-9._%+!$&*=^|~#'?{}/\-]+?@[a-z0-9\- ]+?\.[^@' ]+?'`)

func main() {
	// read in flags
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("Usage: go run main.go <host> <port> <user> <password> <endpoint> <bucket> <key> <secret> <path>")
	}

	host := flag.Arg(0)
	port := flag.Arg(1)
	user := flag.Arg(2)
	password := flag.Arg(3)

	endpoint := flag.Arg(4)
	bucket := flag.Arg(5)
	key := flag.Arg(6)
	secret := flag.Arg(7)
	path := flag.Arg(8)

	// set password
	_ = os.Setenv("PGPASSWORD", password)

	// execute pg_dump
	cmd := exec.Command(
		"pg_dump",
		"-h", host,
		"-p", port,
		"-U", user,
		"-b",
		"--quote-all-identifiers",
		"--inserts",
		"--exclude-table-data", "user_sessions",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("error dumping data: %s\noutput: %s", err, string(output))
	}

	outStr := string(output)

	// sanitize data
	uuid.EnableRandPool()

	// replace all emails
	outStr = emailRegex.ReplaceAllStringFunc(outStr, func(s string) string {
		return "'" + uuid.New().String() + "@" + uuid.New().String() + ".com'"
	})

	// save data
	//if err := os.WriteFile("out.sql", []byte(outStr), 0644); err != nil {
	//	log.Fatalf("error saving output: %s", err)
	//}

	// upload to s3
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(key, secret, "")),
		config.WithRegion("us-west-2"),
	)
	if err != nil {
		log.Fatalf("failed loading config: %s", err)
	}

	client := s3.NewFromConfig(
		cfg,
		func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		},
	)

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
		Body:   bytes.NewReader([]byte(outStr)),
	})
	if err != nil {
		log.Fatalf("failed uploading object: %s", err)
	}

	log.Println("successfully uploaded object")
}
