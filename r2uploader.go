package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Uploader struct {
	client   *s3.Client
	uploader *manager.Uploader
}

type progressReader struct {
	reader     *os.File
	size       int64
	read       int64
	lastUpdate int64
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)

	// Update progress every 1% or at least every 3 seconds
	currentPercent := (r.read * 100) / r.size
	currentTime := time.Now().Unix()
	if currentPercent > (r.lastUpdate*100)/r.size || currentTime-r.lastUpdate >= 3 {
		fmt.Printf("\rProgress: %.1f%% (%d/%d MB)", float64(r.read*100)/float64(r.size), r.read/(1024*1024), r.size/(1024*1024))
		r.lastUpdate = currentTime
	}

	return n, err
}

func NewR2Uploader(creds CloudflareCredentials) (*R2Uploader, error) {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               fmt.Sprintf("https://%s.r2.cloudflarestorage.com", creds.AccountID),
			SigningRegion:     "auto",
			HostnameImmutable: true,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			creds.AccessKey,
			creds.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	client := s3.NewFromConfig(cfg)

	// Create uploader with custom configuration
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 64 * 1024 * 1024 // 64MB per part
		u.Concurrency = 3             // Number of concurrent upload goroutines
	})

	return &R2Uploader{
		client:   client,
		uploader: uploader,
	}, nil
}

func (u *R2Uploader) UploadFile(bucketName string, filePath string, baseDir string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to open file %s: %v", filePath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("unable to get file info: %v", err)
	}

	// Construct the key (path in the bucket)
	fileName := filepath.Base(filePath)
	// Replace problematic characters with underscores
	safeFileName := strings.Map(func(r rune) rune {
		if r == '[' || r == ']' || r == '(' || r == ')' || r == ' ' {
			return '_'
		}
		return r
	}, fileName)

	var key string
	if baseDir != "" {
		// If it's part of a directory upload, include the base directory
		key = filepath.Join(baseDir, safeFileName)
		// Convert to forward slashes for S3 path compatibility
		key = strings.ReplaceAll(key, "\\", "/")
	} else {
		key = safeFileName
	}

	fmt.Printf("Starting upload of %s (%.2f MB)...\n", fileName, float64(fileInfo.Size())/(1024*1024))

	// Create progress reader
	reader := &progressReader{
		reader:     file,
		size:       fileInfo.Size(),
		lastUpdate: time.Now().Unix(),
	}

	_, err = u.uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
		Body:   reader,
	})

	if err != nil {
		fmt.Println() // New line after progress
		return "", fmt.Errorf("unable to upload file %s: %v", filePath, err)
	}

	fmt.Printf("\nSuccessfully uploaded %s as %s\n", fileName, key)
	return key, nil
}

func (u *R2Uploader) GetPresignedURL(bucketName string, key string) (string, error) {
	presignClient := s3.NewPresignClient(u.client)

	request, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(time.Hour*24)) // URL valid for 24 hours

	if err != nil {
		return "", fmt.Errorf("couldn't generate presigned URL: %v", err)
	}

	return request.URL, nil
}
