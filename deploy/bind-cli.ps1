<#
.SYNOPSIS
  LoopForge 绑定本机 CLI（bind-cli for Windows）。

.DESCRIPTION
  在 Windows 执行机上安装 agentflow-runner 与 Gate MCP helper，并把机器接入
  平台。CLI 登录凭证始终保留在本机；runner 仅向平台发起出站 HTTPS。

  用法（绑定码从平台「个人 → 执行器」页面复制的命令中自动携带）：
    iwr -UseBasicParsing https://<平台地址>/runners/install.ps1 -OutFile $env:TEMP\loopforge-bind-cli.ps1
    powershell -ExecutionPolicy Bypass -File $env:TEMP\loopforge-bind-cli.ps1 `
      -Server https://workflow.example.com -Token <绑定码>

  可选参数：
    -Name <名称>          机器显示名（默认主机名）
    -Concurrency <N>      最大并发任务数（默认 1）
    -BinDir <目录>        二进制安装目录（默认 %USERPROFILE%\.agentflow-runner\bin）
    -From <文件>          使用本地已编译的 agentflow-runner.exe
    -DownloadUrl <URL>    从指定 URL 下载 runner（含 {os} {arch} 占位符）
    -Sha256 <hex>         校验下载的 runner
    -Service              安装计划任务（登录时启动）并立即运行
    -NoStart              只安装配置，不启动
    -NoAutoUpdate         禁止从平台自动更新
    -Uninstall            卸载：注销机器、停止计划任务、删除文件

  限制：Windows 支持 local 沙箱执行；native（seatbelt/bwrap）隔离不可用，
  Docker 沙箱不在本版支持范围内。
#>
[CmdletBinding()]
param(
  [string]$Server = "",
  [string]$Token = "",
  [string]$Name = "",
  [int]$Concurrency = 1,
  [string]$BinDir = "",
  [string]$From = "",
  [string]$DownloadUrl = "",
  [string]$Sha256 = "",
  [switch]$Service,
  [switch]$NoStart,
  [switch]$NoAutoUpdate,
  [switch]$Uninstall
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$RunnerHome = Join-Path $env:USERPROFILE ".agentflow-runner"
if ([string]::IsNullOrWhiteSpace($BinDir)) { $BinDir = Join-Path $RunnerHome "bin" }
$StateFile = Join-Path $RunnerHome "state.json"
$ConfigFile = Join-Path $RunnerHome "config.json"
$RunnerExe = Join-Path $BinDir "agentflow-runner.exe"
$GateExe = Join-Path $BinDir "agentflow-gate-mcp.exe"
$Wrapper = Join-Path $RunnerHome "run-runner.ps1"
$TaskName = "LoopForgeRunner"
$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

function Write-Info([string]$Message) { Write-Host $Message }

function Fail([string]$Message) { Write-Error $Message; exit 1 }

function Get-DownloadedFile([string]$Url, [string]$Destination) {
  Invoke-WebRequest -UseBasicParsing -Uri $Url -OutFile $Destination
}

function Get-RemoteSha256([string]$Url) {
  $separator = if ($Url.Contains("?")) { "&" } else { "?" }
  $metadata = Invoke-RestMethod -UseBasicParsing -Uri "$Url${separator}metadata=1"
  if (-not $metadata.sha256) { return "" }
  return [string]$metadata.sha256
}

function Assert-Sha256([string]$Path, [string]$Expected) {
  if ([string]::IsNullOrWhiteSpace($Expected)) { return }
  $actual = (Get-FileHash -Algorithm SHA256 -Path $Path).Hash.ToLowerInvariant()
  if ($actual -ne $Expected.ToLowerInvariant()) {
    Remove-Item -Force $Path -ErrorAction SilentlyContinue
    Fail "文件校验失败：期望 $Expected，实际 $actual"
  }
}

function Install-Runner {
  New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
  if (-not [string]::IsNullOrWhiteSpace($From)) {
    if (-not (Test-Path $From)) { Fail "未找到本地二进制：$From" }
    Copy-Item -Force $From $RunnerExe
    Assert-Sha256 $RunnerExe $Sha256
    return
  }
  if ([string]::IsNullOrWhiteSpace($Server)) { Fail "缺少 -Server 或 -DownloadUrl" }
  $url = if ([string]::IsNullOrWhiteSpace($DownloadUrl)) {
    "$Server/runners/agentflow-runner?os=windows&arch=$Arch"
  } else {
    $DownloadUrl.Replace("{os}", "windows").Replace("{arch}", $Arch)
  }
  Write-Info "下载 runner（windows/$Arch）..."
  Get-DownloadedFile $url $RunnerExe
  $expected = if ([string]::IsNullOrWhiteSpace($Sha256)) { Get-RemoteSha256 $url } else { $Sha256 }
  Assert-Sha256 $RunnerExe $expected
}

function Install-Gate {
  $url = "$Server/runners/agentflow-gate-mcp?os=windows&arch=$Arch"
  Write-Info "下载 Gate MCP（windows/$Arch）..."
  Get-DownloadedFile $url $GateExe
  Assert-Sha256 $GateExe (Get-RemoteSha256 $url)
}

function Write-Wrapper {
  $runnerArgs = "-server `"$Server`" -token `"$Token`" -name `"$($Name)`" -concurrency $Concurrency -state `"$StateFile`" -workspace `"$(Join-Path $RunnerHome 'workspace')`""
  if ($NoAutoUpdate) { $runnerArgs += " -auto-update=false" }
  $content = @"
# 由 bind-cli.ps1 生成的稳定启动包装。计划任务调用本文件，因此自动更新可以
# 在 runner 退出后替换运行中的 agentflow-runner.exe（Windows 不允许覆盖
# 正在运行的映像）。
`$ErrorActionPreference = "Stop"
`$exe = "$RunnerExe"
`$pending = "`$exe.pending"
if (Test-Path `$pending) {
  Move-Item -Force `$pending `$exe
}
`$env:AGENTFLOW_GATE_MCP_BINARY = "$GateExe"
while (`$true) {
  & `$exe $runnerArgs
  # 上一次运行可能刚退出；给 Windows 一点时间释放映像锁再应用待更新版本。
  for (`$i = 0; `$i -lt 30; `$i++) {
    try {
      if (Test-Path `$pending) { Move-Item -Force `$pending `$exe }
      break
    } catch {
      Start-Sleep -Milliseconds 500
    }
  }
  Start-Sleep -Seconds 5
}
"@
  Set-Content -Path $Wrapper -Value $content -Encoding UTF8
}

function Install-Service {
  Write-Wrapper
  $action = "powershell.exe -NoProfile -ExecutionPolicy Bypass -File `"$Wrapper`""
  schtasks /Create /TN $TaskName /TR $action /SC ONLOGON /RL LIMITED /F | Out-Null
  if ($LASTEXITCODE -ne 0) { Fail "计划任务创建失败（schtasks 退出码 $LASTEXITCODE）" }
  Write-Info "计划任务 $TaskName 已创建（登录时启动）"
}

function Write-Config {
  @{ serverUrl = $Server; name = $Name; concurrency = $Concurrency } |
    ConvertTo-Json | Set-Content -Path $ConfigFile -Encoding UTF8
}

function Stop-Runner {
  Get-Process -Name "agentflow-runner" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
}

function Uninstall-Runner {
  schtasks /Delete /TN $TaskName /F 2>$null | Out-Null
  $serverUrl = $Server
  if ([string]::IsNullOrWhiteSpace($serverUrl) -and (Test-Path $ConfigFile)) {
    try { $serverUrl = [string](Get-Content -Raw -Path $ConfigFile | ConvertFrom-Json).serverUrl } catch { }
  }
  if (Test-Path $StateFile) {
    try {
      $state = Get-Content -Raw -Path $StateFile | ConvertFrom-Json
      if (-not [string]::IsNullOrWhiteSpace($serverUrl) -and $state.sessionToken) {
        Invoke-RestMethod -UseBasicParsing -Method Post -Uri "$serverUrl/runners/deregister" `
          -Headers @{ Authorization = "Bearer $($state.sessionToken)" } -Body "{}" -ContentType "application/json" | Out-Null
      }
    } catch {
      Write-Info "注销机器失败（可稍后在平台移除）：$($_.Exception.Message)"
    }
  }
  Stop-Runner
  Remove-Item -Recurse -Force $RunnerHome -ErrorAction SilentlyContinue
  Write-Info "已卸载 LoopForge 执行器"
}

if ($Uninstall) {
  Uninstall-Runner
  exit 0
}

if ([string]::IsNullOrWhiteSpace($Server)) { Fail "-Server 必填" }
if ([string]::IsNullOrWhiteSpace($Token)) { Fail "-Token 必填（平台执行器页面的一次性绑定码）" }
if ([string]::IsNullOrWhiteSpace($Name)) { $Name = $env:COMPUTERNAME }

Stop-Runner
Install-Runner
Install-Gate
Write-Wrapper
Write-Config
Write-Info "配置已写入 $RunnerHome（state: $StateFile，workspace: $(Join-Path $RunnerHome 'workspace')）"

if ($NoStart) {
  Write-Info "已完成安装与配置（-NoStart）。启动：powershell -File `"$Wrapper`""
  exit 0
}
if ($Service) {
  Install-Service
  Start-Process -FilePath "powershell.exe" -ArgumentList "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "`"$Wrapper`"" -WindowStyle Hidden
  Write-Info "runner 已在后台启动；卸载：powershell -File bind-cli.ps1 -Uninstall"
  exit 0
}

& $RunnerExe -server $Server -token $Token -name $Name -concurrency $Concurrency -state $StateFile -workspace (Join-Path $RunnerHome "workspace")