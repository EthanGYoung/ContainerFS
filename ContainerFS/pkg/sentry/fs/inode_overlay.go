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
	"strings"

	"gvisor.googlesource.com/gvisor/pkg/abi/linux"
	"gvisor.googlesource.com/gvisor/pkg/log"
	"gvisor.googlesource.com/gvisor/pkg/sentry/context"
	"gvisor.googlesource.com/gvisor/pkg/sentry/socket/unix/transport"
	"gvisor.googlesource.com/gvisor/pkg/syserror"
)

func overlayHasWhiteout(parent *Inode, name string) bool {
	buf, err := parent.Getxattr(XattrOverlayWhiteout(name))
	return err == nil && string(buf) == "y"
}

func overlayCreateWhiteout(parent *Inode, name string) error {
	return parent.InodeOperations.Setxattr(parent, XattrOverlayWhiteout(name), []byte("y"))
}

func overlayWriteOut(ctx context.Context, o *overlayEntry) error {
	// Hot path. Avoid defers.
	var err error
	o.copyMu.RLock()
	if o.upper != nil {
		err = o.upper.InodeOperations.WriteOut(ctx, o.upper)
	}
	o.copyMu.RUnlock()
	return err
}

// overlayLookup performs a lookup in parent.
//
// If name exists, it returns true if the Dirent is in the upper, false if the
// Dirent is in the lower.
func overlayLookup(ctx context.Context, parent *overlayEntry, inode *Inode, name string) (*Dirent, bool, error) {
	log.Infof("overlayLookup for name: " + name)
	
	// Hot path. Avoid defers.
	parent.copyMu.RLock()

	// Assert that there is at least one upper or lower entry.
	if parent.upper == nil && parent.lower == nil {
		parent.copyMu.RUnlock()
		panic("invalid overlayEntry, needs at least one Inode")
	}

	var upperInode *Inode
	var lowerInode *Inode

	// We must remember whether the upper fs returned a negative dirent,
	// because it is only safe to return one if the upper did.
	var negativeUpperChild bool

	// Does the parent directory exist in the upper file system?
	if parent.upper != nil {
		log.Infof("Checking in upper fs: " + parent.upper.MountSource.name)
		log.Infof("TRACE-layer_lookup-" + parent.upper.MountSource.name)
		// First check if a file object exists in the upper file system.
		// A file could have been created over a whiteout, so we need to
		// check if something exists in the upper file system first.
		child, err := parent.upper.Lookup(ctx, name)
		log.Infof("Lookup complete in parent")
		if err != nil && err != syserror.ENOENT {
			log.Infof("Error on overlay lookup")
			// We encountered an error that an overlay cannot handle,
			// we must propagate it to the caller.
			parent.copyMu.RUnlock()
			return nil, false, err
		}
		if child != nil {
			if child.IsNegative() {
				negativeUpperChild = true
			} else {
				log.Infof("TRACE-lookup_match-" + child.Inode.MountSource.name)
				log.Infof("Found upper inode: " + child.Inode.MountSource.name)
				upperInode = child.Inode
				upperInode.IncRef()
			}
			child.DecRef()
		}

		// Are we done?
		if overlayHasWhiteout(parent.upper, name) {
			if upperInode == nil {
				parent.copyMu.RUnlock()
				if negativeUpperChild {
					// If the upper fs returnd a negative
					// Dirent, then the upper is OK with
					// that negative Dirent being cached in
					// the Dirent tree, so we can return
					// one from the overlay.
					return NewNegativeDirent(name), false, nil
				}
				// Upper fs is not OK with a negative Dirent
				// being cached in the Dirent tree, so don't
				// return one.
				return nil, false, syserror.ENOENT
			}
			entry, err := newOverlayEntry(ctx, upperInode, nil, false)
			if err != nil {
				// Don't leak resources.
				upperInode.DecRef()
				parent.copyMu.RUnlock()
				return nil, false, err
			}
			d, err := NewDirent(newOverlayInode(ctx, entry, inode.MountSource), name), nil
			parent.copyMu.RUnlock()
			return d, true, err
		}
	}

	// Check the lower file system. We do this unconditionally (even for
	// non-directories) because we may need to use stable attributes from
	// the lower filesystem (e.g. device number, inode number) that were
	// visible before a copy up.
	if parent.lower != nil {
		log.Infof("Checking in lower fs: " + parent.lower.MountSource.name)
		// Check the lower file system.
		child, err := parent.lower.Lookup(ctx, name)
		log.Infof("Lookup complete in parent-lower")
		// Same song and dance as above.
		if err != nil && err != syserror.ENOENT {
			// Don't leak resources.
			if upperInode != nil {
				upperInode.DecRef()
			}
			parent.copyMu.RUnlock()
			return nil, false, err
		}
		if child != nil {
			if !child.IsNegative() {
				if upperInode == nil {
					// If nothing was in the upper, use what we found in the lower.
					log.Infof("Using lower inode: " + child.Inode.MountSource.name)
					lowerInode = child.Inode
					lowerInode.IncRef()
				} else {
					log.Infof("Using upper inode: " + upperInode.MountSource.name)
					// If we have something from the upper, we can only use it if the types
					// match.
					// NOTE: Allow SpecialDirectories and Directories to merge.
					// This is needed to allow submounts in /proc and /sys.
					if upperInode.StableAttr.Type == child.Inode.StableAttr.Type ||
						(IsDir(upperInode.StableAttr) && IsDir(child.Inode.StableAttr)) {
						lowerInode = child.Inode
						lowerInode.IncRef()
					}
				}
			}
			child.DecRef()
		}
	}

	// Was all of this for naught?
	if upperInode == nil && lowerInode == nil {
		parent.copyMu.RUnlock()
		// We can only return a negative dirent if the upper returned
		// one as well. See comments above regarding negativeUpperChild
		// for more info.
		if negativeUpperChild {
			return NewNegativeDirent(name), false, nil
		}
		return nil, false, syserror.ENOENT
	}

	// Did we find a lower Inode? Remember this because we may decide we don't
	// actually need the lower Inode (see below).
	lowerExists := lowerInode != nil

	// If we found something in the upper filesystem and the lower filesystem,
	// use the stable attributes from the lower filesystem. If we don't do this,
	// then it may appear that the file was magically recreated across copy up.
	if upperInode != nil && lowerInode != nil {
		// Steal attributes.
		upperInode.StableAttr = lowerInode.StableAttr

		// For non-directories, the lower filesystem resource is strictly
		// unnecessary because we don't need to copy-up and we will always
		// operate (e.g. read/write) on the upper Inode.
		if !IsDir(upperInode.StableAttr) {
			lowerInode.DecRef()
			lowerInode = nil
		}
	}

	// Phew, finally done.
	entry, err := newOverlayEntry(ctx, upperInode, lowerInode, lowerExists)
	if err != nil {
		// Well, not quite, we failed at the last moment, how depressing.
		// Be sure not to leak resources.
		if upperInode != nil {
			upperInode.DecRef()
		}
		if lowerInode != nil {
			lowerInode.DecRef()
		}
		parent.copyMu.RUnlock()
		return nil, false, err
	}
	d, err := NewDirent(newOverlayInode(ctx, entry, inode.MountSource), name), nil
	parent.copyMu.RUnlock()
	return d, upperInode != nil, err
}

func overlayCreate(ctx context.Context, o *overlayEntry, parent *Dirent, name string, flags FileFlags, perm FilePermissions) (*File, error) {
	
	log.Infof("overlayCreate -> Called when new file made")
	// Dirent.Create takes renameMu if the Inode is an overlay Inode.
	if err := copyUpLockedForRename(ctx, parent); err != nil {
		return nil, err
	}

	upperFile, err := o.upper.InodeOperations.Create(ctx, o.upper, name, flags, perm)
	if err != nil {
		return nil, err
	}

	// Take another reference on the upper file's inode, which will be
	// owned by the overlay entry.
	upperFile.Dirent.Inode.IncRef()
	entry, err := newOverlayEntry(ctx, upperFile.Dirent.Inode, nil, false)
	if err != nil {
		cleanupUpper(ctx, o.upper, name)
		return nil, err
	}

	// NOTE: Replace the Dirent with a transient Dirent, since
	// we are about to create the real Dirent: an overlay Dirent.
	//
	// This ensures the *fs.File returned from overlayCreate is in the same
	// state as the *fs.File returned by overlayGetFile, where the upper
	// file has a transient Dirent.
	//
	// This is necessary for Save/Restore, as otherwise the upper Dirent
	// (which has no path as it is unparented and never reachable by the
	// user) will clobber the real path for the underlying Inode.
	upperFile.Dirent.Inode.IncRef()
	upperDirent := NewTransientDirent(upperFile.Dirent.Inode)
	upperFile.Dirent.DecRef()
	upperFile.Dirent = upperDirent

	// Create the overlay inode and dirent.  We need this to construct the
	// overlay file.
	overlayInode := newOverlayInode(ctx, entry, parent.Inode.MountSource)
	// d will own the inode reference.
	overlayDirent := NewDirent(overlayInode, name)
	// The overlay file created below with NewFile will take a reference on
	// the overlayDirent, and it should be the only thing holding a
	// reference at the time of creation, so we must drop this reference.
	defer overlayDirent.DecRef()

	// Create a new overlay file that wraps the upper file.
	flags.Pread = upperFile.Flags().Pread
	flags.Pwrite = upperFile.Flags().Pwrite
	overlayFile := NewFile(ctx, overlayDirent, flags, &overlayFileOperations{upper: upperFile})

	return overlayFile, nil
}

func overlayCreateDirectory(ctx context.Context, o *overlayEntry, parent *Dirent, name string, perm FilePermissions) error {
	// Dirent.CreateDirectory takes renameMu if the Inode is an overlay
	// Inode.
	if err := copyUpLockedForRename(ctx, parent); err != nil {
		return err
	}
	return o.upper.InodeOperations.CreateDirectory(ctx, o.upper, name, perm)
}

func overlayCreateLink(ctx context.Context, o *overlayEntry, parent *Dirent, oldname string, newname string) error {
	// Dirent.CreateLink takes renameMu if the Inode is an overlay Inode.
	if err := copyUpLockedForRename(ctx, parent); err != nil {
		return err
	}
	return o.upper.InodeOperations.CreateLink(ctx, o.upper, oldname, newname)
}

func overlayCreateHardLink(ctx context.Context, o *overlayEntry, parent *Dirent, target *Dirent, name string) error {
	// Dirent.CreateHardLink takes renameMu if the Inode is an overlay
	// Inode.
	if err := copyUpLockedForRename(ctx, parent); err != nil {
		return err
	}
	if err := copyUpLockedForRename(ctx, target); err != nil {
		return err
	}
	return o.upper.InodeOperations.CreateHardLink(ctx, o.upper, target.Inode.overlay.upper, name)
}

func overlayCreateFifo(ctx context.Context, o *overlayEntry, parent *Dirent, name string, perm FilePermissions) error {
	// Dirent.CreateFifo takes renameMu if the Inode is an overlay Inode.
	if err := copyUpLockedForRename(ctx, parent); err != nil {
		return err
	}
	return o.upper.InodeOperations.CreateFifo(ctx, o.upper, name, perm)
}

func overlayRemove(ctx context.Context, o *overlayEntry, parent *Dirent, child *Dirent) error {
	// Dirent.Remove and Dirent.RemoveDirectory take renameMu if the Inode
	// is an overlay Inode.
	if err := copyUpLockedForRename(ctx, parent); err != nil {
		return err
	}
	child.Inode.overlay.copyMu.RLock()
	defer child.Inode.overlay.copyMu.RUnlock()
	if child.Inode.overlay.upper != nil {
		if child.Inode.StableAttr.Type == Directory {
			if err := o.upper.InodeOperations.RemoveDirectory(ctx, o.upper, child.name); err != nil {
				return err
			}
		} else {
			if err := o.upper.InodeOperations.Remove(ctx, o.upper, child.name); err != nil {
				return err
			}
		}
	}
	if child.Inode.overlay.lowerExists {
		return overlayCreateWhiteout(o.upper, child.name)
	}
	return nil
}

func overlayRename(ctx context.Context, o *overlayEntry, oldParent *Dirent, renamed *Dirent, newParent *Dirent, newName string, replacement bool) error {
	// To be able to copy these up below, they have to be part of an
	// overlay file system.
	//
	// Maybe some day we can allow the more complicated case of
	// non-overlay X overlay renames, but that's not necessary right now.
	if renamed.Inode.overlay == nil || newParent.Inode.overlay == nil || oldParent.Inode.overlay == nil {
		return syserror.EXDEV
	}

	if replacement {
		// Check here if the file to be replaced exists and is a
		// non-empty directory. If we copy up first, we may end up
		// copying the directory but none of its children, so the
		// directory will appear empty in the upper fs, which will then
		// allow the rename to proceed when it should return ENOTEMPTY.
		//
		// NOTE: Ideally, we'd just pass in the replaced
		// Dirent from Rename, but we must drop the reference on
		// replaced before we make the rename call, so Rename can't
		// pass the Dirent to the Inode without significantly
		// complicating the API. Thus we look it up again here.
		//
		// For the same reason we can't use defer here.
		replaced, inUpper, err := overlayLookup(ctx, newParent.Inode.overlay, newParent.Inode, newName)
		// If err == ENOENT or a negative Dirent is returned, then
		// newName has been removed out from under us. That's fine;
		// filesystems where that can happen must handle stale
		// 'replaced'.
		if err != nil && err != syserror.ENOENT {
			return err
		}
		if err == nil {
			if !inUpper {
				// newName doesn't exist in
				// newParent.Inode.overlay.upper, thus from
				// that Inode's perspective this won't be a
				// replacing rename.
				replacement = false
			}

			if !replaced.IsNegative() && IsDir(replaced.Inode.StableAttr) {
				children, err := readdirOne(ctx, replaced)
				if err != nil {
					replaced.DecRef()
					return err
				}

				// readdirOne ensures that "." and ".." are not
				// included among the returned children, so we don't
				// need to bother checking for them.
				if len(children) > 0 {
					replaced.DecRef()
					return syserror.ENOTEMPTY
				}
			}

			replaced.DecRef()
		}
	}

	if err := copyUpLockedForRename(ctx, renamed); err != nil {
		return err
	}
	if err := copyUpLockedForRename(ctx, newParent); err != nil {
		return err
	}
	oldName := renamed.name
	if err := o.upper.InodeOperations.Rename(ctx, oldParent.Inode.overlay.upper, oldName, newParent.Inode.overlay.upper, newName, replacement); err != nil {
		return err
	}
	if renamed.Inode.overlay.lowerExists {
		return overlayCreateWhiteout(oldParent.Inode.overlay.upper, oldName)
	}
	return nil
}

func overlayBind(ctx context.Context, o *overlayEntry, name string, data transport.BoundEndpoint, perm FilePermissions) (*Dirent, error) {
	o.copyMu.RLock()
	defer o.copyMu.RUnlock()
	// We do not support doing anything exciting with sockets unless there
	// is already a directory in the upper filesystem.
	if o.upper == nil {
		return nil, syserror.EOPNOTSUPP
	}
	d, err := o.upper.InodeOperations.Bind(ctx, o.upper, name, data, perm)
	if err != nil {
		return nil, err
	}

	// Grab the inode and drop the dirent, we don't need it.
	inode := d.Inode
	inode.IncRef()
	d.DecRef()

	// Create a new overlay entry and dirent for the socket.
	entry, err := newOverlayEntry(ctx, inode, nil, false)
	if err != nil {
		inode.DecRef()
		return nil, err
	}
	return NewDirent(newOverlayInode(ctx, entry, inode.MountSource), name), nil
}

func overlayBoundEndpoint(o *overlayEntry, path string) transport.BoundEndpoint {
	o.copyMu.RLock()
	defer o.copyMu.RUnlock()

	if o.upper != nil {
		return o.upper.InodeOperations.BoundEndpoint(o.upper, path)
	}

	// If the lower is itself an overlay, recurse.
	if o.lower.overlay != nil {
		return overlayBoundEndpoint(o.lower.overlay, path)
	}
	// Lower is not an overlay. Call BoundEndpoint directly.
	return o.lower.InodeOperations.BoundEndpoint(o.lower, path)
}

// TODO: Here it is
func overlayGetFile(ctx context.Context, o *overlayEntry, d *Dirent, flags FileFlags) (*File, error) {
	// Hot path. Avoid defers.
	if flags.Write {
		if err := copyUp(ctx, d); err != nil {
			return nil, err
		}
	}

	o.copyMu.RLock()

	if o.upper != nil {
		log.Infof("Get file in upper overlay: %v", o.upper.MountSource.name)
		upper, err := overlayFile(ctx, o.upper, flags)
		if err != nil {
			o.copyMu.RUnlock()
			return nil, err
		}
		flags.Pread = upper.Flags().Pread
		flags.Pwrite = upper.Flags().Pwrite
		f, err := NewFile(ctx, d, flags, &overlayFileOperations{upper: upper}), nil
		o.copyMu.RUnlock()
		return f, err
	}

	log.Infof("Get file in lower overlay: %v", o.lower.MountSource.name)
	lower, err := overlayFile(ctx, o.lower, flags)
	if err != nil {
		o.copyMu.RUnlock()
		return nil, err
	}
	flags.Pread = lower.Flags().Pread
	flags.Pwrite = lower.Flags().Pwrite
	o.copyMu.RUnlock()
	return NewFile(ctx, d, flags, &overlayFileOperations{lower: lower}), nil
}

func overlayUnstableAttr(ctx context.Context, o *overlayEntry) (UnstableAttr, error) {
	// Hot path. Avoid defers.
	var (
		attr UnstableAttr
		err  error
	)
	o.copyMu.RLock()
	if o.upper != nil {
		attr, err = o.upper.UnstableAttr(ctx)
	} else {
		attr, err = o.lower.UnstableAttr(ctx)
	}
	o.copyMu.RUnlock()
	return attr, err
}

func overlayGetxattr(o *overlayEntry, name string) ([]byte, error) {
	// Hot path. This is how the overlay checks for whiteout files.
	// Avoid defers.
	var (
		b   []byte
		err error
	)

	// Don't forward the value of the extended attribute if it would
	// unexpectedly change the behavior of a wrapping overlay layer.
	if strings.HasPrefix(XattrOverlayPrefix, name) {
		return nil, syserror.ENODATA
	}

	o.copyMu.RLock()
	if o.upper != nil {
		b, err = o.upper.Getxattr(name)
	} else {
		b, err = o.lower.Getxattr(name)
	}
	o.copyMu.RUnlock()
	return b, err
}

func overlayListxattr(o *overlayEntry) (map[string]struct{}, error) {
	o.copyMu.RLock()
	defer o.copyMu.RUnlock()
	var names map[string]struct{}
	var err error
	if o.upper != nil {
		names, err = o.upper.Listxattr()
	} else {
		names, err = o.lower.Listxattr()
	}
	for name := range names {
		// Same as overlayGetxattr, we shouldn't forward along
		// overlay attributes.
		if strings.HasPrefix(XattrOverlayPrefix, name) {
			delete(names, name)
		}
	}
	return names, err
}

func overlayCheck(ctx context.Context, o *overlayEntry, p PermMask) error {
	o.copyMu.RLock()
	// Hot path. Avoid defers.
	var err error
	if o.upper != nil {
		err = o.upper.check(ctx, p)
	} else {
		if p.Write {
			// Since writes will be redirected to the upper filesystem, the lower
			// filesystem need not be writable, but must be readable for copy-up.
			p.Write = false
			p.Read = true
		}
		err = o.lower.check(ctx, p)
	}
	o.copyMu.RUnlock()
	return err
}

func overlaySetPermissions(ctx context.Context, o *overlayEntry, d *Dirent, f FilePermissions) bool {
	if err := copyUp(ctx, d); err != nil {
		return false
	}
	return o.upper.InodeOperations.SetPermissions(ctx, o.upper, f)
}

func overlaySetOwner(ctx context.Context, o *overlayEntry, d *Dirent, owner FileOwner) error {
	if err := copyUp(ctx, d); err != nil {
		return err
	}
	return o.upper.InodeOperations.SetOwner(ctx, o.upper, owner)
}

func overlaySetTimestamps(ctx context.Context, o *overlayEntry, d *Dirent, ts TimeSpec) error {
	if err := copyUp(ctx, d); err != nil {
		return err
	}
	return o.upper.InodeOperations.SetTimestamps(ctx, o.upper, ts)
}

func overlayTruncate(ctx context.Context, o *overlayEntry, d *Dirent, size int64) error {
	if err := copyUp(ctx, d); err != nil {
		return err
	}
	return o.upper.InodeOperations.Truncate(ctx, o.upper, size)
}

func overlayReadlink(ctx context.Context, o *overlayEntry) (string, error) {
	o.copyMu.RLock()
	defer o.copyMu.RUnlock()
	if o.upper != nil {
		log.Infof("Readinglink in upper")
		return o.upper.Readlink(ctx)
	}
	log.Infof("Reading link in lower")
	return o.lower.Readlink(ctx)
}

func overlayGetlink(ctx context.Context, o *overlayEntry) (*Dirent, error) {
	log.Infof("overlayGetLink called")
	var dirent *Dirent
	var err error

	o.copyMu.RLock()
	defer o.copyMu.RUnlock()

	if o.upper != nil {
		log.Infof("Upper FS in OverlayGetLink")
		dirent, err = o.upper.Getlink(ctx)
	} else {
		log.Infof("Upper FS in OverlayGetLink")
		dirent, err = o.lower.Getlink(ctx)
	}
	
	if dirent != nil {
		// This dirent is likely bogus (its Inode likely doesn't contain
		// the right overlayEntry). So we're forced to drop it on the
		// ground and claim that jumping around the filesystem like this
		// is not supported.
		name, _ := dirent.FullName(nil)
		dirent.DecRef()

		// Claim that the path is not accessible.
		err = syserror.EACCES
		log.Infof("Getlink not supported in overlay for %q", name)
	}
	log.Infof("returning from overlayGetlink")
	return nil, err
}

func overlayStatFS(ctx context.Context, o *overlayEntry) (Info, error) {
	o.copyMu.RLock()
	defer o.copyMu.RUnlock()

	var i Info
	var err error
	if o.upper != nil {
		i, err = o.upper.StatFS(ctx)
	} else {
		i, err = o.lower.StatFS(ctx)
	}
	if err != nil {
		return Info{}, err
	}

	i.Type = linux.OVERLAYFS_SUPER_MAGIC

	return i, nil
}

// NewTestOverlayDir returns an overlay Inode for tests.
//
// If `revalidate` is true, then the upper filesystem will require
// revalidation.
func NewTestOverlayDir(ctx context.Context, upper, lower *Inode, revalidate bool) *Inode {
	fs := &overlayFilesystem{}
	var upperMsrc *MountSource
	if revalidate {
		upperMsrc = NewRevalidatingMountSource(fs, MountSourceFlags{})
	} else {
		upperMsrc = NewNonCachingMountSource(fs, MountSourceFlags{})
	}
	msrc := NewMountSource(&overlayMountSourceOperations{
		upper: upperMsrc,
		lower: NewNonCachingMountSource(fs, MountSourceFlags{}),
	}, fs, MountSourceFlags{},"")
	overlay := &overlayEntry{
		upper: upper,
		lower: lower,
	}
	return newOverlayInode(ctx, overlay, msrc)
}

// TestHasUpperFS returns true if i is an overlay Inode and it has a pointer
// to an Inode on an upper filesystem.
func (i *Inode) TestHasUpperFS() bool {
	return i.overlay != nil && i.overlay.upper != nil
}

// TestHasLowerFS returns true if i is an overlay Inode and it has a pointer
// to an Inode on a lower filesystem.
func (i *Inode) TestHasLowerFS() bool {
	return i.overlay != nil && i.overlay.lower != nil
}
