// 回测：mock Widget 全局，验证 douban.js 的请求构造与字段映射
// 运行：node test.js
const fs = require("fs");
const assert = require("assert/strict");

const calls = [];
global.Widget = {
  http: {
    get: async (url, options) => {
      calls.push({ url, headers: options && options.headers });
      if (url.includes("/catalog/movie/movie_hot_gaia.json")) {
        return { data: { metas: movieFixtures } };
      }
      if (url.includes("/catalog/series/ECFA5DI7Q.json")) {
        return { data: { metas: seriesFixtures } };
      }
      if (url.includes("/catalog/movie/__random__.json")) {
        return { data: { metas: [doubanOnlyFixture] } };
      }
      if (url.includes("/catalog/series/128396349.json")) {
        return { data: { metas: seriesFixtures } };
      }
      if (url.includes("/catalog/movie/tmdb_trending_movie.json")) {
        return { data: { metas: tmdbMovieFixtures } };
      }
      if (url.includes("/catalog/series/tmdb_trending_tv.json")) {
        return { data: { metas: tmdbTvFixtures } };
      }
      if (url.includes("/catalog/series/tmdb_discover_anime.json")) {
        return { data: { metas: animeFixtures } };
      }
      throw new Error("unmocked url: " + url);
    },
  },
  tmdb: {
    // tmdb.get 直接返回 body（无 .data 包装），params 记录在 opts.params
    get: async (api, opts) => {
      calls.push({ api, params: opts && opts.params });
      if (api.includes("/discover/")) return platformFixtures;
      throw new Error("unmocked tmdb api: " + api);
    },
  },
  storage: { _m: {}, get(k) { return this._m[k]; }, set(k, v) { this._m[k] = v; } },
};
global.WidgetMetadata = {};
eval(fs.readFileSync("./douban.js", "utf8"));

// 真实响应结构（精简自 rn.127315.xyz:31001）
const movieFixtures = [
  {
    id: "tmdb:1339713", type: "movie", name: "痴迷",
    poster: "https://proxy.laoz.org/url?url=https%3A%2F%2Fimage.tmdb.org%2Ft%2Fp%2Foriginal%2FpoCoOo8VYoWzBAtLjhdNGQE42fa.jpg",
    background: "https://proxy.laoz.org/url?url=https%3A%2F%2Fimage.tmdb.org%2Ft%2Fp%2Foriginal%2FdCJ4Igrbnc35QeNKi3FUEt7nAJm.jpg",
    year: "2025", genres: ["恐怖", "奇幻"], genre_ids: [27, 14],
    links: [{ name: "豆瓣评分：7.6", category: "douban", url: "https://www.douban.com/doubanapp/dispatch/movie/37450627" }],
  },
  { id: "tt37287335", type: "movie", name: "无 imdb 兜底示例", year: "2024",
    poster: "https://proxy.laoz.org/url?url=https%3A%2F%2Fimage.tmdb.org%2Ft%2Fp%2Foriginal%2Ftt1.jpg" },
  { id: "douban:37450627", type: "movie", name: "纯豆瓣条目", year: "2023",
    poster: "https://proxy.laoz.org/url?url=https%3A%2F%2Fimage.tmdb.org%2Ft%2Fp%2Foriginal%2Fdb1.jpg" },
  { id: "tmdb:999", type: "movie", name: "无海报条目", year: "2020" },
];
const seriesFixtures = [
  {
    id: "tmdb:1399", type: "series", name: "权游",
    poster: "https://proxy.laoz.org/url?url=https%3A%2F%2Fimage.tmdb.org%2Ft%2Fp%2Foriginal%2Fabc.jpg",
    year: "2011", genre_ids: [10765, 18],
    links: [{ name: "豆瓣评分：9.3", category: "douban", url: "#" }],
  },
];
const doubanOnlyFixture = {
  id: "tmdb:1083381", type: "movie", name: "后室", year: "2026",
  poster: "https://proxy.laoz.org/url?url=https%3A%2F%2Fimage.tmdb.org%2Ft%2Fp%2Foriginal%2Fhs.jpg",
  links: [{ name: "豆瓣评分：0", category: "douban", url: "#" }],
};
// TMDB catalog 返回三段式 id（tmdb:movie:X / tmdb:tv:X），动漫条目 id 前缀为真实类型但 type 显示 series
const tmdbPoster = "https://proxy.laoz.org/url?url=https%3A%2F%2Fimage.tmdb.org%2Ft%2Fp%2Foriginal%2Ftmdb.jpg";
const tmdbMovieFixtures = [{ id: "tmdb:movie:969681", type: "movie", name: "蜘蛛侠", year: "2026", poster: tmdbPoster, genre_ids: [28, 878] }];
const tmdbTvFixtures = [{ id: "tmdb:tv:108978", type: "series", name: "侠探杰克", year: "2022", poster: tmdbPoster, genre_ids: [80, 18] }];
const animeFixtures = [{ id: "tmdb:movie:1315772", type: "series", name: "小黄人与大怪兽", year: "2026", poster: tmdbPoster, genre_ids: [16, 10751] }];
// 影视平台模块：TMDB discover 标准响应（genre_ids 混合综艺 10764 用于类型标签断言）
const platformFixtures = {
  results: [
    { id: 123456, name: "平台剧集示例", first_air_date: "2025-01-15", vote_average: 8.4,
      genre_ids: [18, 10764], poster_path: "/p1.jpg", backdrop_path: "/b1.jpg" },
    { id: 234567, title: "平台电影示例", release_date: "2025-03-01", vote_average: 7.2,
      genre_ids: [28, 878], poster_path: "/p2.jpg" },
  ],
};

(async () => {
  // 1. 分页换算与请求构造
  const list = await loadList({ page: 2, catalog: "movie:movie_hot_gaia" });
  assert.equal(calls[0].url, "https://proxy.laoz.org/douban/catalog/movie/movie_hot_gaia.json?skip=20");
  assert.equal(calls[0].headers["User-Agent"], "forward widget/1.0");

  // 2. tmdb id 输出字符串数字（与 MakkaPakka 一致，数字类型会被 App 拒收）+ mediaType
  assert.equal(list[0].id, "1339713");
  assert.equal(typeof list[0].id, "string");
  assert.equal(list[0].type, "tmdb");
  assert.equal(list[0].mediaType, "movie");

  // 3. posterPath 还原为文件名（App 自行拼域名+尺寸，不能带 /t/p/original 前缀）
  assert.equal(list[0].posterPath, "/poCoOo8VYoWzBAtLjhdNGQE42fa.jpg");
  assert.equal(list[0].backdropPath, "/dCJ4Igrbnc35QeNKi3FUEt7nAJm.jpg");
  assert.equal(list[0].poster, undefined);

  // 4. 评分解析
  assert.equal(list[0].rating, 7.6);

  // 4.1 TMDB 标准分类：genre_ids → 中文（GENRE_MAP 映射）
  assert.equal(list[0].genreTitle, "恐怖 / 奇幻");
  assert.equal(list[1].genreTitle, undefined); // 无 genre_ids 不输出

  // 5. imdb / douban 兜底；无海报条目被过滤
  assert.equal(list.length, 3); // 无海报条目被过滤
  assert.equal(list[1].type, "imdb");
  assert.equal(list[1].id, "tt37287335");
  assert.equal(list[2].type, "douban");
  assert.equal(list[2].id, "37450627");

  // 6. series → tv
  const series = await loadList({ page: 1, catalog: "series:ECFA5DI7Q" });
  assert.ok(calls[1].url.includes("/catalog/series/ECFA5DI7Q.json?skip=0"));
  assert.equal(series[0].id, "1399");
  assert.equal(series[0].mediaType, "tv");
  assert.equal(series[0].rating, 9.3);
  assert.equal(series[0].genreTitle, "剧情"); // 10765 未映射被过滤

  // 7. 默认值：不传参数也能跑
  const defaults = await loadList({});
  assert.ok(calls[2].url.includes("/catalog/movie/movie_hot_gaia.json?skip=0"));

  // 8. globalParams server 覆盖
  const custom = await loadList({ server: "http://example.com:9999/", catalog: "movie:__random__", page: 3 });
  assert.equal(custom.length, 1); // __random__ 返回单条
  assert.ok(calls[3].url.startsWith("http://example.com:9999/catalog/movie/__random__.json?skip=40"));
  assert.equal(custom[0].id, "1083381");
  // rating 0（未开分）不输出评分字段
  assert.equal(custom[0].rating, undefined);

  // 9. 平台剧场模块（doulist catalog）
  const theater = await loadList({ catalog: "series:128396349" });
  assert.ok(calls[4].url.includes("/catalog/series/128396349.json?skip=0"));
  assert.equal(theater[0].id, "1399");
  assert.equal(theater[0].mediaType, "tv");

  // 10. TMDB 三段式 id：tmdb:movie:X / tmdb:tv:X 拆出纯数字 id 与真实 mediaType
  const tmdbMovie = await loadList({ catalog: "movie:tmdb_trending_movie" });
  assert.equal(tmdbMovie[0].id, "969681");
  assert.equal(tmdbMovie[0].mediaType, "movie");
  assert.equal(tmdbMovie[0].genreTitle, "动作 / 科幻");
  const tmdbTv = await loadList({ catalog: "series:tmdb_trending_tv" });
  assert.equal(tmdbTv[0].id, "108978");
  assert.equal(tmdbTv[0].mediaType, "tv");
  assert.equal(tmdbTv[0].genreTitle, "犯罪 / 剧情");

  // 11. 动漫：id 前缀是真实类型（movie），即使 type 字段是 series 也用 id 前缀
  const anime = await loadList({ catalog: "series:tmdb_discover_anime" });
  assert.equal(anime[0].id, "1315772");
  assert.equal(anime[0].mediaType, "movie");
  assert.equal(anime[0].genreTitle, "动画 / 家庭");

  // 12. 影视平台：剧集走 with_networks（Netflix=213）+ 纯净剧集过滤 + 热度排序
  const platformTv = await loadPlatformList({ sort_by: "netflix", mediaType: "tv", sortBy: "hot", page: 1 });
  const tvCall = calls[calls.length - 1];
  assert.equal(tvCall.api, "/discover/tv");
  assert.equal(tvCall.params.with_networks, "213");
  assert.equal(tvCall.params.without_genres, "16,10764,10767");
  assert.equal(tvCall.params.sort_by, "popularity.desc");
  assert.equal(tvCall.params.language, "zh-CN");
  assert.equal(platformTv.length, 2);

  // 13. 影视平台：字段映射（字符串 id + raw 海报路径 + 数字评分 + 不泄漏原始键）
  const tv0 = platformTv[0];
  assert.equal(tv0.id, "123456");
  assert.equal(typeof tv0.id, "string");
  assert.equal(tv0.type, "tmdb");
  assert.equal(tv0.mediaType, "tv");
  assert.equal(tv0.posterPath, "/p1.jpg");
  assert.equal(tv0.backdropPath, "/b1.jpg");
  assert.equal(tv0.poster_path, undefined);
  assert.equal(tv0.rating, 8.4);
  assert.equal(typeof tv0.rating, "number");
  assert.equal(tv0.releaseDate, "2025-01-15");
  assert.ok(tv0.description.includes("Netflix"));
  assert.ok(tv0.description.includes("⭐ 8.4"));
  // mediaType 由请求场景决定：/discover/tv 返回的条目全部映射为 tv
  assert.equal(platformTv[1].mediaType, "tv");

  // 14. 影视平台：电影走 with_watch_providers + watch_region；动漫/综艺走 with_genres
  const platformMovie = await loadPlatformList({ sort_by: "tencent", mediaType: "movie", sortBy: "top", page: 2 });
  let movieCall = calls[calls.length - 1];
  assert.equal(movieCall.api, "/discover/movie");
  assert.equal(movieCall.params.with_watch_providers, "138");
  assert.equal(movieCall.params.watch_region, "CN");
  assert.equal(movieCall.params.sort_by, "vote_average.desc");
  assert.equal(movieCall.params["vote_count.gte"], 30);
  assert.equal(movieCall.params.page, 2);
  assert.equal(platformMovie[0].mediaType, "movie");
  await loadPlatformList({ sort_by: "hbo", mediaType: "anime", sortBy: "new" });
  let animeCall = calls[calls.length - 1];
  assert.equal(animeCall.api, "/discover/tv");
  assert.equal(animeCall.params.with_networks, "49|3186");
  assert.equal(animeCall.params.with_genres, "16");
  assert.equal(animeCall.params.sort_by, "first_air_date.desc");
  assert.ok(animeCall.params["first_air_date.lte"]);

  // 15. 影视平台：纯电视网（ViuTV）无电影 provider → 空数组，不请求
  const viutvMovie = await loadPlatformList({ sort_by: "viutv", mediaType: "movie" });
  assert.equal(viutvMovie.length, 0);

  console.log("✅ ok", { calls: calls.length });
})().catch((e) => { console.error("❌", e); process.exit(1); });
