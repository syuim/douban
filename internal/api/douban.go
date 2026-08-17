package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"stremio-addon-douban/internal/model"
)

type DoubanAPI struct {
	*BaseAPI
}

var doubanHeaders = map[string]string{
	"Referer":    "https://servicewechat.com/wx2f9b06c1de1ccfca/99/page-frame.html",
	"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Mac MacWechat/WMPF MacWechat/3.8.7(0x13080712) UnifiedPCMacWechat(0xf264101d) XWEB/16390",
}

const DoubanPageSize = 20

func NewDoubanAPI() *DoubanAPI {
	return &DoubanAPI{
		BaseAPI: NewBaseAPI("https://frodo.douban.com/api/v2", doubanHeaders),
	}
}

func (d *DoubanAPI) apiKey() string {
	return os.Getenv("DOUBAN_API_KEY")
}

func (d *DoubanAPI) params(extra map[string]string) map[string]string {
	p := map[string]string{"apiKey": d.apiKey()}
	for k, v := range extra {
		p[k] = v
	}
	return p
}

func (d *DoubanAPI) GetSubjectCollection(ctx context.Context, collectionID string) (*DoubanSubjectCollectionInfo, error) {
	var result DoubanSubjectCollectionInfo
	err := d.RequestJSON(ctx, "GET", "/subject_collection/"+collectionID,
		d.params(map[string]string{"for_mobile": "1"}), nil, nil,
		&CacheConfig{Key: "subject_collection_info:" + collectionID, TTL: model.SecondsDayPlusBuffer},
		&result)
	return &result, err
}

func (d *DoubanAPI) GetSubjectCollectionCategory(ctx context.Context, collectionID string) (*DoubanSubjectCollectionCategory, error) {
	cacheKey := "subject_collection_category:" + collectionID
	var cached DoubanSubjectCollectionCategory
	if data, ok, _ := d.getCache(ctx, cacheKey); ok {
		if err := jsonUnmarshal(data, &cached); err == nil {
			return &cached, nil
		}
	}

	info, err := d.GetSubjectCollection(ctx, collectionID)
	if err != nil {
		return nil, err
	}

	for _, tab := range info.CategoryTabs {
		for _, item := range tab.Items {
			if item.Current {
				category := &DoubanSubjectCollectionCategory{Items: tab.Items}
				d.setCacheJSON(cacheKey, category, model.SecondsPerWeek*4)
				return category, nil
			}
		}
	}
	return nil, nil
}

// GetDoulistItems 分页抓取豆瓣片单（doulist）全部条目，返回 subject 列表，
// 复用统一代理/缓存/重试链路；每页 25 条，按 total 自动翻页。
// maxPages 为 0 时拉全量，大于 0 时最多拉取指定页数。
func (d *DoubanAPI) GetDoulistItems(ctx context.Context, doulistID string, maxPages int) ([]DoubanDoulistItem, error) {
	var all []DoubanDoulistItem
	start := 0
	const pageSize = 25
	for page := 1; maxPages == 0 || page <= maxPages; page++ {
		var result DoubanDoulist
		err := d.RequestJSON(ctx, "GET", fmt.Sprintf("/doulist/%s/items", doulistID),
			d.params(map[string]string{
				"start": fmt.Sprintf("%d", start),
				"count": fmt.Sprintf("%d", pageSize),
			}), nil, nil,
			&CacheConfig{Key: fmt.Sprintf("doulist:%s:%d", doulistID, start), TTL: model.SecondsDayPlusBuffer},
			&result)
		if err != nil {
			return nil, err
		}
		all = append(all, result.DoulistItems...)
		if maxPages > 0 && page >= maxPages {
			break
		}
		if len(result.DoulistItems) < pageSize || result.Total <= start+len(result.DoulistItems) {
			break
		}
		start += pageSize
	}
	return all, nil
}

func (d *DoubanAPI) GetSubjectCollectionItems(ctx context.Context, collectionID string, skip int) (*DoubanSubjectCollection, error) {
	var result DoubanSubjectCollection
	err := d.RequestJSON(ctx, "GET",
		fmt.Sprintf("/subject_collection/%s/items", collectionID),
		d.params(map[string]string{
			"start": fmt.Sprintf("%d", skip),
			"count": fmt.Sprintf("%d", DoubanPageSize),
		}), nil, nil,
		&CacheConfig{Key: fmt.Sprintf("subject_collection:%s:%d", collectionID, skip), TTL: model.SecondsDayPlusBuffer},
		&result)
	if err != nil {
		return nil, err
	}
	// post-process items: resolve cover URL, extract year
	for i := range result.SubjectCollectionItems {
		item := &result.SubjectCollectionItems[i]
		// cover priority: cover.url > cover_url > pic.large > pic.normal
		if len(item.Cover) > 0 {
			var coverStr string
			if json.Unmarshal(item.Cover, &coverStr) == nil {
				if coverStr != "" {
					item.CoverURL = coverStr
				}
			} else {
				var coverObj struct {
					URL string `json:"url"`
				}
				if json.Unmarshal(item.Cover, &coverObj) == nil && coverObj.URL != "" {
					item.CoverURL = coverObj.URL
				}
			}
		}
		if item.CoverURL == "" && item.Pic != nil {
			if item.Pic.Large != "" {
				item.CoverURL = item.Pic.Large
			} else if item.Pic.Normal != "" {
				item.CoverURL = item.Pic.Normal
			}
		}
		if item.Year == "" && item.CardSubtitle != "" {
			parts := strings.Split(item.CardSubtitle, "/")
			if len(parts) > 0 {
				item.Year = strings.TrimSpace(parts[0])
			}
		}
		if item.Description == "" {
			item.Description = item.Comment
		}
		if item.Description == "" {
			item.Description = item.CardSubtitle
		}
		if len(item.Photos) == 0 && item.Pic != nil && item.Pic.Large != "" {
			item.Photos = []string{item.Pic.Large}
		}
	}
	return &result, nil
}

func (d *DoubanAPI) GetSubjectDetail(ctx context.Context, subjectID int) (*DoubanSubjectDetail, error) {
	var result DoubanSubjectDetail
	err := d.RequestJSON(ctx, "GET", fmt.Sprintf("/subject/%d", subjectID),
		d.params(nil), nil, nil,
		&CacheConfig{Key: fmt.Sprintf("subject_detail:%d", subjectID), TTL: model.SecondsDayPlusBuffer},
		&result)
	return &result, err
}

func (d *DoubanAPI) GetSubjectDetailDesc(ctx context.Context, subjectID int) (map[string]string, error) {
	var raw struct {
		HTML string `json:"html"`
	}
	err := d.RequestJSON(ctx, "GET", fmt.Sprintf("/subject/%d/desc", subjectID),
		d.params(nil), nil, nil,
		&CacheConfig{Key: fmt.Sprintf("subject_detail_desc:%d", subjectID), TTL: model.SecondsDayPlusBuffer},
		&raw)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw.HTML))
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	doc.Find(".subject-desc table tr").Each(func(_ int, s *goquery.Selection) {
		key := strings.TrimSpace(s.Find("td:first-child").Text())
		value := strings.TrimSpace(s.Find("td:last-child").Text())
		if key != "" {
			result[key] = value
		}
	})
	return result, nil
}

func (d *DoubanAPI) GetModules(ctx context.Context, modType string) (*DoubanModulesResponse, error) {
	var result DoubanModulesResponse
	err := d.RequestJSON(ctx, "GET", "/"+modType+"/modules",
		d.params(nil), nil, nil,
		&CacheConfig{Key: "douban_" + modType + "_modules", TTL: model.SecondsDayPlusBuffer},
		&result)
	return &result, err
}
