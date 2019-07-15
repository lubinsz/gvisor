// Copyright 2019 The gVisor Authors.
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

package ext

import (
	"io"

	"gvisor.dev/gvisor/pkg/binary"
	"gvisor.dev/gvisor/pkg/sentry/fs/ext/disklayout"
	"gvisor.dev/gvisor/pkg/syserror"
)

// Inode represents an ext inode.
type Inode struct {
	// diskInode gives us access to the inode struct on disk. Immutable.
	diskInode disklayout.Inode

	// root is the root extent node. This lives in the 60 byte inode.Blocks field.
	// Immutable.
	root disklayout.ExtentNode
}

// buildExtTree builds the extent tree by reading it from disk by doing
// running a simple DFS. It first reads the root node from the inode struct in
// memory. Then it recursively builds the rest of the tree by reading it off
// disk.
//
// Preconditions:
//   - Must have mutual exclusion on device fd.
//   - Inode flag InExtents must be set.
func (in *Inode) buildExtTree(dev io.ReadSeeker, blkSize uint64) error {
	rootNodeData := in.diskInode.Blocks()

	var rootHeader disklayout.ExtentHeader
	binary.Unmarshal(rootNodeData[:disklayout.ExtentStructsSize], binary.LittleEndian, &rootHeader)

	// Root node can not have more than 4 entries: 60 bytes = 1 header + 4 entries.
	if rootHeader.NumEntries > 4 {
		// read(2) specifies that EINVAL should be returned if the file is unsuitable
		// for reading.
		return syserror.EINVAL
	}

	rootEntries := make([]disklayout.ExtentEntryPair, rootHeader.NumEntries)
	for i, off := rootHeader.NumEntries, disklayout.ExtentStructsSize; i > 0; i, off = i-1, off+disklayout.ExtentStructsSize {
		var curEntry disklayout.ExtentEntry
		if rootHeader.Height == 0 {
			// Leaf node.
			curEntry = &disklayout.Extent{}
		} else {
			// Internal node.
			curEntry = &disklayout.ExtentIdx{}
		}
		binary.Unmarshal(rootNodeData[off:off+disklayout.ExtentStructsSize], binary.LittleEndian, curEntry)
		rootEntries[i].Entry = curEntry
	}

	// If this node is internal, perform DFS.
	if rootHeader.Height > 0 {
		for i := rootHeader.NumEntries; i > 0; i-- {
			var err error
			rootEntries[i].Node, err = buildExtTreeFromDisk(dev, rootEntries[i].Entry, blkSize)
			if err != nil {
				return err
			}
		}
	}

	in.root = disklayout.ExtentNode{rootHeader, rootEntries}
	return nil
}

// buildExtTreeFromDisk reads the extent tree nodes from disk and recursively
// builds the tree. Performs a simple DFS. It returns the ExtentNode pointed to
// by the ExtentEntry.
//
// Preconditions: Must have mutual exclusion on device fd.
func buildExtTreeFromDisk(dev io.ReadSeeker, entry disklayout.ExtentEntry, blkSize uint64) (*disklayout.ExtentNode, error) {
	off := uint64(entry.FileBlock()) * blkSize
	var header disklayout.ExtentHeader
	err := readFromDisk(dev, int64(off), &header)
	if err != nil {
		return nil, err
	}

	entries := make([]disklayout.ExtentEntryPair, header.NumEntries)
	for i, off := header.NumEntries, off+disklayout.ExtentStructsSize; i > 0; i, off = i-1, off+disklayout.ExtentStructsSize {
		var curEntry disklayout.ExtentEntry
		if header.Height == 0 {
			// Leaf node.
			curEntry = &disklayout.Extent{}
		} else {
			// Internal node.
			curEntry = &disklayout.ExtentIdx{}
		}

		err := readFromDisk(dev, int64(off), curEntry)
		if err != nil {
			return nil, err
		}
		entries[i].Entry = curEntry
	}

	// If this node is internal, perform DFS.
	if header.Height > 0 {
		for i := header.NumEntries; i > 0; i-- {
			var err error
			entries[i].Node, err = buildExtTreeFromDisk(dev, entries[i].Entry, blkSize)
			if err != nil {
				return nil, err
			}
		}
	}

	return &disklayout.ExtentNode{header, entries}, nil
}
