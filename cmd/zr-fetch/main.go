package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/cheggaaa/pb/v3"

	"github.com/jackdoe/zr/pkg/util"
)

func UntarGZ(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return Untar(dst, gzr)
}

// copy pasta from https://medium.com/@skdomino/taring-untaring-files-in-go-6b07cf56bc07
func Untar(dst string, r io.Reader) error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err

		case header == nil:
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// ignore
		case tar.TypeReg:
			parts := strings.Split(header.Name, string(os.PathSeparator))
			if len(parts) < 2 {
				return errors.New("invalid path " + header.Name)
			}
			shard := parts[len(parts)-2]
			fn := parts[len(parts)-1]

			// FIXME: check its actually shard_\d+
			if !strings.HasPrefix(shard, "shard_") || !strings.HasSuffix(fn, ".db") {
				if _, err := io.Copy(ioutil.Discard, tr); err != nil {
					return err
				}
				continue
			}

			targetDir := path.Join(dst, shard)
			if _, err := os.Stat(targetDir); err != nil {
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return err
				}
			}

			target := path.Join(targetDir, fn)

			temp := target + ".tmp"
			f, err := os.OpenFile(temp, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			if err := os.Rename(temp, target); err != nil {
				return err
			}

			f.Close()
		}
	}
}

func DownloadArchive(dst string, url string) {
	log.Printf("download and unarchive %s from %s", dst, url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bar := pb.Full.Start64(resp.ContentLength)
	barReader := bar.NewProxyReader(resp.Body)

	if err := UntarGZ(dst, barReader); err != nil {
		log.Fatal(err)
	}

	bar.Finish()
}

func load(list string) io.Reader {
	if list == "-" {
		return os.Stdin
	}

	resp, err := http.Get(list)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("%s: status %s", list, resp.Status)
	}
	return resp.Body
}

func main() {
	root := flag.String("root", util.GetDefaultRoot(), "root")
	list := flag.String("list", "https://raw.githubusercontent.com/jackdoe/zr/master/public.txt", "url or - for stdin, each line has to be 'name url'")
	flag.Parse()
	if *root == "" {
		log.Fatal("need root")
	}

	r := load(*list)

	s := bufio.NewScanner(r)
	for s.Scan() {
		text := s.Text()
		splitted := strings.Split(text, " ")
		if len(splitted) < 2 {
			log.Printf("skipping line '%s', need 'name url'", text)
			continue
		}
		name := splitted[0]
		url := splitted[1]
		if len(name) == 0 || len(url) == 0 {
			log.Printf("skipping line '%s', need 'name url'", text)
			continue
		}

		DownloadArchive(path.Join(*root, name), url)
	}
}
