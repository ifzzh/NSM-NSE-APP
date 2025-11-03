# ==================================
# 构建阶段
# ==================================
FROM golang:1.23.8-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache make git

# 设置工作目录
WORKDIR /workspace

# 复制go.mod和go.sum（利用Docker缓存）
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY tests/ ./tests/
COPY Makefile ./

# 编译二进制文件
RUN make build

# ==================================
# 运行时阶段
# ==================================
FROM gcr.io/distroless/static-debian11:latest

# 从构建阶段复制二进制文件
COPY --from=builder /workspace/bin/cmd-nse-gateway-vpp /cmd-nse-gateway-vpp

# 设置入口点
ENTRYPOINT ["/cmd-nse-gateway-vpp"]

# 元数据标签
LABEL org.opencontainers.image.title="Gateway NSE"
LABEL org.opencontainers.image.description="IP-based Gateway Network Service Endpoint for NSM"
LABEL org.opencontainers.image.version="1.0.0"
LABEL org.opencontainers.image.vendor="Network Service Mesh"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# 文档信息
LABEL description="IP网关NSE - 基于IP白名单/黑名单策略的简化网关端点"
LABEL maintainer="NSM Team"
