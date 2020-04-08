package data

type Document struct {
	RowID int32  `gorm:"primary_key;auto_increment:true;not null"`
	ID    string `gorm:"unique_index"`

	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	Tags  string `json:"tags,omitempty"`

	Popularity int   `json:"popularity,omitempty"`
	Indexed    int32 `gorm:"index:idx_indexed"`
}

func (d *Document) DocumentID() int32 {
	return d.RowID
}

func (d *Document) IndexableFields() map[string][]string {
	out := map[string][]string{}

	out["title"] = []string{d.Title}
	out["body"] = []string{d.Body}
	out["tags"] = []string{d.Tags}

	return out
}
