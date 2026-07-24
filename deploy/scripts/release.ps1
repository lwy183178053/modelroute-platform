[CmdletBinding()]
param(
    [switch]$Deploy103
)

$ErrorActionPreference = 'Stop'
$ScriptDir = Split-Path -Parent $PSCommandPath
$RootDir = (Resolve-Path (Join-Path $ScriptDir '..\..')).Path
$ApiRepository = 'ghcr.io/lwy183178053/modelroute-platform-api'
$GatewayRepository = 'ghcr.io/lwy183178053/modelroute-platform-gateway'
$SshHost = '38.34.191.103'
$SshPort = 53612
$SshKey = Join-Path $HOME '.ssh\modelroute_103_ed25519'
$RemoteRoot = 'root@38.34.191.103'
$TempRoot = ''

function Invoke-Native([string]$Command, [string[]]$CommandArgs) {
    & $Command @CommandArgs
    if ($LASTEXITCODE -ne 0) { throw "命令失败：$Command $($CommandArgs -join ' ')" }
}

function Invoke-Captured([string]$Command, [string[]]$CommandArgs) {
    $output = & $Command @CommandArgs
    if ($LASTEXITCODE -ne 0) { throw "命令失败：$Command $($CommandArgs -join ' ')" }
    return ($output -join "`n").Trim()
}

function Assert-CleanPushedCommit {
    $status = Invoke-Captured 'git' @('status', '--porcelain')
    if ($status) { throw '发布要求工作区无未提交改动' }

    $script:FullVersion = Invoke-Captured 'git' @('rev-parse', 'HEAD')
    $script:Version = Invoke-Captured 'git' @('rev-parse', '--short=12', 'HEAD')
    $script:Branch = Invoke-Captured 'git' @('branch', '--show-current')
    if (-not $Branch) { throw '发布要求当前位于命名分支' }
    Invoke-Native 'git' @('fetch', 'origin', $Branch)
    $remoteVersion = Invoke-Captured 'git' @('rev-parse', "origin/$Branch")
    if ($remoteVersion -ne $FullVersion) { throw '当前提交尚未完整推送到 origin' }
}

function Test-DockerRegistryLogin {
    $configPath = Join-Path $HOME '.docker\config.json'
    if (-not (Test-Path $configPath)) { return $false }
    try {
        $config = Get-Content $configPath -Raw | ConvertFrom-Json
        return $config.auths.PSObject.Properties.Name -contains 'ghcr.io'
    } catch {
        return $false
    }
}

function Assert-DeploymentPrerequisites {
    if (-not (Test-Path $SshKey)) { throw "缺少 103 SSH 私钥：$SshKey" }
    if (-not (Test-DockerRegistryLogin)) { throw '本机尚未登录 GHCR，请先执行 docker login ghcr.io' }
    Invoke-Native 'ssh' @('-i', $SshKey, '-p', "$SshPort", '-o', 'BatchMode=yes', '-o', 'StrictHostKeyChecking=no', $RemoteRoot, 'test -f /root/.docker/config.json && grep -q ghcr.io /root/.docker/config.json')
}

function Invoke-ReleaseTests {
    Push-Location (Join-Path $RootDir 'apps\api')
    try {
        Invoke-Native 'go' @('test', './...', '-count=1')
        Invoke-Native 'go' @('vet', './...')
        Invoke-Native 'go' @('build', './...')
    } finally { Pop-Location }

    Push-Location (Join-Path $RootDir 'apps\web')
    try {
        Invoke-Native 'npm.cmd' @('test')
        Invoke-Native 'npm.cmd' @('run', 'test:ui')
        Invoke-Native 'npm.cmd' @('run', 'build')
    } finally { Pop-Location }

    Push-Location $RootDir
    try {
        Invoke-Native 'python' @('-m', 'unittest', 'discover', '-s', 'deploy/tests', '-p', 'test_*.py', '-v')
        Invoke-Native 'docker' @('compose', '--env-file', '.env.example', '--profile', 'tools', 'config', '--quiet')
        Invoke-Native 'powershell.exe' @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', 'deploy/scripts/dev.ps1', 'preflight')
    } finally { Pop-Location }
}

function Get-ImageDigest([string]$Image) {
    $inspect = Invoke-Captured 'docker' @('buildx', 'imagetools', 'inspect', $Image)
    $match = [regex]::Match($inspect, '(?m)^Digest:\s+(sha256:[0-9a-f]{64})$')
    if (-not $match.Success) { throw "读取镜像摘要失败：$Image" }
    return $match.Groups[1].Value
}

function Build-PushImages {
    $script:ApiTag = "$ApiRepository`:$Version"
    $script:GatewayTag = "$GatewayRepository`:$Version"
    Invoke-Native 'docker' @(
        'buildx', 'build', '--platform', 'linux/amd64', '--push',
        '--build-arg', "APP_VERSION=$Version",
        '--build-arg', 'GO_BUILDER_IMAGE=m.daocloud.io/docker.io/library/golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191',
        '--build-arg', 'ALPINE_RUNTIME_IMAGE=m.daocloud.io/docker.io/library/alpine:3.22@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce',
        '-f', 'apps/api/Dockerfile', '-t', $ApiTag, '.'
    )
    Invoke-Native 'docker' @(
        'buildx', 'build', '--platform', 'linux/amd64', '--push',
        '--build-arg', 'NODE_BUILDER_IMAGE=m.daocloud.io/docker.io/library/node:24-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd',
        '--build-arg', 'NGINX_RUNTIME_IMAGE=m.daocloud.io/docker.io/library/nginx:1.28-alpine@sha256:a8b39bd9cf0f83869a2162827a0caf6137ddf759d50a171451b335cecc87d236',
        '-f', 'deploy/gateway/Dockerfile', '-t', $GatewayTag, '.'
    )
    $script:ApiDigest = Get-ImageDigest $ApiTag
    $script:GatewayDigest = Get-ImageDigest $GatewayTag
}

function New-ReleaseArtifacts {
    $tempBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    $script:TempRoot = Join-Path $tempBase "modelroute-release-$Version-$([Guid]::NewGuid().ToString('N'))"
    New-Item -ItemType Directory -Path $TempRoot | Out-Null
    $script:Archive = Join-Path $TempRoot "modelroute-release-$Version.tar"
    $script:Manifest = Join-Path $TempRoot "modelroute-release-$Version.env"
    $script:Installer = Join-Path $RootDir 'deploy\scripts\install-release.sh'
    Invoke-Native 'git' @('archive', '--format=tar', "--output=$Archive", 'HEAD')
    $manifestContent = @(
        "RELEASE_VERSION=$Version"
        "PLATFORM_IMAGE=$ApiRepository@$ApiDigest"
        "GATEWAY_IMAGE=$GatewayRepository@$GatewayDigest"
    ) -join "`n"
    [IO.File]::WriteAllText($Manifest, $manifestContent + "`n", [Text.Encoding]::ASCII)
    $script:ArchiveSha = (Get-FileHash -Algorithm SHA256 $Archive).Hash.ToLowerInvariant()
}

function Invoke-RemoteRelease {
    $remoteArchive = "/tmp/modelroute-release-$Version.tar"
    $remoteManifest = "/tmp/modelroute-release-$Version.env"
    $remoteInstaller = "/tmp/modelroute-install-release-$Version.sh"
    $scpCommon = @('-i', $SshKey, '-P', "$SshPort", '-o', 'StrictHostKeyChecking=no')
    Invoke-Native 'scp' ($scpCommon + @($Archive, "${RemoteRoot}:$remoteArchive"))
    Invoke-Native 'scp' ($scpCommon + @($Manifest, "${RemoteRoot}:$remoteManifest"))
    Invoke-Native 'scp' ($scpCommon + @($Installer, "${RemoteRoot}:$remoteInstaller"))
    Invoke-Native 'ssh' @('-i', $SshKey, '-p', "$SshPort", '-o', 'BatchMode=yes', '-o', 'StrictHostKeyChecking=no', $RemoteRoot, "sh $remoteInstaller $Version $remoteArchive $remoteManifest $ArchiveSha")
}

function Remove-ReleaseTemp {
    if (-not $TempRoot -or -not (Test-Path $TempRoot)) { return }
    $tempBase = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    $resolved = [IO.Path]::GetFullPath($TempRoot)
    if (-not $resolved.StartsWith($tempBase, [StringComparison]::OrdinalIgnoreCase)) {
        throw "拒绝清理非临时目录：$resolved"
    }
    Remove-Item -LiteralPath $resolved -Recurse -Force
}

Push-Location $RootDir
try {
    Assert-CleanPushedCommit
    if ($Deploy103) { Assert-DeploymentPrerequisites }
    Invoke-ReleaseTests
    if (-not $Deploy103) {
        Write-Host "本地发布验收通过：$Version；未指定 -Deploy103，未推送镜像或操作服务器。"
        exit 0
    }
    Build-PushImages
    New-ReleaseArtifacts
    Invoke-RemoteRelease
    Write-Host "ModelRoute 已发布到 103：$Version"
} finally {
    try { Remove-ReleaseTemp } finally { Pop-Location }
}
