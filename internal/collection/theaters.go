package collection

// TheaterConfig 剧场片单配置（豆瓣 doulist ID）
type TheaterConfig struct {
	ID   string
	Name string
}

// TheaterConfigs 全部剧场片单（与用户脚本 THEATERS 一致，新增可扩展）
var TheaterConfigs = []TheaterConfig{
	{ID: "128396349", Name: "迷雾剧场"},
	{ID: "158539495", Name: "白夜剧场"},
	{ID: "155026800", Name: "X剧场"},
	{ID: "152299516", Name: "横屏短剧"},
	{ID: "159069554", Name: "生花剧场"},
	{ID: "160644809", Name: "大家剧场"},
	{ID: "146055365", Name: "小逗剧场"},
	{ID: "147708618", Name: "十分剧场"},
	{ID: "163392459", Name: "板凳单元"},
	{ID: "163549603", Name: "萤火单元"},
	{ID: "125370543", Name: "正午阳光"},
	{ID: "156086548", Name: "恋恋剧场"},
	{ID: "128400108", Name: "悬疑剧场"},
	{ID: "161658331", Name: "微尘剧场"},
}

// FindTheaterConfig 按 ID 查找剧场
func FindTheaterConfig(id string) *TheaterConfig {
	for i := range TheaterConfigs {
		if TheaterConfigs[i].ID == id {
			return &TheaterConfigs[i]
		}
	}
	return nil
}
