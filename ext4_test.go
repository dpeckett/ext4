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

package ext4_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/dpeckett/ext4"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	err := loadNBDModule()
	require.NoError(t, err)

	t.Log("Creating virtual block device")

	imagePath := filepath.Join(t.TempDir(), ".qcow2")
	err = createImage(imagePath)
	require.NoError(t, err)

	devPath, err := attachNBDDevice(imagePath)
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Log("Detaching virtual block device")

		err := detachNBDDevice(devPath)
		require.NoError(t, err)
	})

	t.Log("Creating ext4 filesystem")

	c := ext4.NewClient()

	err = c.CreateFilesystem(context.Background(), ext4.CreateFSOptions{
		Device: devPath,
		Size:   "100M",
		Label:  t.Name(),
	})
	require.NoError(t, err, "failed to create ext4 filesystem")

	t.Log("Mounting ext4 filesystem")

	mountPath := t.TempDir()
	err = exec.Command("mount", devPath, mountPath).Run()
	require.NoError(t, err, "failed to mount ext4 filesystem")

	t.Cleanup(func() {
		t.Log("Unmounting ext4 filesystem")

		err := exec.Command("umount", mountPath).Run()
		require.NoError(t, err, "failed to unmount ext4 filesystem")
	})

	t.Log("Verifying filesystem size")

	cmd := exec.Command("df", "-B1", mountPath)
	output, err := cmd.Output()
	require.NoError(t, err, "failed to get filesystem size")

	size, err := strconv.Atoi(strings.Fields(strings.Split(string(output), "\n")[1])[1])
	require.NoError(t, err, "failed to parse filesystem size")

	require.InEpsilon(t, 1.0, float32(size)/100000000.0, 0.25, "unexpected filesystem size")

	t.Log("Writing and verifying file on ext4 filesystem")

	err = os.WriteFile(filepath.Join(mountPath, "test.txt"), []byte("hello world"), 0o644)
	require.NoError(t, err, "failed to write file to ext4 filesystem")

	cmd = exec.Command("sha256sum", filepath.Join(mountPath, "test.txt"))
	output, err = cmd.Output()
	require.NoError(t, err, "failed to sha256sum file on ext4 filesystem")

	expectedHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	require.Contains(t, string(output), expectedHash, "file contents do not match")

	t.Log("Verifying filesystem label")

	cmd = exec.Command("e2label", devPath)
	output, err = cmd.Output()
	require.NoError(t, err, "failed to get filesystem label")

	require.Equal(t, t.Name(), strings.TrimSpace(string(output)), "filesystem label does not match")

	t.Log("Resizing ext4 filesystem")

	err = c.ResizeFilesystem(context.Background(), ext4.ResizeFSOptions{
		Device: devPath,
		Size:   "500M",
	})
	require.NoError(t, err, "failed to resize ext4 filesystem")

	t.Log("Verifying filesystem size")

	cmd = exec.Command("df", "-B1", mountPath)
	output, err = cmd.Output()
	require.NoError(t, err, "failed to get filesystem size")

	size, err = strconv.Atoi(strings.Fields(strings.Split(string(output), "\n")[1])[1])
	require.NoError(t, err, "failed to parse filesystem size")

	require.InEpsilon(t, 1.0, float32(size)/500000000.0, 0.25, "unexpected filesystem size")
}

func loadNBDModule() error {
	cmd := exec.Command("/sbin/modprobe", "nbd")
	return cmd.Run()
}

func createImage(imagePath string) error {
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", imagePath, "1G")
	return cmd.Run()
}

func attachNBDDevice(imagePath string) (string, error) {
	for i := 0; i < 16; i++ {
		devPath := fmt.Sprintf("/dev/nbd%d", i)
		cmd := exec.Command("qemu-nbd", "-c", devPath, imagePath)
		err := cmd.Run()
		if err == nil {
			return devPath, nil
		}
	}

	return "", fmt.Errorf("no free nbd device found")
}

func detachNBDDevice(devPath string) error {
	cmd := exec.Command("qemu-nbd", "-d", devPath)
	return cmd.Run()
}
