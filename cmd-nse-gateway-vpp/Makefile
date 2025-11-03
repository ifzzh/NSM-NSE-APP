.PHONY: all build clean test lint fmt help

# 项目配置
PROJECT_NAME := cmd-nse-gateway-vpp
BINARY_NAME := $(PROJECT_NAME)
BIN_DIR := bin
CMD_DIR := cmd
INTERNAL_DIR := internal
TESTS_DIR := tests

# Go配置
GO := go
GOFLAGS := -v
LDFLAGS := -w -s
BUILD_FLAGS := $(GOFLAGS) -ldflags="$(LDFLAGS)"

# 默认目标
all: clean build test

# 帮助信息
help:
	@echo "Gateway NSE Makefile"
	@echo ""
	@echo "可用目标:"
	@echo "  make build      - 编译二进制文件到 $(BIN_DIR)/$(BINARY_NAME)"
	@echo "  make test       - 运行所有单元测试"
	@echo "  make clean      - 清理构建产物"
	@echo "  make lint       - 运行代码检查（需要安装golangci-lint）"
	@echo "  make fmt        - 格式化代码"
	@echo "  make all        - 执行 clean + build + test"
	@echo ""

# 编译二进制文件
build:
	@echo "====================================="
	@echo "编译 $(PROJECT_NAME)..."
	@echo "====================================="
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo ""
	@echo "✓ 编译成功: $(BIN_DIR)/$(BINARY_NAME)"
	@echo ""

# 运行测试
test:
	@echo "====================================="
	@echo "运行单元测试..."
	@echo "====================================="
	$(GO) test -v -cover ./$(TESTS_DIR)/unit/...
	@echo ""
	@echo "✓ 测试完成"
	@echo ""

# 运行测试（带覆盖率报告）
test-coverage:
	@echo "====================================="
	@echo "运行测试并生成覆盖率报告..."
	@echo "====================================="
	@mkdir -p coverage
	$(GO) test -v -coverprofile=coverage/coverage.out ./$(TESTS_DIR)/unit/...
	$(GO) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo ""
	@echo "✓ 覆盖率报告已生成: coverage/coverage.html"
	@echo ""

# 清理构建产物
clean:
	@echo "====================================="
	@echo "清理构建产物..."
	@echo "====================================="
	@rm -rf $(BIN_DIR)
	@rm -rf coverage
	@rm -f *.log
	@echo "✓ 清理完成"
	@echo ""

# 代码格式化
fmt:
	@echo "====================================="
	@echo "格式化代码..."
	@echo "====================================="
	$(GO) fmt ./...
	@echo "✓ 格式化完成"
	@echo ""

# 代码检查（需要安装golangci-lint）
lint:
	@echo "====================================="
	@echo "运行代码检查..."
	@echo "====================================="
	@which golangci-lint > /dev/null || (echo "错误: golangci-lint 未安装。请运行: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...
	@echo "✓ 代码检查完成"
	@echo ""

# 依赖管理
deps:
	@echo "====================================="
	@echo "下载依赖..."
	@echo "====================================="
	$(GO) mod download
	@echo "✓ 依赖下载完成"
	@echo ""

# 整理依赖
tidy:
	@echo "====================================="
	@echo "整理依赖..."
	@echo "====================================="
	$(GO) mod tidy
	@echo "✓ 依赖整理完成"
	@echo ""

# 验证依赖
verify:
	@echo "====================================="
	@echo "验证依赖..."
	@echo "====================================="
	$(GO) mod verify
	@echo "✓ 依赖验证完成"
	@echo ""
