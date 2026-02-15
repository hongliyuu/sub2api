#!/bin/bash
# =============================================================================
# Sub2API 安全部署脚本 (Production Safe Deploy)
# =============================================================================
# 功能：
#   1. 部署前备份当前运行的镜像版本
#   2. 拉取最新镜像并启动
#   3. 多阶段健康检查（HTTP / 数据库 / Redis / API 烟雾测试）
#   4. 验收失败自动回滚到上一版本
#   5. 全程日志记录
#
# 用法：
#   ./safe-deploy.sh              # 正常部署
#   ./safe-deploy.sh --rollback   # 手动回滚到上一版本
#   ./safe-deploy.sh --status     # 查看当前状态
#   ./safe-deploy.sh --logs       # 查看部署日志
# =============================================================================

set -euo pipefail

# ===================== 配置 =====================
DEPLOY_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_FILE="docker-compose.local.yml"
SERVICE_NAME="sub2api"
IMAGE_NAME="weishaw/sub2api"
BACKUP_FILE="${DEPLOY_DIR}/.deploy-backup"
LOG_FILE="${DEPLOY_DIR}/deploy.log"

# 健康检查配置
HEALTH_URL="http://localhost:${SERVER_PORT:-8080}/health"
SETUP_URL="http://localhost:${SERVER_PORT:-8080}/setup/status"
MAX_WAIT=120          # 最长等待秒数
CHECK_INTERVAL=3      # 每次检查间隔秒数
SMOKE_TEST_RETRIES=5  # 烟雾测试重试次数

# ===================== 颜色 =====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# ===================== 日志 =====================
log() {
    local level="$1"
    shift
    local msg="$*"
    local timestamp
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[${timestamp}] [${level}] ${msg}" >> "${LOG_FILE}"

    case "$level" in
        INFO)    echo -e "${BLUE}[${timestamp}]${NC} ${msg}" ;;
        OK)      echo -e "${GREEN}[${timestamp}] ✓${NC} ${msg}" ;;
        WARN)    echo -e "${YELLOW}[${timestamp}] ⚠${NC} ${msg}" ;;
        ERROR)   echo -e "${RED}[${timestamp}] ✗${NC} ${msg}" ;;
        STEP)    echo -e "${CYAN}[${timestamp}] ▶${NC} ${msg}" ;;
    esac
}

# ===================== 辅助函数 =====================

# 获取当前运行的镜像 digest
get_current_digest() {
    docker inspect "${SERVICE_NAME}" --format='{{.Image}}' 2>/dev/null || echo "none"
}

# 获取当前镜像 tag 信息
get_current_image() {
    docker inspect "${SERVICE_NAME}" --format='{{index .Config.Image}}' 2>/dev/null || echo "none"
}

# 保存当前状态用于回滚
save_backup() {
    local current_digest
    local current_image
    current_digest=$(get_current_digest)
    current_image=$(get_current_image)

    cat > "${BACKUP_FILE}" << EOF
BACKUP_TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
BACKUP_IMAGE=${current_image}
BACKUP_DIGEST=${current_digest}
EOF
    log INFO "备份当前版本: ${current_image} (${current_digest:0:12})"
}

# 检查容器是否运行中
is_container_running() {
    local status
    status=$(docker inspect -f '{{.State.Status}}' "$1" 2>/dev/null || echo "not_found")
    [ "$status" = "running" ]
}

# ===================== 健康检查阶段 =====================

# 阶段 1: 容器启动检查
check_containers() {
    log STEP "阶段 1/4: 容器启动检查"

    local services=("sub2api" "sub2api-postgres" "sub2api-redis")
    local all_ok=true

    for svc in "${services[@]}"; do
        if is_container_running "$svc"; then
            log OK "容器 ${svc} 运行中"
        else
            log ERROR "容器 ${svc} 未运行"
            all_ok=false
        fi
    done

    $all_ok
}

# 阶段 2: HTTP 健康端点检查
check_health_endpoint() {
    log STEP "阶段 2/4: HTTP 健康端点检查 (最长等待 ${MAX_WAIT}s)"

    local elapsed=0
    while [ $elapsed -lt $MAX_WAIT ]; do
        local response
        local http_code
        http_code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 "${HEALTH_URL}" 2>/dev/null || echo "000")

        if [ "$http_code" = "200" ]; then
            response=$(curl -s --connect-timeout 5 --max-time 10 "${HEALTH_URL}" 2>/dev/null || echo "{}")
            if echo "$response" | grep -q '"status"'; then
                log OK "健康端点返回 200 OK (${elapsed}s)"
                return 0
            fi
        fi

        elapsed=$((elapsed + CHECK_INTERVAL))
        if [ $elapsed -lt $MAX_WAIT ]; then
            printf "\r  等待服务启动... %ds/%ds" "$elapsed" "$MAX_WAIT"
            sleep $CHECK_INTERVAL
        fi
    done
    echo ""
    log ERROR "健康端点在 ${MAX_WAIT}s 内未响应 (最后状态码: ${http_code})"
    return 1
}

# 阶段 3: 数据库与 Redis 连通性检查（通过 Docker 健康状态）
check_dependencies() {
    log STEP "阶段 3/4: 数据库与 Redis 连通性检查"

    local all_ok=true

    # 检查 PostgreSQL 健康状态
    local pg_health
    pg_health=$(docker inspect -f '{{.State.Health.Status}}' sub2api-postgres 2>/dev/null || echo "unknown")
    if [ "$pg_health" = "healthy" ]; then
        log OK "PostgreSQL 健康状态: healthy"
    else
        log ERROR "PostgreSQL 健康状态: ${pg_health}"
        all_ok=false
    fi

    # 检查 Redis 健康状态
    local redis_health
    redis_health=$(docker inspect -f '{{.State.Health.Status}}' sub2api-redis 2>/dev/null || echo "unknown")
    if [ "$redis_health" = "healthy" ]; then
        log OK "Redis 健康状态: healthy"
    else
        log ERROR "Redis 健康状态: ${redis_health}"
        all_ok=false
    fi

    # 检查 sub2api 容器自身的健康检查
    local app_health
    app_health=$(docker inspect -f '{{.State.Health.Status}}' sub2api 2>/dev/null || echo "unknown")
    if [ "$app_health" = "healthy" ]; then
        log OK "Sub2API 容器健康状态: healthy"
    elif [ "$app_health" = "starting" ]; then
        log WARN "Sub2API 容器仍在启动中 (Docker 健康检查尚未通过，但 HTTP 已就绪)"
    else
        log WARN "Sub2API 容器健康状态: ${app_health}"
    fi

    $all_ok
}

# 阶段 4: API 烟雾测试
check_smoke_test() {
    log STEP "阶段 4/4: API 烟雾测试"

    local all_ok=true
    local retry=0

    # 测试 1: /health 端点返回正确结构
    local health_response
    health_response=$(curl -s --connect-timeout 5 --max-time 10 "${HEALTH_URL}" 2>/dev/null || echo "")
    if echo "$health_response" | grep -q '"ok"'; then
        log OK "烟雾测试: /health 返回结构正确"
    else
        log WARN "烟雾测试: /health 返回异常: ${health_response}"
    fi

    # 测试 2: /setup/status 端点
    local setup_response
    setup_response=$(curl -s --connect-timeout 5 --max-time 10 "${SETUP_URL}" 2>/dev/null || echo "")
    if echo "$setup_response" | grep -q '"code"'; then
        log OK "烟雾测试: /setup/status 返回结构正确"
    else
        log WARN "烟雾测试: /setup/status 返回异常: ${setup_response}"
    fi

    # 测试 3: 登录接口可达（发空请求，期望 400 而不是 502/503）
    local login_code
    login_code=$(curl -s -o /dev/null -w "%{http_code}" \
        --connect-timeout 5 --max-time 10 \
        -X POST "${HEALTH_URL%/health}/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{}' 2>/dev/null || echo "000")

    if [ "$login_code" -ge 200 ] && [ "$login_code" -lt 500 ]; then
        log OK "烟雾测试: 登录接口可达 (HTTP ${login_code})"
    elif [ "$login_code" -ge 500 ]; then
        log ERROR "烟雾测试: 登录接口返回服务器错误 (HTTP ${login_code})"
        all_ok=false
    else
        log WARN "烟雾测试: 登录接口无法访问 (HTTP ${login_code})"
    fi

    # 测试 4: 检查容器日志中是否有 panic 或致命错误
    local error_count
    error_count=$(docker logs sub2api --since 60s 2>&1 | grep -ciE 'panic|fatal|segfault' || echo "0")
    if [ "$error_count" -eq 0 ]; then
        log OK "烟雾测试: 最近 60 秒无 panic/fatal 错误"
    else
        log ERROR "烟雾测试: 发现 ${error_count} 条 panic/fatal 错误"
        docker logs sub2api --since 60s 2>&1 | grep -iE 'panic|fatal|segfault' | tail -5 >> "${LOG_FILE}"
        all_ok=false
    fi

    $all_ok
}

# ===================== 完整验收流程 =====================
run_verification() {
    log INFO "========== 开始验收检查 =========="

    local passed=0
    local failed=0

    # 阶段 1
    if check_containers; then
        passed=$((passed + 1))
    else
        failed=$((failed + 1))
        log ERROR "容器检查失败，跳过后续检查"
        return 1
    fi

    # 阶段 2
    if check_health_endpoint; then
        passed=$((passed + 1))
    else
        failed=$((failed + 1))
        log ERROR "健康端点检查失败"
        return 1
    fi

    # 阶段 3
    if check_dependencies; then
        passed=$((passed + 1))
    else
        failed=$((failed + 1))
        log WARN "依赖检查失败（非致命，继续烟雾测试）"
    fi

    # 阶段 4
    if check_smoke_test; then
        passed=$((passed + 1))
    else
        failed=$((failed + 1))
    fi

    echo ""
    log INFO "========== 验收结果 =========="
    log INFO "通过: ${passed}  失败: ${failed}"

    if [ $failed -eq 0 ]; then
        log OK "所有验收检查通过！部署成功"
        return 0
    else
        log ERROR "存在 ${failed} 项失败检查"
        return 1
    fi
}

# ===================== 回滚 =====================
do_rollback() {
    if [ ! -f "${BACKUP_FILE}" ]; then
        log ERROR "无备份信息，无法回滚"
        return 1
    fi

    source "${BACKUP_FILE}"
    log WARN "正在回滚到: ${BACKUP_IMAGE} (备份时间: ${BACKUP_TIMESTAMP})"

    # 用备份的 digest 重新 tag
    if [ "${BACKUP_DIGEST}" != "none" ]; then
        docker tag "${BACKUP_DIGEST}" "${IMAGE_NAME}:rollback" 2>/dev/null || true
    fi

    cd "${DEPLOY_DIR}"

    # 停止当前服务
    docker compose -f "${COMPOSE_FILE}" stop "${SERVICE_NAME}"

    # 如果有 digest 备份，恢复它
    if [ "${BACKUP_DIGEST}" != "none" ]; then
        docker tag "${BACKUP_DIGEST}" "${BACKUP_IMAGE}" 2>/dev/null || true
    fi

    # 重新启动
    docker compose -f "${COMPOSE_FILE}" up -d "${SERVICE_NAME}"

    log INFO "等待回滚后的服务启动..."
    sleep 10

    if check_health_endpoint; then
        log OK "回滚成功！服务已恢复"
        return 0
    else
        log ERROR "回滚后服务仍不健康，请手动检查"
        return 1
    fi
}

# ===================== 查看状态 =====================
show_status() {
    echo ""
    echo -e "${CYAN}========== Sub2API 服务状态 ==========${NC}"
    echo ""

    # 容器状态
    local containers=("sub2api" "sub2api-postgres" "sub2api-redis")
    for c in "${containers[@]}"; do
        local status
        local health
        status=$(docker inspect -f '{{.State.Status}}' "$c" 2>/dev/null || echo "not_found")
        health=$(docker inspect -f '{{.State.Health.Status}}' "$c" 2>/dev/null || echo "N/A")

        if [ "$status" = "running" ]; then
            echo -e "  ${GREEN}●${NC} ${c}: running (health: ${health})"
        else
            echo -e "  ${RED}●${NC} ${c}: ${status}"
        fi
    done

    echo ""

    # 镜像信息
    local image
    image=$(get_current_image)
    local created
    created=$(docker inspect -f '{{.Created}}' sub2api 2>/dev/null | cut -d'T' -f1 || echo "unknown")
    echo "  镜像: ${image}"
    echo "  创建时间: ${created}"

    # 健康检查
    local health_resp
    health_resp=$(curl -s --connect-timeout 3 --max-time 5 "${HEALTH_URL}" 2>/dev/null || echo "unreachable")
    echo "  健康检查: ${health_resp}"

    # 上次部署
    if [ -f "${BACKUP_FILE}" ]; then
        source "${BACKUP_FILE}"
        echo "  上次备份: ${BACKUP_TIMESTAMP:-unknown}"
    fi

    echo ""
    echo -e "${CYAN}======================================${NC}"
    echo ""
}

# ===================== 主部署流程 =====================
do_deploy() {
    log INFO "=========================================="
    log INFO "  Sub2API 安全部署开始"
    log INFO "=========================================="

    cd "${DEPLOY_DIR}"

    # 检查 compose 文件
    if [ ! -f "${COMPOSE_FILE}" ]; then
        log ERROR "未找到 ${COMPOSE_FILE}，请先运行 docker-deploy.sh 初始化"
        exit 1
    fi

    # 检查 .env 文件
    if [ ! -f ".env" ]; then
        log ERROR "未找到 .env 文件，请先配置环境变量"
        exit 1
    fi

    # Step 1: 备份当前版本
    log STEP "Step 1: 备份当前版本"
    if is_container_running sub2api; then
        save_backup
    else
        log INFO "当前无运行中的 sub2api 容器（首次部署）"
    fi

    # Step 2: 拉取最新镜像
    log STEP "Step 2: 拉取最新镜像"
    local old_digest
    old_digest=$(docker inspect --format='{{.Id}}' "${IMAGE_NAME}:latest" 2>/dev/null || echo "none")

    if ! docker compose -f "${COMPOSE_FILE}" pull "${SERVICE_NAME}"; then
        log ERROR "镜像拉取失败，中止部署"
        exit 1
    fi

    local new_digest
    new_digest=$(docker inspect --format='{{.Id}}' "${IMAGE_NAME}:latest" 2>/dev/null || echo "none")

    if [ "$old_digest" = "$new_digest" ] && [ "$old_digest" != "none" ]; then
        log INFO "镜像未更新 (digest 相同)，继续重启以确保配置生效"
    else
        log OK "已拉取新镜像"
    fi

    # Step 3: 启动/重启服务
    log STEP "Step 3: 重启服务"
    docker compose -f "${COMPOSE_FILE}" up -d

    # Step 4: 等待并验收
    log STEP "Step 4: 开始验收检查"
    sleep 5  # 给容器一点启动时间

    if run_verification; then
        echo ""
        log OK "=========================================="
        log OK "  部署成功！所有验收检查通过"
        log OK "=========================================="
        echo ""
        log INFO "访问地址: http://$(curl -s --connect-timeout 3 ifconfig.me 2>/dev/null || echo 'YOUR_SERVER_IP'):${SERVER_PORT:-8080}"
        log INFO "查看日志: docker compose -f ${COMPOSE_FILE} logs -f sub2api"
        echo ""
        return 0
    else
        echo ""
        log ERROR "=========================================="
        log ERROR "  验收失败！启动自动回滚..."
        log ERROR "=========================================="
        echo ""

        if [ -f "${BACKUP_FILE}" ]; then
            do_rollback
        else
            log ERROR "无备份可回滚（首次部署），请手动排查"
            log INFO "查看日志: docker compose -f ${COMPOSE_FILE} logs sub2api"
        fi
        return 1
    fi
}

# ===================== 自动更新模式 =====================
# 仅当检测到新镜像时才部署，适合 cron 定时任务
do_auto_update() {
    cd "${DEPLOY_DIR}"

    if [ ! -f "${COMPOSE_FILE}" ] || [ ! -f ".env" ]; then
        exit 0  # 静默退出，未初始化
    fi

    # 获取当前本地镜像 digest
    local local_digest
    local_digest=$(docker inspect --format='{{.Id}}' "${IMAGE_NAME}:latest" 2>/dev/null || echo "none")

    # 静默拉取远程镜像
    docker pull "${IMAGE_NAME}:latest" > /dev/null 2>&1 || exit 0

    # 获取拉取后的 digest
    local remote_digest
    remote_digest=$(docker inspect --format='{{.Id}}' "${IMAGE_NAME}:latest" 2>/dev/null || echo "none")

    # 如果 digest 相同则无需更新
    if [ "$local_digest" = "$remote_digest" ]; then
        exit 0
    fi

    # 发现新版本，执行安全部署
    log INFO "检测到新版本镜像，启动自动安全部署..."
    log INFO "旧版本: ${local_digest:0:12}"
    log INFO "新版本: ${remote_digest:0:12}"
    do_deploy
}

# ===================== 安装/卸载 cron 定时任务 =====================
setup_cron() {
    local interval="${1:-5}"  # 默认 5 分钟
    local cron_cmd="*/${interval} * * * * ${DEPLOY_DIR}/safe-deploy.sh --auto >> ${LOG_FILE} 2>&1"
    local cron_marker="# sub2api-auto-deploy"

    # 移除旧的 cron 条目
    crontab -l 2>/dev/null | grep -v "${cron_marker}" | crontab - 2>/dev/null || true

    # 添加新的 cron 条目
    (crontab -l 2>/dev/null; echo "${cron_cmd} ${cron_marker}") | crontab -

    log OK "自动更新已启用: 每 ${interval} 分钟检查一次"
    log INFO "Cron 任务: ${cron_cmd}"
    log INFO "查看日志: tail -f ${LOG_FILE}"
    log INFO "关闭自动更新: $0 --cron-off"
}

remove_cron() {
    local cron_marker="# sub2api-auto-deploy"
    crontab -l 2>/dev/null | grep -v "${cron_marker}" | crontab - 2>/dev/null || true
    log OK "自动更新已关闭"
}

# ===================== 主入口 =====================
main() {
    case "${1:-}" in
        --auto|-a)
            do_auto_update
            ;;
        --rollback|-r)
            do_rollback
            ;;
        --status|-s)
            show_status
            ;;
        --logs|-l)
            if [ -f "${LOG_FILE}" ]; then
                tail -50 "${LOG_FILE}"
            else
                echo "暂无部署日志"
            fi
            ;;
        --verify|-v)
            run_verification
            ;;
        --cron)
            setup_cron "${2:-5}"
            ;;
        --cron-off)
            remove_cron
            ;;
        --help|-h)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  (无)              正常安全部署（拉取 + 验收 + 回滚）"
            echo "  --auto            自动模式：仅在有新镜像时部署（用于 cron）"
            echo "  --cron [分钟]     启用自动更新（默认每 5 分钟检查）"
            echo "  --cron-off        关闭自动更新"
            echo "  --rollback        手动回滚到上一版本"
            echo "  --status          查看当前服务状态"
            echo "  --verify          仅运行验收检查（不部署）"
            echo "  --logs            查看部署日志"
            echo "  --help            显示帮助"
            echo ""
            echo "示例:"
            echo "  $0                # 立即部署"
            echo "  $0 --cron 3       # 每 3 分钟自动检查更新"
            echo "  $0 --cron-off     # 关闭自动更新"
            echo "  $0 --status       # 查看状态"
            echo "  $0 --rollback     # 出问题了？一键回滚"
            ;;
        *)
            do_deploy
            ;;
    esac
}

main "$@"
