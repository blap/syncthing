function build {
    # Check if Go is installed
    try {
        $goVersion = & go version 2>$null
        if ($null -eq $goVersion) {
            throw "Go not found"
        }
        Write-Host "Found Go: $goVersion"
    } catch {
        Write-Host "Error: Go is not installed or not in PATH."
        Write-Host "Please install Go 1.24 or later from https://golang.org/dl/"
        Write-Host "Make sure to add Go to your PATH environment variable."
        return
    }
    
    # Always enable CGO for Windows builds to avoid the modernc.org/libc issue
    # See docs/windows-cgo-build-guide.md for more details about Windows builds
    
    # Ensure GOOS is set correctly for Windows builds
    if ([string]::IsNullOrEmpty($env:GOOS) -and $env:OS -eq "Windows_NT") {
        $env:GOOS = "windows"
    }
    
    # Ensure GOARCH is set correctly for 64-bit builds
    if ([string]::IsNullOrEmpty($env:GOARCH)) {
        $env:GOARCH = "amd64"
    }
    
    # Always enable CGO for Windows builds
    if ($env:GOOS -eq "windows" -or ([string]::IsNullOrEmpty($env:GOOS) -and $env:OS -eq "Windows_NT")) {
        $env:CGO_ENABLED = "1"
        # Explicitly set CC to the 64-bit MinGW GCC to avoid architecture issues
        $env:CC = "x86_64-w64-mingw32-gcc"
        Write-Host "CGO enabled for Windows build with 64-bit MinGW GCC compiler"
        Write-Host "Target architecture: $env:GOARCH"
    }
    
    # Set version if not already set
    if ([string]::IsNullOrEmpty($env:VERSION)) {
        $env:VERSION = "v2.0.4"
    }
    
    Write-Host "Building Syncthing with CGO enabled using build.go..."
    Write-Host "Version: $env:VERSION"
    Write-Host "GOOS: $env:GOOS, GOARCH: $env:GOARCH, CGO_ENABLED: $env:CGO_ENABLED"
    
    # Use build.go script with forcecgo tag - this approach works reliably
    $envVars = @{
        "CGO_ENABLED" = "1"
        "CC" = "x86_64-w64-mingw32-gcc"
        "GOOS" = $env:GOOS
        "GOARCH" = $env:GOARCH
        "VERSION" = $env:VERSION
    }
    
    # Create a new process with explicit environment variables
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = "go"
    $psi.Arguments = "run build.go -goos $env:GOOS -goarch $env:GOARCH -tags forcecgo build syncthing"
    $psi.UseShellExecute = $false
    $psi.RedirectStandardOutput = $true
    $psi.RedirectStandardError = $true
    $psi.WorkingDirectory = Get-Location
    
    # Set environment variables explicitly
    foreach ($key in $envVars.Keys) {
        $psi.EnvironmentVariables[$key] = $envVars[$key]
    }
    
    try {
        $process = [System.Diagnostics.Process]::Start($psi)
        $output = $process.StandardOutput.ReadToEnd()
        $errorOutput = $process.StandardError.ReadToEnd()
        $process.WaitForExit()
        
        Write-Host $output
        if (-not [string]::IsNullOrEmpty($errorOutput)) {
            Write-Host "Error output: $errorOutput"
        }
        
        if ($process.ExitCode -ne 0) {
            Write-Host "Build failed with exit code: $($process.ExitCode)"
            return
        }
        
        # Check if the build was successful
        if (Test-Path "syncthing.exe") {
            Write-Host "Successfully built syncthing.exe"
            $fileInfo = Get-Item "syncthing.exe"
            Write-Host "File size: $($fileInfo.Length) bytes"
            Write-Host "Created: $($fileInfo.CreationTime)"
            
            # Test the executable
            try {
                $versionOutput = & .\syncthing.exe --version 2>&1
                Write-Host "Version info: $versionOutput"
            } catch {
                Write-Host "Warning: Could not get version info from executable"
            }
        } else {
            Write-Host "Failed to create syncthing.exe"
        }
        
        # Clean up generated files
        if (Test-Path "versioninfo.json") {
            Remove-Item "versioninfo.json" -Force
        }
        if (Test-Path "cmd\syncthing\resource.syso") {
            Remove-Item "cmd\syncthing\resource.syso" -Force
        }
    } catch {
        Write-Host "Error running build command: $_"
        Write-Host "Please ensure you have a C compiler (like MinGW-w64) installed for CGO support."
    }
}

# Always build Syncthing with CGO enabled
build