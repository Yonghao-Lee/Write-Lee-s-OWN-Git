# Write Lee's OWN Git

A from-scratch implementation of Git written in Go, built as I work through the
[CodeCrafters "Build Your Own Git"](https://codecrafters.io/challenges/git)
challenge. The goal is to understand how Git actually works under the hood — the
`.git` directory layout, content-addressable object storage, zlib compression,
and Git's transfer protocols — by reimplementing the plumbing commands myself.

## Implemented commands

| Command            | Description                                                            |
| ------------------ | --------------------------------------------------------------------- |
| `init`             | Initializes a new repository (`.git/`, `.git/objects`, `.git/refs`, `HEAD`). |
| `cat-file -p <sha>`| Reads a Git object: locates it in `.git/objects`, zlib-decompresses it, strips the header, and prints the content. |

More commands (hashing objects, writing trees, creating commits, and cloning a
public repository) are on the way as I progress through the challenge.

## How it works

Git stores every object (blob, tree, commit) under
`.git/objects/<first 2 chars of sha>/<remaining 38 chars>`. Each object is
zlib-compressed and prefixed with a header of the form `"<type> <size>\0"`
before the raw content. This implementation:

1. Splits the 40-character SHA-1 hash into a 2-char directory and a 38-char filename.
2. Opens the object file and wraps it in a `zlib` reader to decompress it.
3. Splits on the first null byte to separate the header from the payload, then
   returns the payload.

## Building and running

Requires Go 1.26+.

```sh
# Build
go build -o mygit ./app

# Initialize a repository
./mygit init

# Print the contents of an object
./mygit cat-file -p <40-char-sha>
```

> **Tip:** run this in a throwaway directory (e.g. `/tmp/testing`) so you don't
> touch the real `.git` folder of this repo.

```sh
mkdir -p /tmp/testing && cd /tmp/testing
/path/to/this/repo/your_program.sh init
```

## Project layout

- `app/main.go` — the entire implementation (command dispatch + object reading).
- `your_program.sh` — build-and-run wrapper used both locally and by the CodeCrafters test runner.
- `codecrafters.yml`, `.codecrafters/` — CodeCrafters challenge scaffolding (test harness configuration). Left in place so the challenge tests keep running; not part of the Git implementation itself.

## License

[MIT](./LICENSE)
