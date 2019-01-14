package reports

import (
	"fmt"
	"log"
	"path"

	"github.com/lassik/massikone/model"
)

func addDocumentImagesToZip(m *model.Model, getWriter GetWriter) {
	images, missing := m.GetDocumentsForImages()
	for _, image := range images {
		if image["image_id"] != nil {
			w, err := getWriter(
				"image/"+path.Ext(image["image_id"].(string)),
				fmt.Sprintf("tositteet/tosite-%03d-%d-%s%s",
					image["document_id"].(int),
					image["document_image_num"].(int),
					slug(image["description"].(string)),
					path.Ext(image["image_id"].(string))))
			if err != nil {
				log.Fatal(err)
			}
			_, err = w.Write(image["image_data"].([]byte))
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if len(missing) > 0 {
		w, err := getWriter("text/plain", "tositteet/puuttuvat.txt")
		check(err)
		for _, documentId := range missing {
			fmt.Fprintf(w, "#%d\r\n", documentId)
		}
	}
}
