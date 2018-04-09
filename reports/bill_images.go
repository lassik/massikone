package reports

import (
	"fmt"
	"log"
	"path"

	"../model"
)

func addBillImagesToZip(getWriter GetWriter) {
	images, missing := model.GetBillsForImages()
	for _, image := range images {
		if image["image_id"] != nil {
			w, err := getWriter(
				"image/"+path.Ext(image["image_id"].(string)),
				fmt.Sprintf("tositteet/tosite-%03d-%d-%s%s",
					image["bill_id"].(string),
					image["bill_image_num"].(string),
					Slug(image["description"].(string)),
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
		for _, billId := range missing {
			fmt.Fprintf(w, "#%d\r\n", billId)
		}
	}
}
