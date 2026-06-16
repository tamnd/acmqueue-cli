# acmq

Read ACM Queue practitioner technology articles.

`acmq` is a single pure-Go binary. It speaks to ACM Queue over its public RSS
feed, shapes the responses into clean records, and pipes into the rest of your
tools. No API key, no JavaScript, no authentication required.

## Install

```bash
go install github.com/tamnd/acmqueue-cli/cmd/acmq@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/acmqueue-cli/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/acmq:latest top -n 5
```

## Commands

```
acmq top [-n N]                     Most recent articles from the RSS feed (default 20)
acmq topics                         ACM Queue editorial topic taxonomy (static, no HTTP)
acmq article <id-or-url>            Single article by numeric ID or full URL
acmq version [--short]              Print version, commit, build date, OS/arch, Go version
```

## Examples

```bash
# List the 5 most recent articles
acmq top -n 5

# List in JSON
acmq top -n 5 -o json

# Narrow to specific fields, pipe into jq
acmq top -o jsonl | jq .title

# Show topic taxonomy
acmq topics

# Look up an article by numeric ID
acmq article 3807964 -o json

# Look up an article by full URL
acmq article 'https://queue.acm.org/detail.cfm?ref=rss&id=3807964'

# Stream URLs for the latest articles
acmq top -o url
```

## Output formats

Every command supports `-o table|json|jsonl|csv|tsv|url|raw`.
Default is `table` on a TTY and `jsonl` when output is piped.
Use `--fields` to select columns and `--template` for Go text/template output.

## Data source

All data comes from the public RSS feed at
`https://queue.acm.org/rss/feeds/queuecontent.xml`. The feed holds the 20
most recent articles with title, URL, pubDate, and a numeric article ID.
Author, abstract, and topic are not present in the feed.

Article detail pages sit behind Cloudflare and are not fetched.

## Development

```
cmd/acmq/        thin main, wires cli.Root into fang
cli/             the cobra command tree
acmqueue/        the library: HTTP client, RSS parser, data models
pkg/render/      output renderer (table, json, jsonl, csv, tsv, url, raw)
docs/            tago documentation site
```

```bash
make build      # ./bin/acmq
make test       # go test -race ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

Apache-2.0. See [LICENSE](LICENSE).
