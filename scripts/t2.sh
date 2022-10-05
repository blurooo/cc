#!/bin/bash

# tc latest version url
ver="latest"
command="t2"
alias="metax"
to_dir="$HOME/.$command"

# shellcheck disable=SC2088
# 保留 "~" ，以使 /usr/local/bin/tc 可以被不同用户同时使用
bin_file="~/.$command/bin/$alias"
download_file_name="/tmp/$command"

# 通过环境变量指定版本
if [ -n "$DOWNLOAD_VERSION" ]; then
    ver="$DOWNLOAD_VERSION"
fi

# 展示错误
function showError() {
    echo -e "\033[31m>> 脚本非正常结束，请关注上下文提示的信息：$1\033[0m"
}

# 展示警告信息
function warn() {
    echo -e "\033[33m$*\033[0m"
}

# 展示成功信息
function success() {
    echo -e "\033[32m$*\033[0m"
}

# 啥也不干
function none() {
    export _NONE=""
}

# 抓取错误
function catch() {
    exit_code=$?
    if [ ${exit_code} -gt 0 ]; then
        if [ -n "$1" ]; then
            showError "$1"
        fi
        rm "$download_file_name"
	      exit ${exit_code}
    fi
}

# 根据操作系统获取下载链接
function getDownloadUrl() {
    if [ "$(uname)" == "Darwin" ]; then
        if [ "$(arch)" == "arm64" ]; then
            echo "https://mirrors.tencent.com/repository/generic/cli-market/$command/$ver/${command}_macos_arm64"
        else
            echo "https://mirrors.tencent.com/repository/generic/cli-market/$command/$ver/${command}_macos"
        fi
    elif [ "$(uname)" == "Linux" ]; then
        echo "https://mirrors.tencent.com/repository/generic/cli-market/$command/$ver/${command}_linux"
    else
        showError "未被支持的操作系统：$(uname)"
        exit 1
    fi
}

# 标准化配置
function configure() {
    # 初始化程序，预期会构建 $bin_file
    $download_file_name config init

    echo "下面将尝试连接命令..."
    bin_dir="/usr/local/bin"
    bin_path="$bin_dir"/"$alias"
    if echo "$PATH" | grep "$bin_dir" &>/dev/null; then
        handle_has_bin_path "$bin_path"
    else
        handle_without_bin_path "$bin_path"
    fi
}

# shellcheck disable=SC2120
function handle_has_bin_path() {
    # 解决 sudo/bash 运行函数的问题
    link=$(cat <<- EOF
if [ -f "$1" ]; then
    rm "$1"
    catch "移除文件 $1 失败，请自行将其删除后重试"
fi
if [ ! -d "$bin_dir" ]; then
    mkdir -p "$bin_dir"
    catch "创建目录 $bin_dir 失败，请自行尝试创建后重试"
fi
echo -e '#!/bin/sh\n\n$bin_file "\$@"' > "$1"
chmod +x "$1"
catch "创建 $1 失败，请自行执行下面的语句：echo -e '#!/bin/sh\n\n$bin_file "\$@"' > "$1""
success "\n💡 创建 $1 成功，接下来将可以直接使用命令 $alias 调用程序！！\n"
EOF
)
    # 当 bin 目录没有写权限，但允许提升权限时，通过 sudo 来提升
    # 1. 存在sudo程序
    # 2. 当前用户不是root
    # 3. 当前用户具备sudo权限
    if [ ! -w "$bin_dir" ] && hasSudoPermission; then
        sudo -E bash -c "$(declare -f catch); $(declare -f success); $link"
    else
        bash -c "$(declare -f catch); $(declare -f success); $link"
    fi
}


function hasSudoPermission() {
    sudo -h &>/dev/null && [ "$(whoami)" != "root" ] && sudo -u "$(whoami)" echo "$command yes" &>/dev/null;
}

function handle_without_bin_path() {
    # 已经存在指向
    if [ "$(which "$alias")" == "$1" ] &>/dev/null; then
        return
    else
        warn "\033[33m!\033[0m 请将 $bin_file 软链到 \$PATH 中（执行 ln -s $bin_file $1），或直接将 $to_dir 添加到 \$PATH"
    fi
}

function test() {
    echo "--------------------------------------------"
    $alias -h
}

# 开始执行流程
function start() {
    # 确保目录存在
    [ -d "$to_dir" ] || mkdir -p "$to_dir"

    download_url=$(getDownloadUrl)

    echo "正在下载资源 $download_url ..."
    curl -o "$download_file_name" -# -k "$download_url"

    catch "下载资源失败，请确认网络没问题再重试"

    # 赋予执行权限
    chmod +x "$download_file_name"

    echo "正在执行安装..."

    configure

    catch "配置失败，请重试或联系 TCC-Helper"

    test

    catch "安装失败，请重试或联系 TCC-Helper"

    [ ! -f "$download_file_name" ] || rm $download_file_name
}

start
