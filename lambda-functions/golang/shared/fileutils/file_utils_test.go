package fileutils

import (
	"os"
	"testing"
	"reflect"
	"io"
	"archive/tar"
)

func TestUnzipFile(t *testing.T) {
	// Load test fixtures for later assertions
	const MockArtifactContentsPath = "../../example-product/main.tf"
	contents, err := os.ReadFile(MockArtifactContentsPath)
	if err != nil {
		t.Errorf("Error opening test artifact %s", MockArtifactContentsPath)
	}
	const MockArtifactPath = "../../example-product/product.tar.gz"
	zipFile, err := os.OpenFile(MockArtifactPath, os.O_RDWR, os.ModePerm)
	if err != nil {
		t.Errorf("Error opening test artifact %s", MockArtifactPath)
	}
	expectedFileMap := make(map[string]string)
	expectedFileMap["main.tf"] = string(contents)

	// Unzip the test file
	unzipResult, err := UnzipFile(zipFile)
	if err != nil {
		t.Error(err)
	}

	// Check the contents of the result
	fileMap, err := getFileMap(unzipResult)
	if err != nil {
		t.Error("failed to map output file", err)
	}
	if !reflect.DeepEqual(fileMap, expectedFileMap) {
		t.Error("decompressed file contained different results than source")
	}
}

func TestZipFile(t *testing.T) {
	// Load test fixtures for later assertions
	const MockArtifactContentsPath = "../../example-product/main.tf"
	contents, err := os.ReadFile(MockArtifactContentsPath)
	if err != nil {
		t.Errorf("Error opening test artifact %s", MockArtifactContentsPath)
	}
	expectedContents := string(contents)
	const MockArtifactPath = "../../example-product/main.tf"
	testSourceFile, err := os.OpenFile(MockArtifactPath, os.O_RDWR, os.ModePerm)
	if err != nil {
		t.Errorf("Error opening test artifact %s", MockArtifactPath)
	}
	expectedFileMap := make(map[string]string)
	expectedFileMap["main.tf"] = string(contents)

	// Zip the file
	file, err := ZipFile(testSourceFile)
	if err != nil {
		t.Error(err)
	}

	// Unzip the test file
	unzipResult, err := UnzipFile(file)

	// Check the contents of the result
	unzippedBytes, err := os.ReadFile(unzipResult.Name())
	if err != nil {
		return
	}
	unzippedResult := string(unzippedBytes)
	if unzippedResult != expectedContents {
		t.Error("file had different contents after zipping and unzipping")
	}
}

func TestAddEntryToTar(t *testing.T) {
	// Load test fixtures for later assertions
	const MockArtifactPath = "../../example-product/product.tar.gz"
	zipFile, err := os.OpenFile(MockArtifactPath, os.O_RDWR, os.ModePerm)
	if err != nil {
		t.Errorf("error opening test artifact %s", MockArtifactPath)
	}

	// Unzip the test file
	tarFile, err := UnzipFile(zipFile)
	if err != nil {
		t.Error(err)
	}

	// Close the tar file
	err = tarFile.Close()
	if err != nil {
		t.Error(err)
	}

	// Open the tar file that we just closed
	newTarFile, err := os.OpenFile(tarFile.Name(), os.O_RDWR, os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	//f, err := os.Create("test.tar")
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//
	//tw := tar.NewWriter(f)
	//
	//var files = []struct {
	//	Name, Body string
	//}{
	//	{"readme.txt", "This archive contains some text files."},
	//	{"gopher.txt", "Gopher names:\nGeorge\nGeoffrey\nGonzo"},
	//	{"todo.txt", "Get animal handling licence."},
	//}
	//for _, file := range files {
	//	hdr := &tar.Header{
	//		Name: file.Name,
	//		Size: int64(len(file.Body)),
	//	}
	//	if err := tw.WriteHeader(hdr); err != nil {
	//		log.Fatalln(err)
	//	}
	//	if _, err := tw.Write([]byte(file.Body)); err != nil {
	//		log.Fatalln(err)
	//	}
	//}
	//if err := tw.Close(); err != nil {
	//	log.Fatalln(err)
	//}
	//f.Close()
	//
	//// Open the tar file that we just closed
	//newTarFile, err := os.OpenFile("test.tar", os.O_RDWR, os.ModePerm)
	//if err != nil {
	//	t.Error(err)
	//}

	// Append entry to tar file
	err = AddEntryToTar(newTarFile, "elephants", "canoe")
	if err != nil {
		t.Error(err)
	}

	err = newTarFile.Close()
	if err != nil {
		t.Error(err)
	}

	// Reopen file
	tarFileWithNewEntry, err := os.OpenFile(newTarFile.Name(), os.O_RDWR, os.ModePerm)
	if err != nil {
		t.Error(err)
	}

	// Check that the entry was added to the tar file
	fileMap, err := getFileMap(tarFileWithNewEntry)
	if err != nil {
		t.Fatal("failed to map output file", err)
	}

	newlyAddedEntry := fileMap["elephants"]
	if newlyAddedEntry == "" {
		t.Fatal("tar file was missing new entry")
	}

	if newlyAddedEntry != "canoe" {
		t.Errorf("entry was added with different contents than expected. contents were: %s", newlyAddedEntry)
	}
}

func getFileMap(reader io.Reader) (map[string]string, error) {
	fileMap := make(map[string]string)
	tarReader := tar.NewReader(reader)

	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fileMap, err
		}

		data, err := io.ReadAll(tarReader)
		if err != nil {
			return fileMap, err
		}

		fileMap[hdr.Name] = string(data)
	}

	return fileMap, nil
}
