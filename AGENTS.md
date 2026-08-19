# AGENTS.md

本仓库提供豆瓣榜单 Go 服务（catalog API）+ ForwardWidget 插件（`forward-widget/douban.js`）。对外经 emby-proxy `/doubanapi` 反代（rn.127315.xyz:4000）。

## 命令速查

| 场景 | 命令 |
|------|------|
| 开发 | `go run ./cmd/server`（PORT 由 `.env` 决定，默认 4000） |
| 构建 | `go build -o server ./cmd/server` |
| 测试 | `go test ./...` |
| widget 离线回测 | `cd forward-widget && node test.cjs` |
| widget live 回测 | 先本地起服务（如 `PORT=4101 go run ./cmd/server`），再 `cd forward-widget && node test-live.cjs`（BASE 指向该端口） |

## 路由约定（catalog）

- `GET /catalog/:type/:catalogID.json`；`:type` ∈ `movie`/`series` 为兼容保留，实际类型由 catalogID 决定
- **extra 参数必须编码在 `.json` 前的路径段**：`/catalog/movie/tmdb_discover_movie/with_networks=213&sort_by=popularity.desc.json?skip=0`。`parseExtra` 只解析路径段内的 `key=value&...`，query 仅有 `skip`/`genre` 特例——其余参数放 query 会被静默丢弃（历史踩坑）
- catalogID 类型：
  - 豆瓣收藏 ID：`movie_hot_gaia`、`tv_hot` 等（`internal/collection/collections.go`）
  - 年度榜单：`__movie_yearly_ranking__` / `__tv_yearly_ranking__`（自动取最新年份）
  - 豆瓣 doulist 剧场：数字 ID（`internal/collection/theaters.go`）
  - TMDB 分类（`IsTmdb` 配置）：
    - `tmdb_trending_movie` / `tmdb_trending_tv`：Trending 周榜，**不支持平台过滤**
    - `tmdb_discover_anime`：动画（movie+tv 合并）
    - `tmdb_discover_movie` / `tmdb_discover_tv`：Discover，支持平台参数白名单透传（`with_networks`/`with_watch_providers`/`watch_region`/`with_genres`/`without_genres`/`sort_by`/`vote_count.gte`/日期），白名单外参数忽略
- 响应带 `CacheMaxAge: 86400` + `StaleRevalidate/StaleError` 1 周；UA 含 `forward` 时 ID 用 `tmdb:xxx` 形式

## 环境变量（`.env`，gitignore）

`DOUBAN_API_KEY`、`TMDB_API_KEY`、`TRAKT_CLIENT_ID`、`FANART_API_KEY`、`PORT`（4000）、`PROXY_URL`（默认 `https://proxy.laoz.org/url`，所有外部请求经它转发）

## 部署（Go 节点）

| 项 | 值 |
|----|----|
| host | rn.127315.xyz |
| port | 22222 |
| user | root |
| 密钥 | ~/.ssh/syu_vps |
| 部署目录 | /root/docker/douban-api |
| 容器 | douban-api（0.0.0.0:4000） |

步骤：
1. `git fetch && git reset --hard origin/main`（兼容 force push，**不可用 `git pull`**）
2. `docker compose up -d --build`
3. 验证：`curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:4000/catalog/series/tv_hot.json` → `200`；`docker logs --tail 5 douban-api` 无启动错误

⚠️ `/root/docker/douban` 是 **addon（syuim/stremio-addon-douban）** 的部署目录（带前端，v2.x），与本仓库无关，勿混淆。

## ForwardWidget 插件（`forward-widget/douban.js`）

- 8 个模块；模块 6（TMDB）/模块 8（影视平台）的平台筛选走 `tmdb_discover_*` catalog（extra 路径段形式），其余走豆瓣榜单
- 修改 widget 后必须：递增 `WidgetMetadata.version`，跑 `test.cjs` 与 `test-live.cjs`
- 默认服务地址 `https://proxy.laoz.org/doubanapi`（emby-proxy 反代，仅 JSON catalog）
