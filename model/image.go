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
	"mime"
	"path"

	sq "github.com/Masterminds/squirrel"
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

func GetImageRotated(imageId string) (string, error) {
	imageData, _, err := GetImage(imageId)
	if err != nil {
		log.Print(err)
		return "", err
	}
	reader := bytes.NewReader(imageData)
	newImageId, newImageData, err := ImageRotate(reader)
	if err != nil {
		log.Print(err)
		return "", err
	}
	return modelStoreImage(newImageId, newImageData)
}

func GetImage(imageId string) ([]byte, string, error) {
	var imageData []byte
	check(sq.Select("image_data").From("image").
		Where(sq.Eq{"image_id": imageId}).
		RunWith(db).Limit(1).QueryRow().Scan(&imageData))
	imageMimeType := mime.TypeByExtension(path.Ext(imageId))
	return imageData, imageMimeType, nil
}

func modelStoreImage(imageId string, imageData []byte) (string, error) {
	transaction, err := db.Begin()
	if err != nil {
		log.Print(err)
		return "", err
	}
	statement, err := transaction.Prepare(
		"update image set image_id = ?, image_data = ? where image_id = ?")
	if err != nil {
		log.Print(err)
		return "", err
	}
	result, err := statement.Exec(imageId, imageData, imageId)
	if err != nil {
		log.Print(err)
		return "", err
	}
	count, err := result.RowsAffected()
	if err != nil {
		log.Print(err)
		return "", err
	}
	statement.Close()
	if count > 0 {
		transaction.Commit()
		return imageId, nil
	}
	statement, err = transaction.Prepare(
		"insert into image (image_id, image_data) values (?, ?)")
	if err != nil {
		log.Print(err)
		return "", err
	}
	_, err = statement.Exec(imageId, imageData)
	if err != nil {
		log.Print(err)
		return "", err
	}
	statement.Close()
	transaction.Commit()
	return imageId, nil
}

func PostImage(reader io.Reader) (string, error) {
	imageId, imageData, err := ImagePrepare(reader)
	if err != nil {
		log.Print(err)
		return "", err
	}
	return modelStoreImage(imageId, imageData)
}
