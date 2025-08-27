function build {
    # Check if force-cgo flag is set
    $forceCGO = $false
    foreach ($arg in $args) {
        if ($arg -eq "-force-cgo" -or $arg -eq "--force-cgo") {
            $forceCGO = $true
            break
        }
    }
    
    # Disable CGO for Windows builds to avoid the modernc.org/libc issue
    # unless explicitly forced to enable it
    # See docs/windows-cgo-build-guide.md for more details about Windows builds
    if (($env:GOOS -eq "windows" -or ([string]::IsNullOrEmpty($env:GOOS) -and $env:OS -eq "Windows_NT")) -and -not $forceCGO) {
        $env:CGO_ENABLED = "0"
    } elseif ($forceCGO) {
        $env:CGO_ENABLED = "1"
    }
    
    # Ensure goversioninfo is available for Windows builds
    if ($env:GOOS -eq "windows" -or ([string]::IsNullOrEmpty($env:GOOS) -and $env:OS -eq "Windows_NT")) {
        # Check if goversioninfo is available
        $goversioninfoPath = Join-Path (go env GOPATH) "bin\goversioninfo.exe"
        if (-not (Test-Path $goversioninfoPath)) {
            Write-Host "Installing goversioninfo tool..."
            go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
        }
        
        # Ensure GOPATH\bin is in PATH
        $goPathBin = Join-Path (go env GOPATH) "bin"
        if (-not ($env:PATH -like "*$goPathBin*")) {
            $env:PATH = "$goPathBin;$env:PATH"
        }
    }
    
    go run build.go @args
}

$cmd, $rest = $args
switch ($cmd) {
    "test" {
        $env:LOGGER_DISCARD=1
        build test
    }

    "bench" {
        $env:LOGGER_DISCARD=1
        build bench
    }

    default {
        build @rest
    }
}