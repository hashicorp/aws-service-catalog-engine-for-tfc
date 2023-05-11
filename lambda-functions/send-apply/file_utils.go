package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func DownloadS3File(ctx context.Context, objectKey string, bucket string, s3Client *s3.Client) (*os.File, error) {
	tmp, err := os.CreateTemp("", "artifact-")
	if err != nil {
		panic(err)
	}

	downloader := manager.NewDownloader(s3Client)

	numBytes, err := downloader.Download(ctx, tmp, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, err
	}

	if numBytes < 1 {
		return nil, errors.New("zero bytes were read from S3")
	}

	if err := tmp.Close(); err != nil {
		panic(err)
	}

	return tmp, nil
}

func UploadS3File(ctx context.Context, s3Client *s3.Client, objectKey string, bucket string, file *os.File) error {
	// Upload rezipped file to s3
	uploader := manager.NewUploader(s3Client)

	_, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Body: file,
	})
	if err != nil {
		return err
	}

	return nil
}

// UnzipFile decompresses the file that is passed and returns an open file containing the newly decompressed source. It
// closes the file that was passed to it after it has been fully read.
func UnzipFile(compressed *os.File) (*os.File, error) {
	// Open the compressed file
	gzippedFile, err := os.Open(compressed.Name())
	if err != nil {
		return nil, err
	}

	// Create a gzip reader
	uncompressedSource, err := gzip.NewReader(gzippedFile)
	if err != nil {
		return nil, err
	}

	// Create new tmp file for uncompressed source
	destinationFile, err := os.CreateTemp("", "uncompressed-artifact-")
	if err != nil {
		return nil, err
	}

	// Copy the contents from the reader to the destination file
	_, err = io.Copy(destinationFile, uncompressedSource)
	if err != nil {
		return nil, err
	}

	// Close the uncompressed source
	err = uncompressedSource.Close()
	if err != nil {
		return nil, err
	}

	// Close the original source
	err = compressed.Close()
	if err != nil {
		return nil, err
	}

	return destinationFile, err
}

func ZipFile(uncompressed *os.File) (*os.File, error) {
	// Open the uncompressed file
	originalSource, err := os.Open(uncompressed.Name())
	if err != nil {
		return nil, err
	}
	defer originalSource.Close()

	// Create a new, gzipped file
	gzippedFile, err := os.Create("compressed-artifact-")
	if err != nil {
		return nil, err
	}

	// Create a new gzip writer
	gzipWriter := gzip.NewWriter(gzippedFile)

	// Copy the contents of the original source to the gzip writer
	_, err = io.Copy(gzipWriter, originalSource)
	if err != nil {
		return nil, err
	}

	// Once all the data is available, close the gzip writer
	err = gzipWriter.Close()
	if err != nil {
		return nil, err
	}

	// Close the uncompressed source file as well
	err = uncompressed.Close()
	if err != nil {
		return nil, err
	}

	return gzippedFile, err
}

func AddEntryToTar(source *os.File, entryName string, entryContents string) error {

	// TODO: Decide on byte offset -- should we use the below or use 1024 bytes or something else?
	// Seek to x number of bytes before the end of the file before writing to it
	if _, err := source.Seek(-2<<9, os.SEEK_END); err != nil {
		return nil
	}

	// Create a new tar writer
	tarWriter := tar.NewWriter(source)

	// Write to the file's header
	header := &tar.Header{
		Name: entryName,
		Size: int64(len(entryContents)),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return nil
	}

	if _, err := tarWriter.Write([]byte(entryContents)); err != nil {
		return nil
	}

	if err := tarWriter.Close(); err != nil {
		return nil
	}
	source.Close()

	return nil
}
