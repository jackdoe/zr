package data

import (
	"encoding/xml"
	"io"
	"log"
	"os"

	"jaytaylor.com/html2text"
)

func DecodeStream(d *xml.Decoder, cb func(p Post) error) error {
	for {
		tok, err := d.Token()
		if tok == nil || err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("Error decoding token: %s", err)
		}

		switch ty := tok.(type) {
		case xml.StartElement:
			if ty.Name.Local == "row" {
				var p Post

				if err = d.DecodeElement(&p, &ty); err != nil {
					return err
				}

				text, err := html2text.FromString(p.Body, html2text.Options{PrettyTables: true})
				if err != nil {
					return err
				}

				p.Body = text

				err = cb(p)
				if err != nil {
					return err
				}
			}
		default:
		}
	}
	return nil
}

func DecodeFile(fn string, cb func(p Post) error) error {
	f, err := os.OpenFile(fn, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	d := xml.NewDecoder(f)
	return DecodeStream(d, cb)
}
