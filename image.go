package main

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

func imageTransform(reader io.Reader,
	transform func(img image.Image) image.Image) (string, []byte, error) {
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
	newImageData := newImageBuf.Bytes()
	newImageHash := sha1.New()
	newImageHash.Write(newImageData)
	newImageId := fmt.Sprintf("%x.%s", newImageHash.Sum(nil), newFormat)
	return newImageId, newImageData, err
}

func ImagePrepare(reader io.Reader) (string, []byte, error) {
	return imageTransform(reader, func(img image.Image) image.Image {
		img = imaging.Resize(img, 900, 0, imaging.Lanczos)
		img = imaging.Grayscale(img)
		return img
	})
}

func ImageRotate(reader io.Reader) (string, []byte, error) {
	return imageTransform(reader, func(img image.Image) image.Image {
		return imaging.Rotate270(img)
	})
}
