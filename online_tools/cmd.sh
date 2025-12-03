#!/bin/bash
# online_tools 快速启停脚本
# 配置区（根据实际情况修改以下3个参数）
APP_NAME="online_tools"          # 服务名称（自定义）
APP_CMD="/mooyliu/projects/online_tools/online_tools"  # online_tools 启动命令（必填！替换为实际路径）
LOG_FILE="/var/log/online_tools.log"  # 日志文件路径
PID_FILE="/var/run/online_tools.pid"  # PID 文件路径（记录进程ID）

# 颜色输出函数（可选，仅为提示更友好）
red() { echo -e "\033[31m$1\033[0m"; }
green() { echo -e "\033[32m$1\033[0m"; }
yellow() { echo -e "\033[33m$1\033[0m"; }

# 检查是否为root（可选，若服务需要权限则保留）
check_root() {
    if [ $EUID -ne 0 ]; then
        red "错误：请以root/管理员身份运行此脚本（sudo ./online_tools.sh）"
        exit 1
    fi
}

# 启动服务
start() {
    # 检查启动命令是否存在
    if [ ! -x "$APP_CMD" ]; then
        red "错误：启动命令 $APP_CMD 不存在或无执行权限！"
        exit 1
    fi

    # 检查是否已运行
    if [ -f "$PID_FILE" ]; then
        OLD_PID=$(cat "$PID_FILE")
        if ps -p "$OLD_PID" > /dev/null 2>&1; then
            yellow "提示：$APP_NAME 已在运行（PID: $OLD_PID），无需重复启动"
            exit 0
        else
            rm -f "$PID_FILE"  # PID文件存在但进程已死，清理无效PID
        fi
    fi

    # 创建日志/ PID目录（防止目录不存在）
    mkdir -p $(dirname "$LOG_FILE")
    mkdir -p $(dirname "$PID_FILE")

    green "正在后台启动 $APP_NAME ..."
    # 后台启动服务，输出日志到指定文件，记录PID
    nohup $APP_CMD > $LOG_FILE 2>&1 &
    NEW_PID=$!
    echo $NEW_PID > $PID_FILE

    # 验证启动是否成功
    sleep 2  # 等待2秒让服务初始化
    if ps -p "$NEW_PID" > /dev/null 2>&1; then
        green "成功：$APP_NAME 启动完成！"
        green "PID: $NEW_PID | 日志：$LOG_FILE"
    else
        red "错误：$APP_NAME 启动失败！请查看日志：tail -f $LOG_FILE"
        rm -f "$PID_FILE"  # 清理无效PID文件
        exit 1
    fi
}

# 停止服务
stop() {
    # 检查是否运行
    if [ ! -f "$PID_FILE" ]; then
        yellow "提示：$APP_NAME 未运行（PID文件不存在）"
        exit 0
    fi

    PID=$(cat "$PID_FILE")
    if ! ps -p "$PID" > /dev/null 2>&1; then
        red "警告：PID文件存在，但进程 $PID 已不存在！"
        rm -f "$PID_FILE"
        exit 0
    fi

    # 杀死进程
    green "正在停止 $APP_NAME（PID: $PID）..."
    kill -9 "$PID" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        rm -f "$PID_FILE"
        green "成功：$APP_NAME 已停止！"
    else
        red "错误：停止 $APP_NAME 失败，请手动执行 kill -9 $PID"
        exit 1
    fi
}

# 查看状态
status() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            green "$APP_NAME 正在运行（PID: $PID）"
        else
            yellow "$APP_NAME 已停止（PID文件无效）"
        fi
    else
        yellow "$APP_NAME 未运行"
    fi
}

# 帮助信息
usage() {
    echo "使用方法：$0 [start|stop|restart|status]"
    echo "  start   - 后台启动 $APP_NAME"
    echo "  stop    - 强制关闭 $APP_NAME"
    echo "  restart - 重启 $APP_NAME"
    echo "  status  - 查看 $APP_NAME 运行状态"
    exit 1
}

# 主逻辑
case "$1" in
    start)
        # check_root  # 若不需要root权限，注释此行
        start
        ;;
    stop)
        # check_root  # 若不需要root权限，注释此行
        stop
        ;;
    restart)
        # check_root  # 若不需要root权限，注释此行
        stop
        sleep 1
        start
        ;;
    status)
        status
        ;;
    *)
        usage
        ;;
esac

exit 0