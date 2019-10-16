// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fs

import (
	"gvisor.googlesource.com/gvisor/pkg/sentry/context"
)

// overlayMountSourceOperations implements MountSourceOperations for an overlay
// mount point. The upper filesystem determines the caching behavior of the
// overlay.
//
// +stateify savable
type overlayMountSourceOperations struct {
	upper *MountSource
	lower *MountSource
}

func newOverlayMountSource(upper, lower *MountSource, flags MountSourceFlags) *MountSource {
	upper.IncRef()
	lower.IncRef()
	return NewMountSource(&overlayMountSourceOperations{
		upper: upper,
		lower: lower,
	}, &overlayFilesystem{}, flags, "overlay-mount")
}

// Revalidate implements MountSourceOperations.Revalidate for an overlay by
// delegating to the upper filesystem's Revalidate method. We cannot reload
// files from the lower filesystem, so we panic if the lower filesystem's
// Revalidate method returns true.
func (o *overlayMountSourceOperations) Revalidate(ctx context.Context, name string, parent, child *Inode) bool {
	if child.overlay == nil {
		panic("overlay cannot revalidate inode that is not an overlay")
	}

	// Revalidate is never called on a mount point, so parent and child
	// must be from the same mount, and thus must both be overlay inodes.
	if parent.overlay == nil {
		panic("trying to revalidate an overlay inode but the parent is not an overlay")
	}

	// We can't revalidate from the lower filesystem.
	if child.overlay.lower != nil && o.lower.Revalidate(ctx, name, parent.overlay.lower, child.overlay.lower) {
		panic("an overlay cannot revalidate file objects from the lower fs")
	}

	// Do we have anything to revalidate?
	if child.overlay.upper == nil {
		return false
	}

	// Does the upper require revalidation?
	return o.upper.Revalidate(ctx, name, parent.overlay.upper, child.overlay.upper)
}

// Keep implements MountSourceOperations by delegating to the upper
// filesystem's Keep method.
func (o *overlayMountSourceOperations) Keep(dirent *Dirent) bool {
	return o.upper.Keep(dirent)
}

// ResetInodeMappings propagates the call to both upper and lower MountSource.
func (o *overlayMountSourceOperations) ResetInodeMappings() {
	o.upper.ResetInodeMappings()
	o.lower.ResetInodeMappings()
}

// SaveInodeMapping propagates the call to both upper and lower MountSource.
func (o *overlayMountSourceOperations) SaveInodeMapping(inode *Inode, path string) {
	inode.overlay.copyMu.RLock()
	defer inode.overlay.copyMu.RUnlock()
	if inode.overlay.upper != nil {
		o.upper.SaveInodeMapping(inode.overlay.upper, path)
	}
	if inode.overlay.lower != nil {
		o.lower.SaveInodeMapping(inode.overlay.lower, path)
	}
}

// Destroy drops references on the upper and lower MountSource.
func (o *overlayMountSourceOperations) Destroy() {
	o.upper.DecRef()
	o.lower.DecRef()
}

// type overlayFilesystem is the filesystem for overlay mounts.
//
// +stateify savable
type overlayFilesystem struct{}

// Name implements Filesystem.Name.
func (ofs *overlayFilesystem) Name() string {
	return "overlayfs"
}

// Custom name for tracing bloom filters
func (ofs *overlayFilesystem) CustomName() string {
	return "Custom mount trace"
}

// TODO: Set cusome name

// Custom name for tracing bloom filters
func (ofs *overlayFilesystem) SetCustomName() string {
	return "Custom mount trace"
}

// Flags implements Filesystem.Flags.
func (ofs *overlayFilesystem) Flags() FilesystemFlags {
	return 0
}

// AllowUserMount implements Filesystem.AllowUserMount.
func (ofs *overlayFilesystem) AllowUserMount() bool {
	return false
}

// AllowUserList implements Filesystem.AllowUserList.
func (*overlayFilesystem) AllowUserList() bool {
	return true
}

// Mount implements Filesystem.Mount.
func (ofs *overlayFilesystem) Mount(ctx context.Context, device string, flags MountSourceFlags, data string, _ interface{}) (*Inode, error) {
	panic("overlayFilesystem.Mount should not be called!")
}
