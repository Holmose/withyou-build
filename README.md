# withyou-build

Internal GitHub Actions build pipeline for `Holmose/withyou-chat-app`.

Only the workflow YAML lives here. The actual source code is cloned from a private repository at build time using a Personal Access Token stored as a Secret — no source code is visible in this repository.

## What this repository contains
- `.github/workflows/build-and-publish-amd64.yml` — the only file

## What this repository does NOT contain
- Application source code
- Dockerfile
- frontend/out artifacts
- Anything else from the source repository

## Configuration (Secrets)

Set the following repository-level Secrets in this repository's settings:

| Secret | Purpose |
|--------|---------|
| `SOURCE_REPO` | Private source repository (e.g. `Holmose/withyou-chat-app`) — full `owner/repo` slug |
| `SOURCE_REPO_PAT` | GitHub Personal Access Token with read access to the source repo |
| `ACR_USERNAME` | Aliyun ACR username |
| `ACR_PASSWORD` | Aliyun ACR password |

## Trigger

- `workflow_dispatch` — manual run with optional `git_commit` and `go_image` inputs
- `push` to `main` — automatic run on workflow changes

## Output

Three tags are pushed to `registry.cn-beijing.aliyuncs.com/withyou_holmose/deeix-chat-withyou`:

- `v2.2.33-amd64`
- `latest-amd64`
- `latest`