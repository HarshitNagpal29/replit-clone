package aws

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Initialize the S3 client
func CreateS3Client() *s3.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
		config.WithCredentialsProvider(aws.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), "")),
		config.WithEndpointResolver(aws.EndpointResolverFromURL(os.Getenv("S3_ENDPOINT"))),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	return s3.NewFromConfig(cfg)
}

var s3Client = CreateS3Client()

// Fetch S3 Folder and download files locally
func FetchS3Folder(key string, localPath string) error {
	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Prefix: aws.String(key),
	}

	response, err := s3Client.ListObjectsV2(context.TODO(), params)
	if err != nil {
		return fmt.Errorf("unable to list items in bucket %q, %v", os.Getenv("S3_BUCKET"), err)
	}

	var wg sync.WaitGroup

	for _, item := range response.Contents {
		if item.Key != nil {
			wg.Add(1)
			go func(objectKey string) {
				defer wg.Done()
				err := DownloadS3Object(objectKey, key, localPath)
				if err != nil {
					log.Println("Failed to download file:", err)
				}
			}(*item.Key)
		}
	}

	wg.Wait()
	return nil
}

func DownloadS3Object(fileKey string, key string, localPath string) error {
	getObjectParams := &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(fileKey),
	}

	result, err := s3Client.GetObject(context.TODO(), getObjectParams)
	if err != nil {
		return fmt.Errorf("unable to download item %q, %v", fileKey, err)
	}
	defer result.Body.Close()

	filePath := filepath.Join(localPath, fileKey[len(key):])
	err = WriteFile(filePath, result.Body)
	if err != nil {
		return fmt.Errorf("unable to write file %q, %v", fileKey, err)
	}

	fmt.Printf("Downloaded %s to %s\n", fileKey, filePath)
	return nil
}

// Copy S3 Folder
func CopyS3Folder(sourcePrefix string, destinationPrefix string, continuationToken *string) error {
	listParams := &s3.ListObjectsV2Input{
		Bucket:            aws.String(os.Getenv("S3_BUCKET")),
		Prefix:            aws.String(sourcePrefix),
		ContinuationToken: continuationToken,
	}

	listedObjects, err := s3Client.ListObjectsV2(context.TODO(), listParams)
	if err != nil {
		return fmt.Errorf("unable to list items in bucket %q, %v", os.Getenv("S3_BUCKET"), err)
	}

	if len(listedObjects.Contents) == 0 {
		return nil
	}

	for _, object := range listedObjects.Contents {
		if object.Key == nil {
			continue
		}
		destinationKey := destinationPrefix + (*object.Key)[len(sourcePrefix):]
		copyParams := &s3.CopyObjectInput{
			Bucket:     aws.String(os.Getenv("S3_BUCKET")),
			CopySource: aws.String(fmt.Sprintf("%s/%s", os.Getenv("S3_BUCKET"), *object.Key)),
			Key:        aws.String(destinationKey),
		}

		_, err := s3Client.CopyObject(context.TODO(), copyParams)
		if err != nil {
			return fmt.Errorf("unable to copy item %q, %v", *object.Key, err)
		}

		fmt.Printf("Copied %s to %s\n", *object.Key, destinationKey)
	}

	if *listedObjects.IsTruncated == true {
		return CopyS3Folder(sourcePrefix, destinationPrefix, listedObjects.NextContinuationToken)
	}

	return nil
}

// Save content to S3
func SaveToS3(key string, filePath string, content string) error {
	params := &s3.PutObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(key + filePath),
		Body:   bytes.NewReader([]byte(content)),
	}

	_, err := s3Client.PutObject(context.TODO(), params)
	if err != nil {
		return fmt.Errorf("unable to upload item %q, %v", filePath, err)
	}

	return nil
}

func WriteFile(filePath string, body io.ReadCloser) error {
	err := CreateFolder(filepath.Dir(filePath))
	if err != nil {
		return err
	}

	fileData, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, fileData, 0644)
}

func CreateFolder(dirName string) error {
	err := os.MkdirAll(dirName, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create directory %q, %v", dirName, err)
	}
	return nil
}
