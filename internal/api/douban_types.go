package api

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// FlexInt handles JSON values that may be either a number or a string-encoded number.
type FlexInt int

func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*fi = FlexInt(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		n, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		*fi = FlexInt(n)
		return nil
	}
	return fmt.Errorf("FlexInt: cannot unmarshal %s", string(data))
}

type DoubanSubjectCollectionItem struct {
	ID           FlexInt          `json:"id"`
	Type         string           `json:"type"`
	Title        string           `json:"title"`
	CardSubtitle string           `json:"card_subtitle"`
	Description  string           `json:"description"`
	Comment      string           `json:"comment"`
	CoverURL     string           `json:"cover_url"`
	Cover        json.RawMessage  `json:"cover"`
	URL          string           `json:"url"`
	Year         string           `json:"year"`
	Rating       *DoubanRating    `json:"rating"`
	Photos       []string         `json:"photos"`
	Pic          *DoubanPic       `json:"pic"`
}

type DoubanRating struct {
	Value float64 `json:"value"`
}

type DoubanPic struct {
	Large  string `json:"large"`
	Normal string `json:"normal"`
}

type DoubanSubjectCollection struct {
	SubjectCollectionItems []DoubanSubjectCollectionItem `json:"subject_collection_items"`
	Total                  int                           `json:"total"`
}

type DoubanSubjectCollectionInfo struct {
	CategoryTabs []DoubanCategoryTab `json:"category_tabs"`
}

type DoubanCategoryTab struct {
	Items []DoubanCategoryItem `json:"items"`
}

type DoubanCategoryItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Current bool   `json:"current"`
}

type DoubanSubjectCollectionCategory struct {
	Items []DoubanCategoryItem `json:"items"`
}

type DoubanDoulistItem struct {
	ID       FlexInt        `json:"id"`
	TargetID FlexInt        `json:"target_id"`
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Subtitle string         `json:"subtitle"`
	Comment  string         `json:"comment"`
	CoverURL string         `json:"cover_url"`
	URL      string         `json:"url"`
	Rating   *DoubanRating  `json:"rating"`
}

type DoubanDoulist struct {
	DoulistItems []DoubanDoulistItem `json:"items"`
	Total        int                 `json:"total"`
}
