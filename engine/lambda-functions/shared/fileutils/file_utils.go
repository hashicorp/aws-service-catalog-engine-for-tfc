/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package fileutils

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func DownloadS3File(ctx context.Context, s3Downloader S3Downloader, launchRoleArn string, s3Path string) (*os.File, error) {
	log.Default().Printf("parsing s3 Path: %s", s3Path)
	bucket, objectKey := resolveArtifactPath(s3Path)

	log.Default().Print("downloading product terraform configuration from s3")

	tmp, err := os.CreateTemp("", "artifact-")
	if err != nil {
		panic(err)
	}

	numBytes, err := s3Downloader.Download(ctx, launchRoleArn, tmp, bucket, objectKey)
	if err != nil {
		return nil, err
	}

	if numBytes < 1 {
		return nil, errors.New("zero bytes were read from S3")
	}

	// Rewind the file so that it can be read in the future
	_, err = tmp.Seek(0, io.SeekStart)

	log.Default().Print("downloaded product terraform configuration from s3")

	return tmp, err
}

// UnzipFile decompresses the file that is passed and returns an open file containing the newly decompressed source.
//
//	It closes the file that was passed to it after it has been fully read.
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
	tmpFileNameSuffix := strings.TrimSuffix(path.Base(compressed.Name()), filepath.Ext(compressed.Name()))
	tmpFileName := fmt.Sprintf("%d-%s", time.Now().Unix(), tmpFileNameSuffix)
	tmpDir := os.TempDir()
	tmpFilePath := path.Join(tmpDir, tmpFileName)
	destinationFile, err := os.OpenFile(tmpFilePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
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

	// Rewind the destination file so that it can be read
	_, err = destinationFile.Seek(0, io.SeekStart)
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
	gzippedFileNameSuffix := fmt.Sprintf("%s.gz", path.Base(originalSource.Name()))
	gzippedFileName := fmt.Sprintf("%d-%s", time.Now().Unix(), gzippedFileNameSuffix)
	tmpDir := os.TempDir()
	tmpFilePath := path.Join(tmpDir, gzippedFileName)
	gzippedFile, err := os.OpenFile(tmpFilePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
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

	// Rewind the destination file so that it can be read
	_, err = gzippedFile.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	return gzippedFile, err
}

func AddEntryToTar(source *os.File, entryName string, entryContents string) error {
	// Seek to the start of the last file and check its size
	var lastFileSize, lastStreamPos int64
	tr := tar.NewReader(source)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		lastStreamPos, err = source.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		lastFileSize = hdr.Size
	}

	// Find the next block boundary
	const blockSize = 512
	newOffset := lastStreamPos + lastFileSize
	distanceToNextBlockBoundary := newOffset % blockSize
	// If the newOffset is already on a block boundary, we need to avoid writing an empty block
	if distanceToNextBlockBoundary != 0 {
		newOffset += blockSize - (newOffset % blockSize)
	}

	if _, err := source.Seek(newOffset, io.SeekStart); err != nil {
		return err
	}

	// Create a new tar writer
	tarWriter := tar.NewWriter(source)

	// Write to the file's header
	header := &tar.Header{
		Name: entryName,
		Size: int64(len(entryContents)),
		Mode: 0777,
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := tarWriter.Write([]byte(entryContents)); err != nil {
		return err
	}

	if err := tarWriter.Close(); err != nil {
		return err
	}

	return nil
}

// Resolves artifactPath to bucket and key
func resolveArtifactPath(artifactPath string) (string, string) {
	bucket := strings.Split(artifactPath, "/")[2]
	key := strings.SplitN(artifactPath, "/", 4)[3]
	return bucket, key
}
