/* SPDX-License-Identifier: Apache-2.0
 *
 * Copyright 2023 Damian Peckett <damian@pecke.tt>.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ext4

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dpeckett/args"
)

type Client struct {
	path string
}

// Construct a new e2fsprogs client.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		path: fmt.Sprintf("PATH=%s:/sbin:/usr/sbin", os.Getenv("PATH")),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// CreateOptions provides options for creating an ext4 filesystem.
type CreateOptions struct {
	Device                   string `arg:"0"` // Device where the filesystem will be created.
	Size                     string `arg:"1"` // Optional size of the filesystem.
	CheckForBadBlocks        bool   `arg:"c"` // Check for bad blocks before creating the filesystem.
	BlockSize                *int   `arg:"b"` // Block size in bytes (supported: 1024, 2048 and 4096 bytes).
	ClusterSize              *int   `arg:"C"` // Cluster size in bytes for filesystems using the bigalloc feature (supported: [2048, 256M]).
	BytesPerInode            *int   `arg:"i"` // Bytes/inode ratio, generally shouldn't be smaller than the block size.
	InodeSize                *int   `arg:"I"` // The size of each inode in bytes.
	JournalOptions           string `arg:"J"` // Journal options, comma separated list.
	NumberOfGroups           *int   `arg:"G"` // The number of block groups packed into a flex_bg group.
	NumberOfInodes           *int   `arg:"N"` // Override the default number of reserved inodes.
	RootDirectory            string `arg:"d"` // Copy directory contents into the filesystem.
	ReservedBlocksPercentage *int   `arg:"m"` // Percentage of blocks reserved for the super-user.
	CreatorOS                string `arg:"o"` // Override creator os.
	BlocksPerGroup           *int   `arg:"g"` // The number of blocks in each block group.
	Label                    string `arg:"L"` // Volume label (max length 16 bytes).
	LastMountedDirectory     string `arg:"M"` // Directory where the filesystem was last mounted.
	Features                 string `arg:"O"` // Filesystem features/options, comma separated list.
	FilesystemRevision       *int   `arg:"r"` // Revision level for the filesystem.
	ExtendedOptions          string `arg:"E"` // Extended options, comma separated list.
	UsageType                string `arg:"T"` // Filesystem usage type (supported: floppy, small, default).
	UUID                     string `arg:"U"` // UUID for the filesystem.
	ErrorBehavior            string `arg:"e"` // Kernel behavior when errors are detected (supported: continue, remount-ro, panic).
	UndoFile                 string `arg:"z"` // Before overwriting blocks, backup the contents.
	Journal                  bool   `arg:"j"` // Create an ext3 journal.
	DryRun                   bool   `arg:"n"` // Dry run (don't actually create the filesystem).
	DirectIO                 bool   `arg:"D"` // Use direct I/O when writing to the disk.
	Force                    bool   `arg:"F"` // Force filesystem creation on any device.
	WriteSuperblocks         bool   `arg:"S"` // Write superblock and group descriptors only.
}

// Create an ext4 filesystem.
func (c *Client) CreateFilesystem(ctx context.Context, opts CreateOptions) error {
	cmdArgs := []string{"-q", "-t", "ext4"}
	cmdArgs = append(cmdArgs, args.Marshal(opts)...)

	_, err := c.run(ctx, "mke2fs", cmdArgs...)
	return err
}

// ResizeOptions provides options for resizing an ext4 filesystem.
type ResizeOptions struct {
	Device       string `arg:"0"` // Device containing the filesystem to resize.
	Size         string `arg:"1"` // Optional size of the filesystem.
	Force        bool   `arg:"f"` // Skip safety checks.
	Flush        bool   `arg:"F"` // Flush the device's buffer cache.
	Shrink       bool   `arg:"M"` // Shrink the filesystem to the minimum size.
	Enable64Bit  bool   `arg:"b"` // Enable 64-bit feature.
	Disable64Bit bool   `arg:"s"` // Disable 64-bit feature.
	RAIDStride   *int   `arg:"S"` // RAID stride size in filesystem blocks.
	UndoFile     string `arg:"z"` // Before overwriting blocks, backup the contents.
}

// Resize an ext4 filesystem.
func (c *Client) ResizeFilesystem(ctx context.Context, opts ResizeOptions) error {
	_, err := c.run(ctx, "resize2fs", args.Marshal(opts)...)
	return err
}

// CheckOptions provides options for checking an ext4 filesystem.
type CheckOptions struct {
	Device              string `arg:"0"` // Device containing the filesystem to check.
	Preen               bool   `arg:"p"` // Automatically repair the filesystem.
	NoFix               bool   `arg:"n"` // Perform a read-only check.
	CheckForBadBlocks   bool   `arg:"c"` // Check for bad blocks.
	AppendBadBlocks     bool   `arg:"k"` // Append to existing bad blocks list.
	AppendBadBlocksFile string `arg:"l"` // Append bad blocks from file to existing bad blocks list.
	BadBlocksFile       string `arg:"L"` // Use bad blocks list from file.
	Force               bool   `arg:"f"` // Force checking even if the filesystem seems clean.
	OptimizeDirectories bool   `arg:"D"` // Optimize directories.
	Flush               bool   `arg:"F"` // Flush the device's buffer cache.
	Superblock          *int   `arg:"b"` // Use alternative superblock.
	Blocksize           *int   `arg:"B"` // Block size in bytes (supported: 1024, 2048 and 4096 bytes).
	ExternalJournal     string `arg:"j"` // External journal for the filesystem.
	ExtendedOptions     string `arg:"E"` // Extended options, comma separated list.
	UndoFile            string `arg:"z"` // Before overwriting blocks, backup the contents.
}

// Check an ext4 filesystem.
func (c *Client) CheckFilesystem(ctx context.Context, opts CheckOptions) error {
	var cmdArgs []string
	if !opts.Preen && !opts.NoFix {
		cmdArgs = []string{"-y"}
	}
	cmdArgs = append(cmdArgs, args.Marshal(opts)...)
	_, err := c.run(ctx, "e2fsck", cmdArgs...)
	return err
}

func (c *Client) run(ctx context.Context, cmdName string, cmdArgs ...string) ([]byte, error) {
	cmdPath, err := c.findExecutable(cmdName)
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, cmdPath, cmdArgs...)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, errOut.String())
	}

	return out.Bytes(), nil
}

func (c *Client) findExecutable(cmdName string) (string, error) {
	for _, dir := range filepath.SplitList(c.path) {
		if dir == "" {
			dir = "."
		}
		cmdPath := filepath.Join(filepath.Clean(dir), cmdName)
		if _, err := os.Stat(cmdPath); err == nil {
			return cmdPath, nil
		}
	}

	return "", fmt.Errorf("command not found: %w", os.ErrNotExist)
}
