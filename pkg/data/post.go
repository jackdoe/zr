package data

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/rekki/go-pen"
)

func Read(m *pen.Monotonic, id int32) Post {
	b, err := m.Read(uint64(id))
	if err != nil {
		log.Fatal(err)
	}

	var p Post
	err = json.Unmarshal(b, &p)
	if err != nil {
		log.Fatal(err)
	}
	return p
}

type Post struct {
	PostID           int32   `xml:"Id,attr" gorm:"primary_key;auto_increment:false"`
	PostTypeID       int32   `xml:"PostTypeId,attr" gorm:"index:idx_post_type_id"` // 1=Question 2=Answer
	ParentID         int32   `xml:"ParentId,attr" gorm:"index:idx_parent_id"`
	AcceptedAnswerID int32   `xml:"AcceptedAnswerId,attr"`
	CreationDate     string  `xml:"CreationDate,attr"`
	Title            string  `xml:"Title,attr"`
	Body             string  `xml:"Body,attr"`
	Tags             string  `xml:"Tags,attr"`
	ViewCount        int     `xml:"ViewCount,attr"`
	Score            int     `xml:"Score,attr"`
	CommentCount     int     `xml:"CommentCount,attr"`
	AnswerCount      int     `xml:"AnswerCount,attr"`
	FavoriteCount    int     `xml:"FavoriteCount,attr"`
	ScoreF           float32 `gorm:"-"`
	Accepted         bool    `gorm:"-"`
	Indexed          int32   `gorm:"index:idx_indexed"`
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

	sb.WriteString(WrapString(p.Body, 78))
	sb.WriteRune('\n')

	return sb.String()
}

func (p *Post) IndexableFields() map[string][]string {
	out := map[string][]string{}

	out["body"] = []string{p.Body}
	out["title"] = []string{p.Title}
	for _, t := range strings.Split(p.Tags, "<") {
		t = strings.Trim(t, ">")
		if len(t) > 0 {
			out["tags"] = append(out["tags"], t)
		}
	}

	return out
}

func (p *Post) DocumentID() int32 {
	return p.PostID
}

func (p *Post) Bytes() []byte {
	encoded, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	return encoded
}
