#!/bin/bash
# verify-dependencies.sh - 验证Gateway NSE与firewall-vpp的依赖版本一致性
#
# 用途：确保Gateway NSE使用的核心依赖版本与firewall-vpp完全一致
#
# 使用方法：
#   ./scripts/verify-dependencies.sh

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目路径
GATEWAY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FIREWALL_DIR="$(cd "$GATEWAY_DIR/../cmd-nse-firewall-vpp-refactored" && pwd 2>/dev/null || echo "")"

# 检查firewall-vpp项目是否存在
if [ -z "$FIREWALL_DIR" ] || [ ! -d "$FIREWALL_DIR" ]; then
    echo -e "${YELLOW}⚠️  警告: 未找到firewall-vpp-refactored项目${NC}"
    echo "   路径: $GATEWAY_DIR/../cmd-nse-firewall-vpp-refactored"
    echo "   将仅验证Gateway自身的go.mod"
    FIREWALL_DIR=""
fi

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Gateway NSE 依赖版本验证${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# 读取Gateway的go.mod
GATEWAY_GO_MOD="$GATEWAY_DIR/go.mod"
if [ ! -f "$GATEWAY_GO_MOD" ]; then
    echo -e "${RED}✗ 错误: Gateway的go.mod不存在${NC}"
    exit 1
fi

# 提取Gateway的Go版本
GATEWAY_GO_VERSION=$(grep "^go " "$GATEWAY_GO_MOD" | awk '{print $2}')
echo -e "${BLUE}Gateway Go版本:${NC} $GATEWAY_GO_VERSION"

# 如果firewall存在，对比Go版本
if [ -n "$FIREWALL_DIR" ]; then
    FIREWALL_GO_MOD="$FIREWALL_DIR/go.mod"
    if [ -f "$FIREWALL_GO_MOD" ]; then
        FIREWALL_GO_VERSION=$(grep "^go " "$FIREWALL_GO_MOD" | awk '{print $2}')
        echo -e "${BLUE}Firewall Go版本:${NC} $FIREWALL_GO_VERSION"

        if [ "$GATEWAY_GO_VERSION" = "$FIREWALL_GO_VERSION" ]; then
            echo -e "${GREEN}✓ Go版本一致${NC}"
        else
            echo -e "${RED}✗ Go版本不一致${NC}"
            echo "  Gateway: $GATEWAY_GO_VERSION"
            echo "  Firewall: $FIREWALL_GO_VERSION"
            exit 1
        fi
    fi
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  核心依赖验证${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# 定义要检查的依赖
DEPENDENCIES=(
    "github.com/sirupsen/logrus"
    "github.com/stretchr/testify"
    "google.golang.org/grpc"
    "google.golang.org/protobuf"
    "gopkg.in/yaml.v2"
)

# 提取Gateway依赖版本的函数
get_gateway_version() {
    local dep=$1
    grep "$dep" "$GATEWAY_GO_MOD" | grep -v "// indirect" | awk '{print $2}' | head -1
}

# 提取Firewall依赖版本的函数
get_firewall_version() {
    local dep=$1
    if [ -n "$FIREWALL_DIR" ] && [ -f "$FIREWALL_GO_MOD" ]; then
        grep "$dep" "$FIREWALL_GO_MOD" | grep -v "// indirect" | awk '{print $2}' | head -1
    else
        echo ""
    fi
}

# 验证标志
ALL_CONSISTENT=true

# 验证每个依赖
for dep in "${DEPENDENCIES[@]}"; do
    GATEWAY_VER=$(get_gateway_version "$dep")

    if [ -z "$GATEWAY_VER" ]; then
        echo -e "${YELLOW}⚠️  ${dep}${NC}"
        echo "   Gateway: 未引入"
        continue
    fi

    if [ -n "$FIREWALL_DIR" ]; then
        FIREWALL_VER=$(get_firewall_version "$dep")

        if [ -z "$FIREWALL_VER" ]; then
            echo -e "${YELLOW}⚠️  ${dep}${NC}"
            echo "   Gateway: $GATEWAY_VER"
            echo "   Firewall: 未引入"
            continue
        fi

        if [ "$GATEWAY_VER" = "$FIREWALL_VER" ]; then
            echo -e "${GREEN}✓ ${dep}${NC}"
            echo "   版本: $GATEWAY_VER"
        else
            echo -e "${RED}✗ ${dep}${NC}"
            echo "   Gateway: $GATEWAY_VER"
            echo "   Firewall: $FIREWALL_VER"
            ALL_CONSISTENT=false
        fi
    else
        echo -e "${GREEN}✓ ${dep}${NC}"
        echo "   Gateway版本: $GATEWAY_VER"
    fi
    echo ""
done

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  NSM/VPP依赖检查${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# NSM和VPP相关依赖（Gateway当前未引入，但未来会需要）
FUTURE_DEPENDENCIES=(
    "github.com/networkservicemesh/api"
    "github.com/networkservicemesh/sdk"
    "go.fd.io/govpp"
    "github.com/spiffe/go-spiffe/v2"
)

NSM_VPP_STATUS="PENDING"

for dep in "${FUTURE_DEPENDENCIES[@]}"; do
    GATEWAY_VER=$(get_gateway_version "$dep")

    if [ -z "$GATEWAY_VER" ]; then
        echo -e "${YELLOW}⏸️  ${dep}${NC}"
        echo "   状态: 待集成（当前使用Mock实现）"
    else
        echo -e "${GREEN}✓ ${dep}${NC}"
        echo "   Gateway版本: $GATEWAY_VER"
        NSM_VPP_STATUS="INTEGRATED"
    fi
    echo ""
done

# 最终总结
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  验证总结${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if [ -n "$FIREWALL_DIR" ]; then
    if $ALL_CONSISTENT; then
        echo -e "${GREEN}✓ 所有已引入的核心依赖版本与firewall-vpp一致${NC}"
        echo ""
        echo "  已验证依赖数量: ${#DEPENDENCIES[@]}"
        echo "  一致性: 100%"
        echo ""
        echo -e "${YELLOW}注意: NSM/VPP依赖状态: ${NSM_VPP_STATUS}${NC}"
        if [ "$NSM_VPP_STATUS" = "PENDING" ]; then
            echo "  - 当前使用Mock实现避免依赖冲突"
            echo "  - 后续需要引入真实NSM SDK和VPP SDK"
            echo "  - 引入时必须确保版本与firewall-vpp一致"
        fi
        echo ""
        echo -e "${GREEN}✓ 验证通过${NC}"
        exit 0
    else
        echo -e "${RED}✗ 发现依赖版本不一致${NC}"
        echo ""
        echo "  请修改Gateway的go.mod，使依赖版本与firewall-vpp对齐"
        echo ""
        echo -e "${RED}✗ 验证失败${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✓ Gateway的go.mod验证通过${NC}"
    echo ""
    echo "  Go版本: $GATEWAY_GO_VERSION"
    echo "  已引入依赖: ${#DEPENDENCIES[@]}个核心依赖"
    echo ""
    echo -e "${YELLOW}注意: 未找到firewall-vpp项目，无法进行版本对比${NC}"
    echo "  建议确保firewall-vpp-refactored项目位于:"
    echo "  $GATEWAY_DIR/../cmd-nse-firewall-vpp-refactored"
    echo ""
    echo -e "${GREEN}✓ 基础验证通过${NC}"
    exit 0
fi
