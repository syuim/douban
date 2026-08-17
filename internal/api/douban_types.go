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

type DoubanSubjectDetail struct {
	ID          FlexInt            `json:"id"`
	Type        string             `json:"type"`
	Title       string             `json:"title"`
	OriginalTitle string           `json:"original_title"`
	Intro       string             `json:"intro"`
	CoverURL    string             `json:"cover_url"`
	URL         string             `json:"url"`
	Year        string             `json:"year"`
	Genres      []string           `json:"genres"`
	Countries   []string           `json:"countries"`
	Languages   []string           `json:"languages"`
	Directors   []DoubanPerson     `json:"directors"`
	Actors      []DoubanPerson     `json:"actors"`
	Rating      *DoubanRating      `json:"rating"`
	Linewatches []DoubanLinewatch  `json:"linewatches"`
	HonorInfos  []DoubanHonorInfo  `json:"honor_infos"`
	Pic         *DoubanPic         `json:"pic"`
	Photos      []string           `json:"photos"`
}

type DoubanPerson struct {
	Name string `json:"name"`
}

type DoubanLinewatch struct {
	Source     DoubanLinewatchSource `json:"source"`
	SourceURI  string                `json:"source_uri"`
}

type DoubanLinewatchSource struct {
	Name string `json:"name"`
}

type DoubanHonorInfo struct {
	Title string `json:"title"`
}

type DoubanModule struct {
	ModuleName string `json:"module_name"`
	Data       struct {
		SelectedCollections []DoubanSelectedCollection `json:"selected_collections"`
	} `json:"data"`
}

type DoubanSelectedCollection struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	IsMergedCover  bool   `json:"is_merged_cover"`
}

type DoubanModulesResponse struct {
	Modules []DoubanModule `json:"modules"`
}
