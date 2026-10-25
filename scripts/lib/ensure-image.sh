#!/usr/bin/env bash
# dev.sh / dev-docker.sh 共用的镜像预拉:本地无镜像时优先走 daoCloud 加速源
# (直连 Docker Hub 常超时),再 tag 回标准名。source 后以函数形式调用:
#   ensure_image postgres:16-alpine
ensure_image() {
  local name="$1"
  if ! docker image inspect "$name" >/dev/null 2>&1; then
    echo "  拉取镜像 $name(经 daoCloud 加速)..."
    docker pull "docker.m.daocloud.io/library/$name"
    docker tag "docker.m.daocloud.io/library/$name" "$name"
  fi
}
