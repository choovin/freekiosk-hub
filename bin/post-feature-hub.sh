#!/bin/bash
# post-feature.sh — freekiosk-hub 专用功能完成工作流
#
# 与 freekiosk 版本的主要区别：
#   - 版本管理使用 git tag（而非 package.json）
#   - 知识库目录使用 docs/knowledge/
#   - CHANGELOG.md 位于项目根目录
#
# 使用：
#   cd freekiosk-hub && bash ../bin/post-feature-hub.sh

set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
KNOWLEDGE_DIR="$REPO_ROOT/knowledge"
CHANGELOG="$REPO_ROOT/CHANGELOG.md"
GO_MOD="$REPO_ROOT/go.mod"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

info()    { echo -e "${BLUE}[ℹ️  INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[✅ SUCCESS]${NC}  $*"; }
warn()    { echo -e "${YELLOW}[⚠️  WARN]${NC}  $*"; }
error()   { echo -e "${RED}[❌ ERROR]${NC}  $*"; }
section() { echo ""; echo -e "${BOLD}${CYAN}══ $1 ══${NC}"; }

check_git() {
    if [ ! -d ".git" ]; then
        error "不是 git 仓库：$(pwd)"
        exit 1
    fi
}

check_clean() {
    if [ -n "$(git status --porcelain)" ]; then
        error "工作区不干净，请先 commit 所有更改"
        git status --short
        exit 1
    fi
}

get_branch() { git branch --show-current; }

# 获取最新 tag 版本
get_version() {
    local latest
    latest=$(git tag --sort=-version:refname 'v*' 2>/dev/null | head -1)
    if [ -z "$latest" ]; then
        echo "v0.0.0"
    else
        echo "$latest"
    fi
}

# 递增版本号（返回新版本）
bump_version() {
    local bump_type="${1:-patch}"
    local current
    current=$(get_version | sed 's/^v//')
    local major minor patch
    IFS='.' read -r major minor patch <<< "$current"
    case "$bump_type" in
        major) major=$((major + 1)); minor=0; patch=0 ;;
        minor) minor=$((minor + 1)); patch=0 ;;
        patch) patch=$((patch + 1)) ;;
    esac
    local new_version="v$major.$minor.$patch"
    success "版本已更新: $(get_version) → $new_version"
    echo "$new_version"
}

generate_filename() {
    local title="$1"
    local timestamp
    timestamp=$(date '+%Y%m%d-%H%M%S')
    local safe_title
    safe_title=$(echo "$title" | sed 's/[^a-zA-Z0-9\u4e00-\u9fa5]/-/g' | tr 'A-Z' 'a-z' | sed 's/--*/-/g' | sed 's/^-\|-\$//g' | cut -c1-50)
    echo "${timestamp}-${safe_title}.md"
}

get_change_type() {
    local branch
    branch=$(get_branch)
    if echo "$branch" | grep -qE "feat"; then echo "feat"
    elif echo "$branch" | grep -qE "fix"; then echo "fix"
    elif echo "$branch" | grep -qE "docs?"; then echo "docs"
    elif echo "$branch" | grep -qE "ui"; then echo "ui"
    elif echo "$branch" | grep -qE "api"; then echo "api"
    elif echo "$branch" | grep -qE "refactor"; then echo "refactor"
    else echo "feat"
    fi
}

generate_doc() {
    local feature_title="$1"
    local feature_desc="$2"
    local impact="$3"
    local type="$4"
    local filename
    filename=$(generate_filename "$feature_title")
    local today
    today=$(date '+%Y-%m-%d')
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local author
    author=$(git config user.name || echo "Unknown")

    mkdir -p "$KNOWLEDGE_DIR"

    info "生成文档: $KNOWLEDGE_DIR/$filename"

    cat > "$KNOWLEDGE_DIR/$filename" << EOF
# ${feature_title}

**日期:** ${today}
**类型:** ${type}
**影响范围:** ${impact}
**负责人:** ${author}

---

## 概述

${feature_desc}

---

## 变更详情

### 修改的文件

\`\`\`
$(git diff --name-only HEAD~1..HEAD 2>/dev/null | sed 's/^/  /')
\`\`\`

### 核心变更

-

---

## 技术实现

### 关键代码

\`\`\`go
//
\`\`\`

### API 变更（如有）

| 接口 | 方法 | 说明 |
|------|------|------|
| | | |

---

## 测试验证

- [ ] 功能测试：
- [ ] 回归测试：

---

## 部署注意事项

-
-

---

*本文档由 post-feature-hub.sh 自动生成于 ${timestamp}*
EOF
    success "文档已生成: $filename"
    echo "$filename"
}

update_changelog() {
    local version="$1"
    local type="$2"
    local feature_title="$3"
    local feature_desc="$4"

    info "更新 CHANGELOG.md"

    local changelog_entry="## [${version}] - $(date '+%Y-%m-%d')

### ${type^}

- **${feature_title}** ${feature_desc}"

    if grep -q "## \[Unreleased\]" "$CHANGELOG"; then
        local temp_file
        temp_file=$(mktemp)
        local marker_found=false
        while IFS= read -r line; do
            echo "$line" >> "$temp_file"
            if [ "$marker_found" = false ] && echo "$line" | grep -q "## \[Unreleased\]"; then
                marker_found=true
                IFS= read -r next_line
                echo "$next_line" >> "$temp_file"
                echo "" >> "$temp_file"
                echo "$changelog_entry" >> "$temp_file"
            fi
        done < "$CHANGELOG"
        mv "$temp_file" "$CHANGELOG"
    else
        local temp_file
        temp_file=$(mktemp)
        echo "$changelog_entry" >> "$temp_file"
        echo "" >> "$temp_file"
        cat "$CHANGELOG" >> "$temp_file"
        mv "$temp_file" "$CHANGELOG"
    fi

    success "CHANGELOG.md 已更新"
}

create_commit() {
    local type="$1"
    local scope="$2"
    local description="$3"
    local doc_file="$4"

    local scope_part=""
    [ -n "$scope" ] && scope_part="($scope)"

    local commit_msg="${type}${scope_part}: ${description}"

    info "Git commit: $commit_msg"

    git add -A
    git commit -m "$commit_msg" -m "
Co-Authored-By: $(git config user.name) <$(git config user.email)>

自动生成 by post-feature-hub.sh"
}

main() {
    echo ""
    echo -e "${BOLD}${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${CYAN}║          freekiosk-hub 功能发布工作流                     ║${NC}"
    echo -e "${BOLD}${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""

    check_git
    check_clean

    local branch
    branch=$(get_branch)
    info "当前分支: $branch"
    info "当前版本: $(get_version)"
    echo ""

    section "Step 1/5 — 功能基本信息"
    echo -n -e "${BOLD}功能标题${NC}: "
    read -r feature_title
    [ -z "$feature_title" ] && { error "标题不能为空"; exit 1; }

    echo -n -e "${BOLD}功能描述${NC}: "
    read -r feature_desc
    [ -z "$feature_desc" ] && feature_desc="$feature_title"

    echo -n -e "${BOLD}影响范围${NC}: "
    read -r impact
    [ -z "$impact" ] && impact="hub"
    echo ""

    section "Step 2/5 — 变更类型"
    local default_type
    default_type=$(get_change_type)
    echo "请选择变更类型 (直接回车使用: ${default_type}):"
    echo "  1) feat   新功能"
    echo "  2) fix    Bug 修复"
    echo "  3) docs   文档更新"
    echo "  4) ui     UI 变更"
    echo "  5) api    API 变更"
    echo "  6) db     数据库变更"
    echo -n "选择 [1-6]: "
    read -r type_choice

    local change_type="$default_type"
    case "$type_choice" in
        1) change_type="feat" ;;
        2) change_type="fix" ;;
        3) change_type="docs" ;;
        4) change_type="ui" ;;
        5) change_type="api" ;;
        6) change_type="db" ;;
    esac
    echo -e "${GREEN}变更类型: $change_type${NC}"
    echo ""

    section "Step 3/5 — 变更范围 (可选)"
    echo -n -e "${BOLD}范围${NC} (如: api, ui, db 等): "
    read -r scope
    [ -z "$scope" ] && info "跳过范围" || info "范围: $scope"
    echo ""

    section "Step 4/5 — 版本号递增"
    echo "当前版本: $(get_version)"
    echo "请选择版本递增类型:"
    echo "  1) patch 补丁版本"
    echo "  2) minor 次版本 (新功能)"
    echo "  3) major 主版本 (破坏性变更)"
    echo -n "选择 [1-3]: "
    read -r version_choice

    local bump_type="patch"
    case "$version_choice" in
        2) bump_type="minor" ;;
        3) bump_type="major" ;;
    esac

    local new_version
    new_version=$(bump_version "$bump_type")
    echo ""

    section "Step 5/5 — 生成文档"
    local doc_file
    doc_file=$(generate_doc "$feature_title" "$feature_desc" "$impact" "$change_type")
    echo ""

    section "更新 Changelog"
    update_changelog "$new_version" "$change_type" "$feature_title" "$feature_desc"
    echo ""

    section "Git Commit"
    local scope_part=""
    [ -n "$scope" ] && scope_part="($scope)"
    create_commit "$change_type" "$scope" "$feature_title" "$doc_file"
    echo ""

    section "完成"
    echo -e "${GREEN}✅ 版本: $new_version${NC}"
    echo -e "${GREEN}✅ 文档: $KNOWLEDGE_DIR/$doc_file${NC}"
    echo -e "${GREEN}✅ Commit: $change_type${scope_part}: $feature_title${NC}"
    echo ""
    echo -e "${YELLOW}下一步操作:${NC}"
    echo "  1. 运行测试: go test ./..."
    echo "  2. 推送代码: git push"
    echo "  3. 打 tag: git tag $new_version && git push origin $new_version"
    echo ""
}

main "$@"
