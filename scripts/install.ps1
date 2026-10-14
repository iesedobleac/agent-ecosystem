# agent-ecosystem installer for Windows.
#
#   irm https://raw.githubusercontent.com/iesedobleac/agent-ecosystem/main/scripts/install.ps1 | iex
#
# Windows installs from source with `go install` until the binaries can be signed.
$ErrorActionPreference = "Stop"

$Module = "github.com/iesedobleac/agent-ecosystem/cmd/agent-ecosystem"

function Write-Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }
function Write-Ok($msg) { Write-Host "[ok]    $msg" -ForegroundColor Green }
function Write-Warn($msg) { Write-Host "[warn]  $msg" -ForegroundColor Yellow }

Write-Host "agent-ecosystem - agents, skills and MCP servers for your AI assistants" -ForegroundColor Cyan

Write-Step "Checking prerequisites"
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Host "[error] git is required. Install it with: winget install Git.Git" -ForegroundColor Red
    exit 1
}
Write-Ok "git is available"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "[error] Go is required on Windows (installs build from source). Install it, open a new terminal and run this again:" -ForegroundColor Red
    Write-Host "          winget install GoLang.Go"
    Write-Host "        Then install manually with:"
    Write-Host "          go install $Module@latest"
    exit 1
}
Write-Ok ("Go " + ((go env GOVERSION) -replace "^go", ""))

if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Warn "Node.js not found: CodeGraph needs it (winget install OpenJS.NodeJS.LTS)"
}

Write-Step "Installing with go install"
go install "$Module@latest"
if ($LASTEXITCODE -ne 0) {
    Write-Host "[error] go install failed" -ForegroundColor Red
    exit 1
}

$gobin = go env GOBIN
if (-not $gobin) { $gobin = Join-Path (go env GOPATH) "bin" }
if (($env:Path -split ";") -notcontains $gobin) {
    Write-Warn "$gobin is not in your PATH. Add it with:"
    Write-Host "          [Environment]::SetEnvironmentVariable('Path', `$env:Path + ';$gobin', 'User')"
}
Write-Ok "Installed agent-ecosystem"

Write-Host "`nInstallation complete." -ForegroundColor Green
Write-Host "Run agent-ecosystem to open the interactive installer, or agent-ecosystem install --help."
