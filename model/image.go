package model

import (
	"bytes"
	"io"
	"log"
        "mime"
        "path"

	sq "github.com/Masterminds/squirrel"
)

// type image struct {
// 	ImageID  string // ^[0-9a-f]{40}\.(?:jpeg|png)$
// 	MimeType string
// 	Bytes    []byte
// }

func storePreparedImage(m *Model, imageId string, imageData []byte) (string, error) {
	statement, err := m.tx.Prepare(
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
		m.tx.Commit()
		return imageId, nil
	}
	statement, err = m.tx.Prepare(
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
	return imageId, nil
}

func PostImage(m *Model, reader io.Reader) (string, error) {
	imageId, imageData, err := prepareImage(reader)
	if err != nil {
		log.Print(err)
		return "", err
	}
	return storePreparedImage(m, imageId, imageData)
}

func GetImageRotated(m *Model, imageId string) (string, error) {
	imageData, _, err := GetImage(m, imageId)
	if err != nil {
		log.Print(err)
		return "", err
	}
	reader := bytes.NewReader(imageData)
	newImageId, newImageData, err := rotateImage(reader)
	if err != nil {
		log.Print(err)
		return "", err
	}
	return storePreparedImage(m, newImageId, newImageData)
}

func GetImage(m *Model, imageId string) ([]byte, string, error) {
	var imageData []byte
	check(sq.Select("image_data").From("image").
		Where(sq.Eq{"image_id": imageId}).
		RunWith(db).Limit(1).QueryRow().Scan(&imageData))
        imageMimeType := mime.TypeByExtension(path.Ext(imageId))
	return imageData, imageMimeType, nil
}
