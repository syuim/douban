// live 回测：真实请求本地服务(4100)，验证 genreTitle / 字段映射
const fs = require("fs");
const assert = require("assert/strict");

const BASE = "http://127.0.0.1:4100";
global.Widget = {
  http: {
    get: async (url, options) => {
      const res = await fetch(url, { headers: { "User-Agent": "forward widget/1.0" } });
      return { data: await res.json() };
    },
  },
  storage: { _m: {}, get(k) { return this._m[k]; }, set(k, v) { this._m[k] = v; } },
};
global.WidgetMetadata = {};
eval(fs.readFileSync("./douban.js", "utf8"));

(async () => {
  // 1. 豆瓣榜单：genreTitle 由服务端 genre_ids 映射
  const list = await loadList({ server: BASE, catalog: "movie:movie_hot_gaia" });
  assert.ok(list.length > 0);
  const withGenre = list.filter((it) => it.genreTitle);
  console.log(`movie_hot_gaia: ${list.length} items, ${withGenre.length} with genreTitle`);
  assert.ok(withGenre.length > 0, "应至少有一条 genreTitle");
  const sample = withGenre[0];
  assert.equal(sample.type, "tmdb");
  assert.equal(typeof sample.id, "string");
  assert.ok(!/^https?:/.test(sample.posterPath || ""), "posterPath 必须是 raw 路径");
  console.log("  示例:", sample.title, "|", sample.genreTitle, "|", sample.posterPath);

  // 2. TMDB 榜单：genreTitle 全覆盖
  const tmdb = await loadList({ server: BASE, catalog: "movie:tmdb_trending_movie" });
  const tmdbGenre = tmdb.filter((it) => it.genreTitle);
  console.log(`tmdb_trending_movie: ${tmdb.length} items, ${tmdbGenre.length} with genreTitle`);
  assert.equal(tmdbGenre.length, tmdb.length, "TMDB 榜单应全部带 genreTitle");
  assert.equal(tmdb[0].mediaType, "movie");
  assert.ok(tmdb[0].genreTitle.includes(" / ") || tmdb[0].genreTitle.length > 0);

  // 3. 剧集：mediaType tv
  const tv = await loadList({ server: BASE, catalog: "series:tv_hot" });
  assert.ok(tv.every((it) => it.mediaType === "tv"));
  console.log(`tv_hot: ${tv.length} items, ${tv.filter((it) => it.genreTitle).length} with genreTitle`);

  // 4. 影视平台模块：直连 TMDB（走真实 Widget.tmdb 会失败——本测试无 tmdb mock，跳过）

  console.log("✅ live ok");
})().catch((e) => { console.error("❌", e); process.exit(1); });
