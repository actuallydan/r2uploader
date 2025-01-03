package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CloudflareCredentials struct {
	APIToken   string
	AccessKey  string
	SecretKey  string
	AccountID  string
	BucketName string
}

type FileInfo struct {
	Path string
	Size int64
}

func getInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')

	// Clean the input by removing quotes and trimming spaces
	cleaned := strings.TrimSpace(input)
	cleaned = strings.Trim(cleaned, "'")
	cleaned = strings.Trim(cleaned, "\"")

	return cleaned
}

func getCredentialsFromUser() (CloudflareCredentials, error) {
	creds := CloudflareCredentials{}

	for creds.APIToken == "" {
		creds.APIToken = getInput("Enter Cloudflare API Token: ")
	}

	for creds.AccountID == "" {
		creds.AccountID = getInput("Enter Cloudflare Account ID: ")
	}

	for creds.AccessKey == "" {
		creds.AccessKey = getInput("Enter Access Key: ")
	}

	for creds.SecretKey == "" {
		creds.SecretKey = getInput("Enter Secret Key: ")
	}

	for creds.BucketName == "" {
		creds.BucketName = getInput("Enter R2 Bucket Name: ")
	}

	return creds, nil
}

func getFilesInfo(path string) ([]FileInfo, error) {
	var files []FileInfo

	// Clean and evaluate the path to handle special characters
	cleanPath, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		// If EvalSymlinks fails, try using the original path
		cleanPath = path
	}

	// Try to escape the path if it contains special characters
	if strings.ContainsAny(cleanPath, "[]() ") {
		cleanPath = fmt.Sprintf(`%s`, cleanPath)
	}

	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("error accessing path: %v\nPath tried: %s", err, cleanPath)
	}

	if fileInfo.IsDir() {
		err := filepath.Walk(cleanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, FileInfo{
					Path: path,
					Size: info.Size(),
				})
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory: %v", err)
		}
	} else {
		files = append(files, FileInfo{
			Path: cleanPath,
			Size: fileInfo.Size(),
		})
	}

	// Verify all files exist before proceeding
	for _, file := range files {
		if _, err := os.Stat(file.Path); err != nil {
			return nil, fmt.Errorf("error accessing file %s: %v", file.Path, err)
		}
	}

	return files, nil
}

func confirmUpload(files []FileInfo) bool {
	fmt.Printf("\nFound %d files to upload:\n", len(files))
	var totalSize int64
	for _, file := range files {
		fmt.Printf("- %s (%.2f MB)\n", file.Path, float64(file.Size)/(1024*1024))
		totalSize += file.Size
	}
	fmt.Printf("\nTotal size: %.2f MB\n", float64(totalSize)/(1024*1024))

	confirmation := getInput("\nDo you want to proceed with the upload? (Y/n): ")
	return confirmation == "" || strings.ToLower(confirmation) == "y"
}

func uploadFiles(creds CloudflareCredentials, files []FileInfo, sourcePath string) error {
	uploader, err := NewR2Uploader(creds)
	if err != nil {
		return fmt.Errorf("failed to create uploader: %v", err)
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get source info: %v", err)
	}

	var baseDir string
	if sourceInfo.IsDir() {
		baseDir = filepath.Base(sourcePath)
	}

	// Store uploaded file keys
	var uploadedKeys []string

	for i, file := range files {
		fmt.Printf("[%d/%d] Uploading %s...\n", i+1, len(files), file.Path)
		key, err := uploader.UploadFile(creds.BucketName, file.Path, baseDir)
		if err != nil {
			return fmt.Errorf("failed to upload %s: %v", file.Path, err)
		}
		uploadedKeys = append(uploadedKeys, key)
	}

	// Generate and display presigned URLs
	fmt.Println("\nPresigned URLs (valid for 24 hours):")
	for i, key := range uploadedKeys {
		url, err := uploader.GetPresignedURL(creds.BucketName, key)
		if err != nil {
			fmt.Printf("Warning: Couldn't generate URL for %s: %v\n", key, err)
			continue
		}
		fmt.Printf("%d. %s\n   URL: %s\n", i+1, key, url)
	}

	return nil
}

func uploadLoop() {
	// Initialize profile manager
	pm, err := NewProfileManager()
	if err != nil {
		fmt.Printf("Error initializing profile manager: %v\n", err)
		os.Exit(1)
	}

	// Get credentials using profile manager
	creds, err := pm.getCredentials()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	for {
		// Get file/directory path
		path := getInput("\nEnter file or directory path (or 'q' to quit): ")
		if strings.ToLower(path) == "q" {
			fmt.Println("Goodbye!")
			return
		}

		// Get files information
		files, err := getFilesInfo(path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Confirm upload
		if !confirmUpload(files) {
			fmt.Println("Upload cancelled.")
			continue
		}

		// Upload files
		fmt.Println("\nUploading files...")
		err = uploadFiles(creds, files, path)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Println("\nUpload completed successfully!")
	}
}

func main() {
	fmt.Println("Cloudflare R2 File Uploader")
	fmt.Println("==========================")
	uploadLoop()
}
