[CmdletBinding()]
param(
    [switch]$SkipFrontend,
    [switch]$SkipBuild,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$bashScript = Join-Path $scriptDir "deploy-zero-downtime.sh"

if (-not (Test-Path $bashScript)) {
    Write-Error "Cannot find deploy script: $bashScript"
    exit 1
}

function Resolve-GitBashPath {
    param(
        [string]$OverridePath
    )

    if ($OverridePath) {
        if (Test-Path $OverridePath) {
            return $OverridePath
        }
        throw "YI_CODE_GIT_BASH points to a missing file: $OverridePath"
    }

    $candidates = @(
        (Join-Path $env:ProgramFiles "Git\\bin\\bash.exe"),
        (Join-Path $env:ProgramFiles "Git\\usr\\bin\\bash.exe"),
        (Join-Path $env:ProgramW6432 "Git\\bin\\bash.exe"),
        (Join-Path $env:ProgramW6432 "Git\\usr\\bin\\bash.exe"),
        (Join-Path $env:LocalAppData "Programs\\Git\\bin\\bash.exe"),
        (Join-Path $env:LocalAppData "Programs\\Git\\usr\\bin\\bash.exe")
    ) | Where-Object { $_ -and (Test-Path $_) } | Select-Object -Unique

    if ($candidates.Count -gt 0) {
        return $candidates[0]
    }

    $bashCommand = Get-Command bash -ErrorAction SilentlyContinue
    if ($bashCommand -and (Split-Path $bashCommand.Source -Leaf) -ieq "bash.exe" -and $bashCommand.Source -match "\\Git\\") {
        return $bashCommand.Source
    }

    throw "Git Bash not found. Install Git for Windows, or set YI_CODE_GIT_BASH to bash.exe."
}

try {
    $bashExe = Resolve-GitBashPath -OverridePath $env:YI_CODE_GIT_BASH
} catch {
    Write-Error $_
    exit 1
}

$resolvedScript = (Resolve-Path $bashScript).Path -replace "\\", "/"
$bashArgs = @($resolvedScript)

if ($SkipFrontend) {
    $bashArgs += "--skip-frontend"
}
if ($SkipBuild) {
    $bashArgs += "--skip-build"
}
if ($Help) {
    $bashArgs += "--help"
}
if ($args.Count -gt 0) {
    $bashArgs += $args
}

& $bashExe @bashArgs
exit $LASTEXITCODE
