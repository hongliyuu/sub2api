$ErrorActionPreference = 'Stop'

$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$FrontendDir = Join-Path $Root 'frontend'
$BackendDir = Join-Path $Root 'backend'

Push-Location $FrontendDir
try {
  pnpm run build
} finally {
  Pop-Location
}

$env:DATA_DIR = $BackendDir

Push-Location $BackendDir
try {
  go run -tags embed ./cmd/server
} finally {
  Pop-Location
}
