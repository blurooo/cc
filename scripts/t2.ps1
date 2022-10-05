#!/usr/bin/env pwsh
# Copyright 2020 the metax authors. All rights reserved. MIT license.

$Command = "t2"
$ToDir = "${Home}\.${Command}\bin"
$DownloadFileName="$Env:Tmp\${Command}.exe"

$DownloadVersion = $Env:DOWNLOAD_VERSION
$Version = if ($DownloadVersion) {
  "$Env:DOWNLOAD_VERSION"
} else {
  "latest"
}

# 设置证书
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

if (!(Test-Path $ToDir)) {
  New-Item $ToDir -ItemType Directory | Out-Null
}

# 构建URL
$DownloadUrl = "https://mirrors.tencent.com/repository/generic/cli-market/${Command}/${Version}/${Command}_windows.exe"

Write-Output "Download $DownloadUrl ...`n"

# 下载资源
Invoke-WebRequest $DownloadUrl -OutFile $DownloadFileName -UseBasicParsing

# 执行初始化逻辑，构建程序
&"$DownloadFileName" config init

# 移除缓存文件
Remove-Item "$DownloadFileName"

# 设置环境变量，预期前面的 config init 会构建 $ToDir 下的程序
$User = [EnvironmentVariableTarget]::User
$Path = [Environment]::GetEnvironmentVariable('Path', $User)
if (!(";$Path;".ToLower() -like "*;$ToDir;*".ToLower())) {
  [Environment]::SetEnvironmentVariable('Path', "$Path;$ToDir", $User)
  $Env:Path += ";$ToDir"
}

# 测试执行
metax --help
