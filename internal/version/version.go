package version

import (
	"encoding/json"
	"os"
	"sync"
)

// fallback 仅在 package.json 缺失（如非仓库目录运行）时兜底，需与 package.json 同步。
const fallback = "2.1.26"

var (
	once sync.Once
	v    string
)

// Get 返回 package.json 的 version 字段，作为对外版本信息的单一来源。
func Get() string {
	once.Do(func() {
		v = fallback
		data, err := os.ReadFile("package.json")
		if err != nil {
			return
		}
		var pkg struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &pkg) == nil && pkg.Version != "" {
			v = pkg.Version
		}
	})
	return v
}
