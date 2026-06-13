package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// main dispatches the requested subcommand.
//
// Usage: mygit <command> [<args>...]
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: mygit <command> [<args>...]\n")
		os.Exit(1)
	}

	switch command := os.Args[1]; command {
	case "init":
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
		// Expect: mygit cat-file -p <hash> (program, command, flag, hash).
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
	case "hash-object":
		// Supports: mygit hash-object [-w] <file>
		// The -w flag tells us to write the object to .git/objects; without it we
		// only compute and print the hash.
		args := os.Args[2:]
		write := len(args) > 0 && args[0] == "-w"
		if write {
			args = args[1:] // drop the consumed flag, leaving just the file path
		}
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "usage: mygit hash-object [-w] <file>\n")
			os.Exit(1)
		}

		hash, err := HashObject(args[0], write)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error hashing object: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(hash)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", command)
		os.Exit(1)
	}
}

// ReadBlobData returns the payload of the loose Git object identified by hash.
//
// The object is read from .git/objects, zlib-decompressed, and stripped of its
// "<type> <size>\0" header; only the content that follows is returned. hash must
// be the full 40-character SHA-1 hex digest.
func ReadBlobData(hash string) ([]byte, error) {
	if len(hash) != 40 {
		return nil, fmt.Errorf("invalid hash length %d", len(hash))
	}

	// Loose objects are stored at .git/objects/<first 2 hex chars>/<remaining 38>.
	objectPath := filepath.Join(".git", "objects", hash[:2], hash[2:])

	file, err := os.Open(objectPath)
	if err != nil {
		return nil, fmt.Errorf("open object: %w", err)
	}
	defer file.Close()

	// Loose objects are zlib-compressed on disk.
	zlibReader, err := zlib.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("create zlib reader: %w", err)
	}
	defer zlibReader.Close()

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, zlibReader); err != nil {
		return nil, fmt.Errorf("decompress object: %w", err)
	}

	// Separate the "<type> <size>\0" header from the payload at the first NUL byte.
	parts := bytes.SplitN(buffer.Bytes(), []byte{0}, 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("malformed object: missing header terminator")
	}
	return parts[1], nil
}

// HashObject computes the 40-character SHA-1 hex digest of filePath as a Git
// blob object. When write is true it also stores the object on disk (the
// behaviour of `git hash-object -w`); when false it only returns the digest.
//
// The on-disk format mirrors real Git: the file content is framed with a
// "blob <size>\0" header, the SHA-1 of that framed data becomes the object's
// ID, and the framed data is zlib-compressed into .git/objects/<id[:2]>/<id[2:]>.
func HashObject(filePath string, write bool) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	// A blob object is the file content prefixed with the header
	// "blob <byte-count>\0" (\0 is a single NUL byte separating header and content).
	header := []byte(fmt.Sprintf("blob %d\x00", len(content)))
	fullData := append(header, content...) // header bytes followed by the content bytes

	// The object's ID is the SHA-1 of the framed data (header + content).
	hasher := sha1.New()
	hasher.Write(fullData)
	hashBytes := hasher.Sum(nil)            // raw 20-byte digest
	hashStr := fmt.Sprintf("%x", hashBytes) // rendered as 40 hex characters

	// Without -w we report the ID but leave the object store untouched.
	if !write {
		return hashStr, nil
	}

	// Objects are sharded by the first two hex chars of the ID: those two chars
	// name the directory, the remaining 38 name the file inside it.
	objectDir := filepath.Join(".git", "objects", hashStr[:2])
	if err := os.MkdirAll(objectDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir objects: %w", err)
	}

	objectPath := filepath.Join(objectDir, hashStr[2:])
	file, err := os.OpenFile(objectPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("open object: %w", err)
	}
	defer file.Close()

	// Loose objects are stored zlib-compressed; write the framed data through a
	// zlib writer so it lands on disk compressed.
	zlibWriter := zlib.NewWriter(file)
	if _, err := zlibWriter.Write(fullData); err != nil {
		zlibWriter.Close()
		return "", fmt.Errorf("write object: %w", err)
	}

	// Close flushes any buffered compressed data, so its error must be checked.
	if err := zlibWriter.Close(); err != nil {
		return "", fmt.Errorf("close zlib writer: %w", err)
	}
	return hashStr, nil
}
