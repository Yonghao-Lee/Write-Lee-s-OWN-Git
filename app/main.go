package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Usage: your_program.sh <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintf(os.Stderr, "Logs from your program will appear here!\n")

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
		// TODO: Uncomment the code below to pass the first stage!

		for _, dir := range []string{".git", ".git/objects", ".git/refs"} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			}
		}

		headFileContents := []byte("ref: refs/heads/main\n")
		if err := os.WriteFile(".git/HEAD", headFileContents, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %s\n", err)
		}

		fmt.Println("Initialized git directory")
	case "cat-file":
		// Check if we have enough arguments, (we need at least 4: prog, command, flag, hash)
		if len(os.Args) < 4 {
			fmt.Fprintf(os.Stderr, "usage: mygit cat-file -p hash\n")
			os.Exit(1)
		}

		hash := os.Args[3]
		content, err := ReadBlobData(hash)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(string(content))
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}

// ReadBlobData locates, decompresses, and reads a Git object
func ReadBlobData(hash string) ([]byte, error) {
	if len(hash) != 40 {
		return nil, fmt.Errorf("Invalid hash length %d", len(hash))
	}
	// The first 2 chars are used to create the folder to save space
	folder := hash[:2]
	// The rest of the chars are used as filenames
	filename := hash[2:]

	// Combine them into a proper path: .git/objects/xx/xxxx...
	objectPath := filepath.Join(".git", "objects", folder, filename)

	// We open the raw file from our disk
	file, err := os.Open(objectPath)
	if err != nil {
		return nil, fmt.Errorf("Error opening file: %w", err)
	}
	// Use defer to close the file at the very end
	defer file.Close()

	// Wrap the file in a zlib reader to decompress it
	zlibReader, err := zlib.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("Error creating zlib reader: %w", err)
	}
	defer zlibReader.Close()

	// Read all the unzipped data into memory
	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, zlibReader); err != nil {
		return nil, fmt.Errorf("Error reading zlib reader: %w", err)
	}
	allData := buffer.Bytes()

	// Split the data into 2 parts at the first null byte
	parts := bytes.SplitN(allData, []byte{0}, 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("Error reading data from zlib reader")
	}
	return parts[1], nil

}
