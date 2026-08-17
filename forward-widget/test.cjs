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
    year: "2025", genres: ["恐怖", "奇幻"],
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
    year: "2011",
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
const tmdbMovieFixtures = [{ id: "tmdb:movie:969681", type: "movie", name: "蜘蛛侠", year: "2026", poster: tmdbPoster }];
const tmdbTvFixtures = [{ id: "tmdb:tv:108978", type: "series", name: "侠探杰克", year: "2022", poster: tmdbPoster }];
const animeFixtures = [{ id: "tmdb:movie:1315772", type: "series", name: "小黄人与大怪兽", year: "2026", poster: tmdbPoster }];

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
  const tmdbTv = await loadList({ catalog: "series:tmdb_trending_tv" });
  assert.equal(tmdbTv[0].id, "108978");
  assert.equal(tmdbTv[0].mediaType, "tv");

  // 11. 动漫：id 前缀是真实类型（movie），即使 type 字段是 series 也用 id 前缀
  const anime = await loadList({ catalog: "series:tmdb_discover_anime" });
  assert.equal(anime[0].id, "1315772");
  assert.equal(anime[0].mediaType, "movie");

  console.log("✅ ok", { calls: calls.length });
})().catch((e) => { console.error("❌", e); process.exit(1); });
