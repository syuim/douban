// live 回测：真实请求本地服务(4100)，验证 genreTitle / 字段映射
const fs = require("fs");
const assert = require("assert/strict");

const BASE = "http://127.0.0.1:4101";
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

  // 4. 影视平台模块（模块 8）：走 catalog tmdb_discover
  const pl = await loadPlatformList({ server: BASE, sort_by: "netflix", mediaType: "tv", sortBy: "hot" });
  assert.ok(pl.length > 0);
  assert.ok(pl.every((it) => it.type === "tmdb" && it.mediaType === "tv"), "平台模块应全部 tmdb/tv");
  assert.ok(pl.every((it) => it.genreTitle), "平台模块应全部带 genreTitle");
  assert.ok(pl.some((it) => it.rating > 0), "平台模块应解析出 TMDB 评分");
  console.log(`loadPlatformList(netflix/tv/hot): ${pl.length} items`);
  console.log("  示例:", pl[0].title, "|", pl[0].genreTitle, "|", pl[0].rating, "|", pl[0].posterPath);
  assert.ok(!/^https?:/.test(pl[0].posterPath || ""), "posterPath 必须是 raw 路径");

  // 5. 模块 6 平台筛选：trending catalog + platform 参数转 tmdb_discover
  const pf = await loadList({ server: BASE, catalog: "movie:tmdb_trending_movie", platform: "netflix" });
  assert.ok(pf.length > 0);
  assert.ok(pf.every((it) => it.mediaType === "movie"), "平台筛选应全为 movie");
  console.log(`loadList(platform=netflix): ${pf.length} items, 示例: ${pf[0].title} | ${pf[0].genreTitle} | ${pf[0].rating}`);

  // 6. 平台=全部时保持原 trending 行为
  const all = await loadList({ server: BASE, catalog: "movie:tmdb_trending_movie", platform: "all" });
  assert.ok(all.length > 0, "platform=all 应正常返回");
  console.log(`loadList(platform=all): ${all.length} items`);

  console.log("✅ live ok");
})().catch((e) => { console.error("❌", e); process.exit(1); });
