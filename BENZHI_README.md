# Docker 运行说明

项目使用 Go 1.25 和标准库，不依赖外部服务。

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh compliance-evidence:arm64 linux/arm64
docker run --rm --platform linux/arm64 compliance-evidence:arm64 go test ./...

./build_benzhi_docker.sh compliance-evidence:amd64 linux/amd64
docker run --rm --platform linux/amd64 compliance-evidence:amd64 go test ./...
```

镜像保留完整 Go 工具链，也可以启动交互 shell 或直接执行 `go run ./cmd/server`。
