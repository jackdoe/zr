package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"

	"github.com/jackdoe/zr/pkg/data"
	"github.com/jackdoe/zr/pkg/util"
	"jaytaylor.com/html2text"
)

type Post struct {
	PostID           int32  `xml:"Id,attr" gorm:"primary_key;auto_increment:false"`
	PostTypeID       int32  `xml:"PostTypeId,attr" gorm:"index:idx_post_type_id"` // 1=Question 2=Answer
	ParentID         int32  `xml:"ParentId,attr" gorm:"index:idx_parent_id"`
	AcceptedAnswerID int32  `xml:"AcceptedAnswerId,attr"`
	CreationDate     string `xml:"CreationDate,attr"`
	Title            string `xml:"Title,attr"`
	Body             string `xml:"Body,attr"`
	Tags             string `xml:"Tags,attr"`
	ViewCount        int    `xml:"ViewCount,attr"`
	Score            int    `xml:"Score,attr"`
	CommentCount     int    `xml:"CommentCount,attr"`
	AnswerCount      int    `xml:"AnswerCount,attr"`
	FavoriteCount    int    `xml:"FavoriteCount,attr"`
	Indexed          int32  `gorm:"index:idx_post_type_id"`
	Accepted         bool   `gorm:"-"`
}

func (p *Post) IsQuestion() bool {
	return p.PostTypeID == 1
}

func (p *Post) IsAnswer() bool {
	return p.PostTypeID == 2
}

func (p *Post) String() string {
	var sb strings.Builder

	if p.IsQuestion() {
		if len(p.Title) > 0 {
			sb.WriteString("Q: ")
			sb.WriteString(p.Title)
			sb.WriteRune('\n')
		}

		if len(p.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("   tags:     %s\n", p.Tags))
		}
		sb.WriteString(fmt.Sprintf("   url:      stackoverflow.com/q/%d\n", p.PostID))
		sb.WriteString(fmt.Sprintf("   score:    %d/%d\n", p.Score, p.ViewCount))
		sb.WriteString(fmt.Sprintf("   created:  %s\n", p.CreationDate))
		if p.AcceptedAnswerID != 0 {
			sb.WriteString(fmt.Sprintf("   accepted: stackoverflow.com/a/%d\n", p.AcceptedAnswerID))
		}
		sb.WriteString("---\n\n")
	} else {
		url := fmt.Sprintf("A: stackoverflow.com/a/%d", p.PostID)
		sb.WriteString(fmt.Sprintf("%s score: %d, created: %s\n", url, p.Score, p.CreationDate))
		if p.Accepted {
			sb.WriteString(strings.Repeat("^", len(url)))
			sb.WriteRune('\n')
		}

		sb.WriteString("\n")
	}

	sb.WriteString(util.WrapString(p.Body, 78))
	sb.WriteRune('\n')

	s := sb.String()
	if p.IsQuestion() {
		return util.BannerLeft(30, "│", strings.Split(s, "\n"))
	}
	return util.BannerLeft(5, " ", strings.Split(s, "\n"))
}

func DecodeStream(limit int, d *xml.Decoder, cb func(p Post) error) error {
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
					panic(err)
				}

				p.Body = text

				tags := []string{}
				for _, t := range strings.Split(p.Tags, "<") {
					t = strings.Trim(t, ">")
					if len(t) > 0 {
						tags = append(tags, t)
					}
				}

				p.Tags = strings.Join(tags, ",")

				err = cb(p)
				if err != nil {
					return err
				}
			}
		default:
		}
		if limit > 0 {
			limit--
			if limit == 0 {
				return nil
			}
		}
	}
	return nil
}

func DecodeFile(limit int, fn string, cb func(p Post) error) error {
	f, err := os.OpenFile(fn, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	d := xml.NewDecoder(f)
	return DecodeStream(limit, d, cb)
}

func main() {
	root := flag.String("root", util.GetDefaultRoot(), "root")
	kind := flag.String("kind", "so", "kind of object (prependet to the id)")

	posts := flag.String("posts", "", "path to Posts.xml")
	limit := flag.Int("debug-limit", 0, "just take N documents from Posts.xml")

	batchSize := flag.Int("batch-size", 100, "insert N per chunk")

	onlyAccepted := flag.Bool("only-accepted", false, "only questions with accepted answers")

	onlyWithAnswers := flag.Bool("only-with-answers", false, "only questions with at least 1 answer")
	onlyNScore := flag.Int("at-least-score", -1000, "only questions with at least that much score")
	onlyWithNViews := flag.Int("at-least-n-views", 0, "only question threads with at least N views")
	answers := flag.Bool("index-answers", false, "index the answers (second pass)")
	questions := flag.Bool("index-questions", false, "index the questions (first pass)")

	profBind := flag.String("pprof-bind", "", "bind pprof (e.g. localhost:6060)")

	flag.Parse()

	if *profBind != "" {
		go func() {
			log.Println(http.ListenAndServe(*profBind, nil))
		}()
	}

	if *posts == "" {
		log.Fatal("need posts")
	}

	if *root == "" {
		log.Fatal("root")
	}

	if *kind == "" {
		log.Fatal("kind")
	}

	t0 := time.Now()

	store := data.NewStore(*root, *kind)
	defer store.Close()

	postCount := int(1)
	if *questions {
		type Stats struct {
			NoAccept int
			NoAnswer int
			NoView   int
			NoScore  int
			Skip     int
		}

		stats := Stats{}

		batch := []*data.Document{}
		err := DecodeFile(*limit, *posts, func(p Post) error {
			postCount++

			if p.IsQuestion() {
				noAccepted := p.AcceptedAnswerID == 0
				noAnswers := p.AnswerCount == 0

				if noAccepted {
					stats.NoAccept++
				}

				if noAnswers {
					stats.NoAnswer++
				}

				if p.ViewCount < *onlyWithNViews {
					stats.NoView++
				}

				if p.Score < *onlyNScore {
					stats.NoScore++
				}

				if (noAccepted && *onlyAccepted) || (noAnswers && *onlyWithAnswers) {
					stats.Skip++
					return nil
				}

				if p.ViewCount < *onlyWithNViews || p.Score < *onlyNScore {
					stats.Skip++
					return nil
				}
			}

			if !p.IsQuestion() {
				return nil
			}

			batch = append(batch, toDoc(p))

			if len(batch) > *batchSize {
				store.BulkUpsert(batch)

				batch = []*data.Document{}

				took := time.Since(t0)
				perSecond := float64(postCount) / took.Seconds()
				log.Printf("[first phase] storing questions [stats %+v] ... %d, per second: %.2f", stats, postCount, perSecond)
				t0 = time.Now()
			}

			return nil
		})

		if err != nil {
			panic(err)
		}

		store.BulkUpsert(batch)
	}

	if *answers {
		postCount = 0
		namedBatch := map[int32]data.Document{}

		err := DecodeFile(*limit, *posts, func(p Post) error {
			if p.IsQuestion() || p.ParentID == 0 {
				return nil
			}

			if p.Score < *onlyNScore {
				return nil
			}

			postCount++
			parent, ok := namedBatch[p.ParentID]
			if !ok {
				if err := store.DB.Where("id = ?", p.ParentID).First(&parent).Error; err != nil {
					log.Printf("missing parent %v", p.ParentID)
					return nil
				}

			}

			parent.Body = parent.Body + "\n" + p.String()
			namedBatch[p.ParentID] = parent

			if len(namedBatch) > *batchSize {
				store.BulkUpsert(toSlice(namedBatch))

				namedBatch = map[int32]data.Document{}
				took := time.Since(t0)
				perSecond := float64(postCount) / took.Seconds()
				log.Printf("[second phase] storing answers ... %d, per second: %.2f", postCount, perSecond)
				t0 = time.Now()
			}
			return nil
		})

		if err != nil {
			panic(err)
		}
		store.BulkUpsert(toSlice(namedBatch))
	}
}

func toDoc(v Post) *data.Document {
	popularity := v.ViewCount

	doc := &data.Document{
		Title:      v.Title,
		Body:       v.String(),
		Tags:       v.Tags,
		Popularity: popularity,
		ID:         fmt.Sprintf("%d", v.PostID),
	}

	return doc
}

func toSlice(in map[int32]data.Document) []*data.Document {
	out := make([]*data.Document, 0, len(in))
	for _, v := range in {
		out = append(out, &v)
	}
	return out
}
