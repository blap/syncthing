function build {
    # Disable CGO for Windows builds to avoid the modernc.org/libc issue
    $env:CGO_ENABLED = "1"
    
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