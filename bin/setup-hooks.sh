#!/bin/bash
# setup-hooks.sh — 一键配置 git hooks
#
# 在新 clone 项目后运行此脚本，自动配置 .githooks 目录
# 两种模式：
#   1. 自动模式：直接运行即可
#   2. 手动模式：指定 hooks 路径
#
# 使用：
#   bash bin/setup-hooks.sh
#   bash bin/setup-hooks.sh --global  # 同时设置 global config

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOKS_DIR="$REPO_ROOT/.githooks"
GLOBAL_MODE=false

if [ "$1" = "--global" ] || [ "$1" = "-g" ]; then
    GLOBAL_MODE=true
fi

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║          Git Hooks 安装脚本                                 ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# 检查 .githooks 目录
if [ ! -d "$HOOKS_DIR" ]; then
    echo "❌ .githooks 目录不存在: $HOOKS_DIR"
    echo "   请确保在项目根目录运行此脚本"
    exit 1
fi

# 检查 hook 文件
hooks_found=$(find "$HOOKS_DIR" -maxdepth 1 -type f ! -name "*.md" ! -name "*.txt" ! -name "*.sample" 2>/dev/null | wc -l)
if [ "$hooks_found" -eq 0 ]; then
    echo "❌ .githooks 目录为空，未找到 hook 文件"
    exit 1
fi

# 配置 local hooks
echo "✅ 配置 local hooks: git config core.hooksPath .githooks"
git config core.hooksPath ".githooks"

# 配置 global hooks（可选）
if [ "$GLOBAL_MODE" = true ]; then
    echo "✅ 配置 global hooks: git config --global core.hooksPath $HOOKS_DIR"
    git config --global core.hooksPath "$HOOKS_DIR"
    echo ""
    echo "⚠️  注意: global hooks 会影响所有 git 仓库"
    echo "   如需只为 freekiosk 项目使用 hooks，请不要加 --global"
fi

# 确保 hook 文件可执行
find "$HOOKS_DIR" -maxdepth 1 -type f -exec chmod +x {} \; 2>/dev/null

echo ""
echo "✅ Hooks 安装完成!"
echo ""
echo "可用的 hooks:"
find "$HOOKS_DIR" -maxdepth 1 -type f ! -name "*.md" ! -name "*.txt" ! -name "*.sample" -exec basename {} \; | sort | while read hook; do
    echo "   • $hook"
done

echo ""
echo "已配置的 hook 路径:"
git config core.hooksPath

echo ""
echo "💡 提示: "
echo "   • commit-msg: 验证 commit message 格式"
echo "   • prepare-commit-msg: 自动显示格式提示"
echo "   • post-feature.sh: 功能完成后运行此脚本生成文档"
echo ""
