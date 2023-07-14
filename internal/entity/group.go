package entity

import (
	"time"
)

type Category int

const (
	Photo Category = iota
	Video
)

const (
	LayoutTime     = "15:04"
	LayoutDateTime = "2006-01-02T15:04"
)

//type Datetime struct {
//	Time time.Time `db:"date_time"`
//}
//
//func (c *Datetime) UnmarshalJSON(b []byte) error {
//	t, err := time.Parse(LayoutDateTime, string(b)) //parse time
//	if err != nil {
//		return err
//	}
//
//	*c = Datetime{t}
//	return nil
//}
//
//type ScanTime struct {
//	Time time.Time `db:"scan_time"`
//}
//
//func (c *ScanTime) UnmarshalJSON(b []byte) error {
//	t, err := time.Parse(LayoutTime, string(b)) //parse time
//	if err != nil {
//		return err
//	}
//	log.Println(t.String() + "!!!!!!!!!!!!!!!!!!!!!!!!!!")
//	*c = ScanTime{t}
//	return nil
//}

type Group struct {
	Id           int       `db:"id,omitempty"`
	Name         string    `db:"name" form:"name"`
	Description  string    `db:"description" form:"description"`
	Tag          string    `db:"tag" form:"tag"`
	Link         string    `db:"link" form:"link"`
	Stopwords    string    `db:"stopwords" form:"stopwords"`
	NDays        int       `db:"n_days" form:"n-days"`
	ScanTime     time.Time `db:"scan_time"`
	LastScanTime time.Time `db:"last_scan_time"`
}

type Source struct {
	Id            int      `db:"id"`
	Category      Category `db:"category" form:"category"`
	Link          string   `db:"link" form:"link"`
	DurationLimit int      `db:"duration_limit" form:"duration"`
	LikeLimit     int      `db:"like_limit" form:"like"`
	CommentLimit  int      `db:"comment_limit" form:"comment"`
	RepostLimit   int      `db:"repost_limit" form:"repost"`
	ViewLimit     int      `db:"view_limit" form:"view"`
}

type Event struct {
	Id             int       `db:"id"`
	Category       Category  `db:"category" form:"category"`
	Datetime       time.Time `db:"date_time"`
	RepeatInterval int       `db:"repeat_interval" form:"repeat-interval"`
}

type Post struct {
	Id          int
	FromId      int
	Likes       int
	Views       int
	Reposts     int
	Comments    int
	Attachments []Attachment
	Text        string
	Date        time.Time
}

type Attachment struct {
	Id       int
	Category Category
	Link     string
	Duration int
}
