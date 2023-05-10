package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"os"
)

func createTarGzipArchive(files []string, outputPath string) {
	out, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, filename := range files {
		func() {
			file, err := os.Open(filename)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()

			info, err := file.Stat()
			if err != nil {
				log.Fatal(err)
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				log.Fatal(err)
			}

			header.Name = filename
			err = tw.WriteHeader(header)
			if err != nil {
				log.Fatal(err)
			}

			_, err = io.Copy(tw, file)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}
}

func main() {
	createTarGzipArchive([]string{"main.tf"}, "product.tar.gz")
}
