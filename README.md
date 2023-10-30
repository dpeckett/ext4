# ext4

A Go wrapper for working with ext4 filesystems (e2fsprogs).

## Usage

For detailed examples see the [ext4_test.go](./ext4_test.go) file.

```go
package main

import (
    "context"
    "log"

    "github.com/dpeckett/ext4"
)

func main() {
    c := ext4.NewClient()

    ctx := context.Background()

    err := c.CreateFilesystem(ctx, ext4.CreateOptions{
        Device: "/dev/loop0",
    })
    if err != nil {
        log.Fatal("Failed to create filesystem: ", err)
    }

    // Shrink the filesystem to its minimum size.
    err = c.ResizeFilesystem(ctx, ext4.ResizeOptions{
        Device: "/dev/loop0",
        Shrink: true,
    })
    if err != nil {
        log.Fatal("Failed to resize filesystem: ", err)
    }
}
```

## Commands

This is a work in progress. The following commands are implemented:

- [x] e2fsck
- [ ] e2image
- [ ] e2label
- [ ] e2mmpstatus
- [ ] e2scrub
- [ ] e2undo
- [x] mke2fs
- [x] resize2fs
- [ ] tune2fs