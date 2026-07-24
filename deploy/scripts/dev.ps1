[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [ValidateSet('up', 'down', 'logs', 'verify', 'reset', 'rebuild', 'preflight')]
    [string]$Action = 'up',

    [switch]$ConfirmReset
)

$ErrorActionPreference = 'Stop'
$ScriptDir = Split-Path -Parent $PSCommandPath
$RootDir = (Resolve-Path (Join-Path $ScriptDir '..\..')).Path
$EnvFile = Join-Path $RootDir '.env.dev'
$ProjectName = 'modelroute-dev'
$DevelopmentImages = [ordered]@{
    POSTGRES_IMAGE = 'm.daocloud.io/docker.io/library/postgres:17-alpine@sha256:742f40ea20b9ff2ff31db5458d127452988a2164df9e17441e191f3b72252193'
    REDIS_IMAGE = 'm.daocloud.io/docker.io/library/redis:8-alpine@sha256:9d317178eceac8454a2284a9e6df2466b93c745529947f0cd42a0fa9609d7005'
    NEW_API_IMAGE = 'm.daocloud.io/docker.io/calciumion/new-api:v1.0.0-rc.21@sha256:428018a37c0b26c163a3367c18401161707cd0e08d0f26a3dde9ff0caa05e34c'
    GO_DEV_IMAGE = 'm.daocloud.io/docker.io/library/golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191'
    GO_BUILDER_IMAGE = 'm.daocloud.io/docker.io/library/golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191'
    ALPINE_RUNTIME_IMAGE = 'm.daocloud.io/docker.io/library/alpine:3.22@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce'
    NODE_DEV_IMAGE = 'm.daocloud.io/docker.io/library/node:24-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd'
    NODE_BUILDER_IMAGE = 'm.daocloud.io/docker.io/library/node:24-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd'
    NGINX_RUNTIME_IMAGE = 'm.daocloud.io/docker.io/library/nginx:1.28-alpine@sha256:a8b39bd9cf0f83869a2162827a0caf6137ddf759d50a171451b335cecc87d236'
}

function New-RandomBytes([int]$Count) {
    $bytes = New-Object byte[] $Count
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
    return $bytes
}

function New-Hex([int]$Count) {
    return -join ((New-RandomBytes $Count) | ForEach-Object { $_.ToString('x2') })
}

function Initialize-EnvironmentFile {
    if (Test-Path $EnvFile) { return }

    $encryptionKey = [Convert]::ToBase64String((New-RandomBytes 32))
    $adminPassword = "Dv1!$(New-Hex 8)"
    $content = @"
COMPOSE_PROJECT_NAME=$ProjectName
APP_VERSION=dev
PLATFORM_IMAGE=modelroute-platform-api:dev
GATEWAY_IMAGE=modelroute-platform-gateway:dev
GATEWAY_BIND=127.0.0.1:28100
PUBLIC_ORIGIN=http://127.0.0.1:57723
SESSION_COOKIE_NAME=modelroute_dev_session
SESSION_COOKIE_SECURE=false
SESSION_TTL=24h
NEW_API_REQUEST_TIMEOUT=30s
NEW_API_ADMIN_USERNAME=mrdevadmin
NEW_API_ADMIN_PASSWORD=$adminPassword
PLATFORM_ENCRYPTION_KEY=$encryptionKey
POSTGRES_SUPERUSER=modelroute_dev_admin
POSTGRES_SUPER_PASSWORD=$(New-Hex 24)
PLATFORM_DB_NAME=modelroute_dev
PLATFORM_DB_USER=modelroute_dev
PLATFORM_DB_PASSWORD=$(New-Hex 24)
NEWAPI_DB_NAME=newapi_dev
NEWAPI_DB_USER=newapi_dev
NEWAPI_DB_PASSWORD=$(New-Hex 24)
REDIS_PASSWORD=$(New-Hex 24)
NEWAPI_SESSION_SECRET=$(New-Hex 32)
NEWAPI_CRYPTO_SECRET=$(New-Hex 32)
$($DevelopmentImages.GetEnumerator() | ForEach-Object { "$($_.Key)=$($_.Value)" } | Out-String)
"@
    [IO.File]::WriteAllText($EnvFile, $content.Trim() + [Environment]::NewLine, [Text.Encoding]::ASCII)
    Write-Host "已生成本地开发环境文件：$EnvFile"
}

function Import-EnvironmentFile {
    Get-Content $EnvFile | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]*)=(.*)$') {
            [Environment]::SetEnvironmentVariable($Matches[1].Trim(), $Matches[2], 'Process')
        }
    }
    foreach ($entry in $DevelopmentImages.GetEnumerator()) {
        if (-not [Environment]::GetEnvironmentVariable($entry.Key, 'Process')) {
            [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'Process')
        }
    }
}

function Invoke-Docker([string[]]$CommandArgs) {
    & docker @CommandArgs
    if ($LASTEXITCODE -ne 0) { throw "Docker 命令失败：docker $($CommandArgs -join ' ')" }
}

function Invoke-Python([string[]]$CommandArgs) {
    & python @CommandArgs
    if ($LASTEXITCODE -ne 0) { throw "Python 命令失败：python $($CommandArgs -join ' ')" }
}

function Dev-Compose([string[]]$CommandArgs) {
    Invoke-Docker (@('compose', '--project-name', $ProjectName, '--env-file', $EnvFile, '-f', 'compose.yaml', '-f', 'compose.dev.yaml', '--profile', 'tools') + $CommandArgs)
}

function Local-Compose([string[]]$CommandArgs) {
    Invoke-Docker (@('compose', '--project-name', $ProjectName, '--env-file', $EnvFile, '-f', 'compose.yaml', '-f', 'compose.local.yaml', '--profile', 'tools') + $CommandArgs)
}

function Get-StatusCode([string]$Url) {
    try {
        return [int](Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 10).StatusCode
    } catch {
        if ($_.Exception.Response) { return [int]$_.Exception.Response.StatusCode }
        throw
    }
}

function Wait-ForUrl([string]$Url, [int]$ExpectedStatus = 200) {
    $lastStatus = 0
    for ($attempt = 0; $attempt -lt 90; $attempt++) {
        try { $lastStatus = Get-StatusCode $Url } catch { $lastStatus = 0 }
        if ($lastStatus -eq $ExpectedStatus) { return }
        Start-Sleep -Seconds 1
    }
    throw "等待本地服务超时：$Url，最后状态 $lastStatus"
}

function Test-LocalStack([string]$BaseUrl, [string]$ExpectedVersion) {
    foreach ($path in @('/', '/api/health/live', '/api/health/ready')) {
        $status = Get-StatusCode "$BaseUrl$path"
        if ($status -ne 200) { throw "本地验收失败：$path 返回 $status" }
    }
    $modelsStatus = Get-StatusCode "$BaseUrl/v1/models"
    if ($modelsStatus -ne 401) { throw "本地验收失败：无密钥模型列表应返回 401，实际为 $modelsStatus" }
    $version = Invoke-RestMethod -Uri "$BaseUrl/api/version" -TimeoutSec 10
    if ($version.data.version -ne $ExpectedVersion) {
        throw "本地版本不一致：期望 $ExpectedVersion，实际 $($version.data.version)"
    }
    Write-Host "本地验收通过：$BaseUrl，版本 $ExpectedVersion"
}

function Start-DevelopmentStack {
    Initialize-EnvironmentFile
    Import-EnvironmentFile
    Dev-Compose @('up', '-d', '--wait', 'postgres', 'redis', 'new-api')
    Invoke-Python @('deploy/scripts/bootstrap-newapi.py', 'http://127.0.0.1:28101')
    Dev-Compose @('run', '--rm', 'model-seed')
    & docker image inspect modelroute-platform-api-dev *> $null
    $upArgs = @('up', '-d')
    if ($LASTEXITCODE -ne 0) { $upArgs += '--build' }
    Dev-Compose ($upArgs + @('platform-api', 'platform-worker', 'web-dev'))
    Wait-ForUrl 'http://127.0.0.1:57723/api/health/ready'
    Test-LocalStack 'http://127.0.0.1:57723' 'dev'
    Write-Host '本地开发环境已启动：http://127.0.0.1:57723/'
}

function Stop-DevelopmentStack {
    Initialize-EnvironmentFile
    Dev-Compose @('down', '--remove-orphans')
}

function Invoke-Preflight {
    Start-DevelopmentStack
    $version = (& git rev-parse --short=12 HEAD).Trim()
    if ($LASTEXITCODE -ne 0 -or -not $version) { throw '读取 Git 版本失败' }

    Dev-Compose @('down', '--remove-orphans')
    $oldOrigin = $env:PUBLIC_ORIGIN
    $oldVersion = $env:APP_VERSION
    $env:PUBLIC_ORIGIN = 'http://127.0.0.1:28100'
    $env:APP_VERSION = $version
    try {
        Local-Compose @('build', 'platform-api', 'gateway')
        Local-Compose @('up', '-d', '--wait', 'postgres', 'redis', 'new-api')
        Local-Compose @('run', '--rm', 'model-seed')
        Local-Compose @('up', '-d', '--wait', 'postgres', 'redis', 'new-api', 'platform-api', 'platform-worker', 'gateway')
        Test-LocalStack 'http://127.0.0.1:28100' $version
    } finally {
        try { Local-Compose @('down', '--remove-orphans') } catch { Write-Warning $_ }
        $env:PUBLIC_ORIGIN = $oldOrigin
        $env:APP_VERSION = $oldVersion
        Start-DevelopmentStack
    }
}

Push-Location $RootDir
try {
    if ($Action -ne 'down') {
        Initialize-EnvironmentFile
        Import-EnvironmentFile
    }
    switch ($Action) {
        'up' { Start-DevelopmentStack }
        'down' { Stop-DevelopmentStack }
        'logs' { Dev-Compose @('logs', '--follow', '--tail', '200', 'platform-api', 'platform-worker', 'new-api', 'web-dev') }
        'verify' { Test-LocalStack 'http://127.0.0.1:57723' 'dev' }
        'rebuild' {
            Dev-Compose @('build', 'platform-api')
            Start-DevelopmentStack
        }
        'preflight' { Invoke-Preflight }
        'reset' {
            if (-not $ConfirmReset) { throw '重置本地开发数据必须增加 -ConfirmReset' }
            Dev-Compose @('down', '--volumes', '--remove-orphans')
            Start-DevelopmentStack
        }
    }
} finally {
    Pop-Location
}
