$ErrorActionPreference = 'Stop'
$project = Split-Path -Parent $PSScriptRoot
$output = Join-Path $project 'build/pixel-tetris-windows-amd64'
$engine = Join-Path $output 'engine'
$archive = "$output.zip"
$compiler = Join-Path $env:WINDIR 'Microsoft.NET/Framework64/v4.0.30319/csc.exe'
$go = (Get-Command go.exe -CommandType Application).Source

if (-not (Test-Path -LiteralPath $compiler)) {
    throw '.NET Framework C# 컴파일러를 찾을 수 없습니다.'
}
New-Item -ItemType Directory -Force -Path $engine | Out-Null

$previousOS = $env:GOOS
$previousArch = $env:GOARCH
$previousCgo = $env:CGO_ENABLED
Push-Location -LiteralPath $project
try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    & $go build -trimpath -o (Join-Path $engine 'pixel-tetris.exe') .
    if ($LASTEXITCODE -ne 0) { throw '게임 엔진 빌드에 실패했습니다.' }
} finally {
    $env:GOOS = $previousOS
    $env:GOARCH = $previousArch
    $env:CGO_ENABLED = $previousCgo
    Pop-Location
}

& $compiler /nologo /target:winexe /optimize+ /codepage:65001 /reference:System.Web.Extensions.dll (
    '/out:' + (Join-Path $output 'PixelTetris.exe')
) (Join-Path $PSScriptRoot 'Window.cs')
if ($LASTEXITCODE -ne 0) { throw '윈도우 앱 빌드에 실패했습니다.' }

Compress-Archive -LiteralPath $output -DestinationPath $archive -Force
Write-Output $archive
