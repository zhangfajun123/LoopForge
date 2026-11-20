param(
  [string]$Server,
  [string]$Token,
  [string]$Name = $env:COMPUTERNAME,
  [string]$Workspace = (Join-Path $env:USERPROFILE 'LoopForge\workspace'),
  [switch]$Uninstall
)

$ErrorActionPreference = 'Stop'
$installRoot = Join-Path $env:LOCALAPPDATA 'LoopForge\Runner'
$binary = Join-Path $installRoot 'agentflow-runner.exe'

if ($Uninstall) {
  if (Test-Path $binary) {
    $running = Get-Process -Name 'agentflow-runner' -ErrorAction SilentlyContinue |
      Where-Object { $_.Path -and $_.Path.Equals($binary, [StringComparison]::OrdinalIgnoreCase) }
    if ($running) {
      throw '请先在 Runner 窗口按 Ctrl+C 停止执行器，再运行卸载命令。'
    }
  }
  Remove-Item -LiteralPath $installRoot -Recurse -Force -ErrorAction SilentlyContinue
  Write-Output '本机 Runner 文件和绑定状态已删除；请在 LoopForge 执行机器页面删除对应机器。'
  return
}

if ([string]::IsNullOrWhiteSpace($Server) -or [string]::IsNullOrWhiteSpace($Token)) {
  throw '需要从 LoopForge 执行机器页面复制包含 Server 和 Token 的接入命令。'
}
New-Item -ItemType Directory -Path $installRoot -Force | Out-Null

$Server = $Server.TrimEnd('/')
$downloadUrl = "$Server/runners/agentflow-runner?os=windows&arch=amd64"
$metadata = Invoke-RestMethod -UseBasicParsing -Uri ($downloadUrl + '&metadata=1')
if ($metadata.sha256 -notmatch '^[0-9a-fA-F]{64}$') {
  throw '服务端没有返回有效的 Runner 校验值。'
}
$temporaryBinary = $binary + '.download'
try {
  Invoke-WebRequest -UseBasicParsing -Uri $downloadUrl -OutFile $temporaryBinary
  $actualHash = (Get-FileHash -LiteralPath $temporaryBinary -Algorithm SHA256).Hash
  if (-not $actualHash.Equals($metadata.sha256, [StringComparison]::OrdinalIgnoreCase)) {
    throw 'Runner SHA256 校验失败，已中止安装。'
  }
  Move-Item -LiteralPath $temporaryBinary -Destination $binary -Force
} finally {
  Remove-Item -LiteralPath $temporaryBinary -Force -ErrorAction SilentlyContinue
}

$configPath = Join-Path $installRoot 'runner.json'
$config = [ordered]@{
  server = $Server
  name = $Name
  workspace = $Workspace
  state = (Join-Path $installRoot 'state.json')
}
$utf8NoBom = New-Object System.Text.UTF8Encoding -ArgumentList $false
[System.IO.File]::WriteAllText($configPath, ($config | ConvertTo-Json), $utf8NoBom)
$launcherPath = Join-Path $installRoot 'Start-LoopForgeRunner.ps1'
$launcherCmdPath = Join-Path $installRoot 'Start-LoopForgeRunner.cmd'
$launcher = @'
$paramText = $args
$ErrorActionPreference = 'Stop'
$installRoot = Join-Path $env:LOCALAPPDATA 'LoopForge\Runner'
$config = Get-Content -LiteralPath (Join-Path $installRoot 'runner.json') -Raw | ConvertFrom-Json
$token = ''
if ($paramText.Count -gt 0) { $token = [string]$paramText[0] }
$binary = Join-Path $installRoot 'agentflow-runner.exe'
& $binary -server $config.server -token $token -name $config.name -state $config.state -workspace $config.workspace -auto-update=false
'@
Set-Content -LiteralPath $launcherPath -Value $launcher -Encoding UTF8
@(
  '@echo off'
  'powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0Start-LoopForgeRunner.ps1"'
) | Set-Content -LiteralPath $launcherCmdPath -Encoding ASCII

Write-Output "Runner 已安装到 $installRoot。现在以前台方式启动，保持此 PowerShell 窗口开启；之后可运行 $launcherCmdPath 重启。"
Write-Output 'Windows 当前使用 local（本机直接执行）模式；没有系统级任务隔离，请只接入可信仓库和工作流。'
& ([scriptblock]::Create((Get-Content -LiteralPath $launcherPath -Raw))) $Token
