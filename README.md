# R2 Uploader

A command-line tool for uploading files to Cloudflare R2 storage with profile management and presigned URL generation.

## Features

- Upload single files or entire directories to Cloudflare R2
- Maintain directory structure when uploading
- Save multiple profiles for different R2 buckets/accounts
- Generate presigned URLs for uploaded files
- Progress tracking for large file uploads
- Multi-part upload support for large files

## Installation

### Download Pre-built Binary

Visit the [Releases](https://github.com/actuallydan/r2uploader/releases) page to download the latest version for your platform:

- Windows (AMD64): `r2uploader-windows-amd64.exe`
- Linux (AMD64): `r2uploader-linux-amd64`
- macOS (AMD64): `r2uploader-darwin-amd64`
- macOS (ARM64/M1): `r2uploader-darwin-arm64`

### Build from Source

Requirements:
- Go 1.16 or later

## Clone the repository
```
git clone https://github.com/actuallydan/r2uploader.git
cd r2uploader
```

## Build the binary
```
go build -o r2uploader
```

### Optional: Install to your Go bin directory
```
go install
```

## Usage
1. Run 
```
./r2uploader
```
2. On first run, you'll be prompted to either create a new profile or proceed without saving credentials.
3. For profiles, you'll need:
   - Cloudflare API Token
   - Cloudflare Account ID
   - R2 Access Key
   - R2 Secret Key
   - R2 Bucket Name

4. Enter the path to a file or directory to upload:
   - Drag and drop is supported
   - Type 'q' to quit
5. Confirm the upload and wait for completion
6. Copy the generated presigned URLs to access your files

## Building for Multiple Platforms
To build for multiple platforms manually:
```
# Windows (64-bit)
GOOS=windows GOARCH=amd64 go build -o r2uploader-windows-amd64.exe
# Linux (64-bit)
GOOS=linux GOARCH=amd64 go build -o r2uploader-linux-amd64
# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o r2uploader-darwin-amd64
# macOS (M1/ARM)
GOOS=darwin GOARCH=arm64 go build -o r2uploader-darwin-arm64
```

## GitHub Actions Workflow

This repository includes automated builds using GitHub Actions. Each release will automatically build binaries for all supported platforms.

## Configuration

Profiles are stored in:
- Linux/macOS: `~/.r2uploader/profiles.json`
- Windows: `%USERPROFILE%\.r2uploader\profiles.json`

## License
MIT License - See LICENSE file for details


## Contributing
Don't, but I can't be bothered to remember how to push new builds (which I hope are rare) so here it is future me:
```
git tag v1.0.0
git push origin v1.0.0
```
