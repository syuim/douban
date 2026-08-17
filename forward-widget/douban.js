// 豆瓣榜单 Widget：数据来自自建 Go 服务（douban）的 catalog 接口
// 请求带 forward UA，服务端返回 tmdb 优先的 ID 与已代理的图片 URL
WidgetMetadata = {
  id: "com.douban.discovery",
  title: "豆瓣榜单",
  version: "1.0.5",
  requiredVersion: "0.0.1",
  description: "豆瓣热门电影、口碑榜、剧场与剧集榜单",
  author: "suyu",
  site: "https://github.com/InchStudio/ForwardWidgets",
  globalParams: [
    { name: "server", title: "服务地址", type: "input", value: "https://proxy.laoz.org/douban" },
  ],
  modules: [
    {
      id: "douban_random",
      title: "随便看看",
      type: "video",
      description: "随机混排多个榜单",
      functionName: "loadList",
      cacheDuration: 0, // 调试阶段不缓存；上线前改回 3600
      params: [
        { name: "catalog", title: "榜单", type: "constant", value: "movie:__random__" },
        { name: "page", title: "页码", type: "page", startPage: 1 },
      ],
    },
    {
      id: "douban_hot_tv",
      title: "热门剧集",
      type: "video",
      description: "近期热门剧集、动画、综艺与纪录片",
      functionName: "loadList",
      cacheDuration: 21600, // 6 小时缓存
      params: [
        {
          name: "catalog",
          title: "榜单",
          type: "enumeration",
          value: "series:tv_hot",
          enumOptions: [
            { title: "近期热门剧集", value: "series:tv_hot" },
            { title: "近期热门美剧", value: "series:tv_american" },
            { title: "近期热门韩剧", value: "series:tv_korean" },
            { title: "近期热门国产剧", value: "series:tv_domestic" },
            { title: "近期热门日剧", value: "series:tv_japanese" },
            { title: "近期热门动画", value: "series:tv_animation" },
            { title: "近期热门综艺节目", value: "series:show_hot" },
            { title: "近期热门纪录片", value: "series:tv_documentary" },
            { title: "实时热门电视", value: "series:tv_real_time_hotest" },
          ],
        },
        { name: "page", title: "页码", type: "page", startPage: 1 },
      ],
    },
    {
      id: "douban_hot_movie",
      title: "热门电影",
      type: "video",
      description: "豆瓣热门、口碑、实时、Top250 与影院热映",
      functionName: "loadList",
      cacheDuration: 21600, // 6 小时缓存
      params: [
        {
          name: "catalog",
          title: "榜单",
          type: "enumeration",
          value: "movie:movie_hot_gaia",
          enumOptions: [
            { title: "豆瓣热门电影", value: "movie:movie_hot_gaia" },
            { title: "一周口碑电影榜", value: "movie:movie_weekly_best" },
            { title: "实时热门电影", value: "movie:movie_real_time_hotest" },
            { title: "豆瓣电影 Top250", value: "movie:movie_top250" },
            { title: "影院热映", value: "movie:movie_showing" },
          ],
        },
        { name: "page", title: "页码", type: "page", startPage: 1 },
      ],
    },
    {
      id: "douban_tv_genres",
      title: "剧集类型榜",
      type: "video",
      description: "口碑剧集、地区剧榜与年度剧集",
      functionName: "loadList",
      cacheDuration: 21600, // 6 小时缓存
      params: [
        {
          name: "catalog",
          title: "榜单",
          type: "enumeration",
          value: "series:tv_chinese_best_weekly",
          enumOptions: [
            { title: "华语口碑剧集榜", value: "series:tv_chinese_best_weekly" },
            { title: "全球口碑剧集榜", value: "series:tv_global_best_weekly" },
            { title: "国内口碑综艺榜", value: "series:show_chinese_best_weekly" },
            { title: "国外口碑综艺榜", value: "series:show_global_best_weekly" },
            { title: "大陆剧榜", value: "series:EC74443FY" },
            { title: "美剧榜", value: "series:ECFA5DI7Q" },
            { title: "英剧榜", value: "series:ECVACXBWI" },
            { title: "日剧榜", value: "series:ECNA46YBA" },
            { title: "韩剧榜", value: "series:ECBE5CBEI" },
            { title: "港剧榜", value: "series:ECVM47WUA" },
            { title: "台剧榜", value: "series:ECBI5EL6A" },
            { title: "欧洲剧榜", value: "series:EC6I5FYHA" },
            { title: "豆瓣年度评分最高剧集", value: "series:__tv_yearly_ranking__" },
          ],
        },
        { name: "page", title: "页码", type: "page", startPage: 1 },
      ],
    },
    {
      id: "douban_movie_genres",
      title: "电影类型榜",
      type: "video",
      description: "26 个电影类型榜单与年度电影",
      functionName: "loadList",
      cacheDuration: 21600, // 6 小时缓存
      params: [
        {
          name: "catalog",
          title: "类型",
          type: "enumeration",
          value: "movie:movie_comedy",
          enumOptions: [
            { title: "剧情片榜", value: "movie:film_genre_27" },
            { title: "喜剧片榜", value: "movie:movie_comedy" },
            { title: "爱情片榜", value: "movie:movie_love" },
            { title: "动作片榜", value: "movie:movie_action" },
            { title: "科幻片榜", value: "movie:movie_scifi" },
            { title: "动画片榜", value: "movie:film_genre_31" },
            { title: "悬疑片榜", value: "movie:film_genre_32" },
            { title: "犯罪片榜", value: "movie:film_genre_46" },
            { title: "惊悚片榜", value: "movie:film_genre_33" },
            { title: "冒险片榜", value: "movie:film_genre_49" },
            { title: "家庭片榜", value: "movie:film_genre_41" },
            { title: "儿童片榜", value: "movie:film_genre_42" },
            { title: "历史片榜", value: "movie:film_genre_44" },
            { title: "音乐片榜", value: "movie:film_genre_39" },
            { title: "奇幻片榜", value: "movie:film_genre_48" },
            { title: "恐怖片榜", value: "movie:film_genre_34" },
            { title: "战争片榜", value: "movie:film_genre_45" },
            { title: "传记片榜", value: "movie:film_genre_43" },
            { title: "歌舞片榜", value: "movie:film_genre_40" },
            { title: "武侠片榜", value: "movie:film_genre_50" },
            { title: "情色片榜", value: "movie:film_genre_37" },
            { title: "灾难片榜", value: "movie:natural_disasters" },
            { title: "西部片榜", value: "movie:film_genre_47" },
            { title: "古装片榜", value: "movie:film_genre_51" },
            { title: "运动片榜", value: "movie:ECCEPGM4Y" },
            { title: "短片榜", value: "movie:film_genre_36" },
            { title: "豆瓣年度评分最高电影", value: "movie:__movie_yearly_ranking__" },
          ],
        },
        { name: "page", title: "页码", type: "page", startPage: 1 },
      ],
    },
    {
      id: "douban_tmdb",
      title: "TMDB",
      type: "video",
      description: "TMDB 热门电影、剧集与动漫",
      functionName: "loadList",
      cacheDuration: 21600, // 6 小时缓存
      params: [
        {
          name: "catalog",
          title: "榜单",
          type: "enumeration",
          value: "movie:tmdb_trending_movie",
          enumOptions: [
            { title: "TMDB 本周热门电影", value: "movie:tmdb_trending_movie" },
            { title: "TMDB 本周热门剧集", value: "series:tmdb_trending_tv" },
            { title: "TMDB 热门动漫", value: "series:tmdb_discover_anime" },
          ],
        },
        { name: "page", title: "页码", type: "page", startPage: 1 },
      ],
    },
    {
      id: "douban_theater",
      title: "平台剧场",
      type: "video",
      description: "各平台剧场片单",
      functionName: "loadList",
      cacheDuration: 21600, // 6 小时缓存
      params: [
        {
          name: "catalog",
          title: "剧场",
          type: "enumeration",
          value: "series:128396349",
          enumOptions: [
            { title: "迷雾剧场", value: "series:128396349" },
            { title: "白夜剧场", value: "series:158539495" },
            { title: "X剧场", value: "series:155026800" },
            { title: "横屏短剧", value: "series:152299516" },
            { title: "生花剧场", value: "series:159069554" },
            { title: "大家剧场", value: "series:160644809" },
            { title: "小逗剧场", value: "series:146055365" },
            { title: "十分剧场", value: "series:147708618" },
            { title: "板凳单元", value: "series:163392459" },
            { title: "萤火单元", value: "series:163549603" },
            { title: "正午阳光", value: "series:125370543" },
            { title: "恋恋剧场", value: "series:156086548" },
            { title: "悬疑剧场", value: "series:128400108" },
            { title: "微尘剧场", value: "series:161658331" },
          ],
        },
        { name: "page", title: "页码", type: "page", startPage: 1 },
      ],
    },
  ],
};

async function loadList(params = {}) {
  try {
    const server = (params.server || "https://proxy.laoz.org/douban").replace(/\/+$/, "");
    const [type, catalogId] = String(params.catalog || "movie:movie_hot_gaia").split(":");
    const page = Math.max(1, Number(params.page || 1));
    const skip = (page - 1) * 20;

    const url = `${server}/catalog/${type}/${catalogId}.json?skip=${skip}`;
    const res = await Widget.http.get(url, { headers: { "User-Agent": "forward widget/1.0" } });
    const metas = (res.data && res.data.metas) || [];
    // 过滤无 tmdb 海报的条目（豆瓣图/缺图），App 无法展示且可能整批拒收
    return metas.map(toVideoItem).filter((it) => it.posterPath);
  } catch (error) {
    console.error("[loadList] 失败:", error && error.message ? error.message : error);
    throw error;
  }
}

// 服务端返回 id 形态：tmdb:123（普通榜单）、tmdb:movie:123 / tmdb:tv:123（TMDB 榜单）、
// 纯 imdb "ttxxx"、douban:123（兜底）。id 统一输出字符串数字，App 据此打开内置详情页
function toVideoItem(m) {
  const id = String(m.id || "");
  const fallbackMediaType = m.type === "series" ? "tv" : "movie";
  const item = {
    type: "tmdb",
    title: m.name,
    posterPath: imagePath(m.poster),
    backdropPath: imagePath(m.background),
    releaseDate: m.year,
  };

  if (id.startsWith("tmdb:")) {
    const parts = id.split(":");
    if (parts.length >= 3 && (parts[1] === "movie" || parts[1] === "tv")) {
      item.id = parts[2];
      item.mediaType = parts[1];
    } else {
      item.id = parts[1];
      item.mediaType = fallbackMediaType;
    }
  } else if (id.startsWith("douban:")) {
    item.id = id.slice(7);
    item.type = "douban";
    item.mediaType = fallbackMediaType;
  } else {
    item.id = id;
    item.type = "imdb";
    item.mediaType = fallbackMediaType;
  }

  const rating = parseRating(m.links);
  if (rating != null && rating > 0) {
    item.rating = rating;
  }
  return item;
}

// 图片为代理完整 URL，tmdb 类型只需文件名（App 自行拼接域名与尺寸）
function imagePath(url) {
  if (!url) return undefined;
  const decoded = decodeURIComponent(url);
  const m = /https?:\/\/image\.tmdb\.org\/[^?#]*\/([^/?#]+\.(?:jpg|jpeg|png|webp))/.exec(decoded);
  return m ? "/" + m[1] : undefined;
}

// links: [{ name: "豆瓣评分：7.6", category: "douban", url }]
function parseRating(links) {
  if (!Array.isArray(links)) return null;
  for (const link of links) {
    if (link && typeof link.name === "string" && link.name.startsWith("豆瓣评分：")) {
      const n = parseFloat(link.name.slice("豆瓣评分：".length));
      if (!Number.isNaN(n)) return n;
    }
  }
  return null;
}
