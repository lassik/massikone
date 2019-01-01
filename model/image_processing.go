package model

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"

	"github.com/disintegration/imaging"
)

type transformFunc func(img image.Image) image.Image

func transformImage(reader io.Reader, transform transformFunc) (string, []byte, error) {
	img, oldFormat, err := image.Decode(reader)
	if err != nil {
		log.Print(err)
		return "", []byte{}, err
	}
	newFormat := "png"
	if oldFormat == "jpeg" {
		newFormat = oldFormat
	}
	img = transform(img)
	var newImageBuf bytes.Buffer
	writer := bufio.NewWriter(&newImageBuf)
	switch newFormat {
	case "jpeg":
		jpeg.Encode(writer, img, nil)
	case "png":
		png.Encode(writer, img)
	}
	writer.Flush()
	newImageBytes := newImageBuf.Bytes()
	newImageHash := sha1.New()
	newImageHash.Write(newImageBytes)
	newImageID := fmt.Sprintf("%x.%s", newImageHash.Sum(nil), newFormat)
	return newImageID, newImageBytes, err
}

func prepareImage(reader io.Reader) (string, []byte, error) {
	return transformImage(reader, func(img image.Image) image.Image {
		const maxWidth = 900
		imgWidth := img.Bounds().Dx()
		if imgWidth > maxWidth {
			img = imaging.Resize(img, maxWidth, 0, imaging.Lanczos)
		}
		img = imaging.Grayscale(img)
		return img
	})
}

func rotateImage(reader io.Reader) (string, []byte, error) {
	return transformImage(reader, func(img image.Image) image.Image {
		return imaging.Rotate270(img)
	})
}
