// Package vfs holds the wire surface of the vfs module: the entry and info
// types, the three actions, the error values, the operation names and the
// canonical listing encoder (VFS 6, sections 4, 5, 7 and 8).
package vfs

const (
	ModuleName = "vfs"

	MethodLs       = "vfs.ls"
	MethodStat     = "vfs.stat"
	MethodStatfs   = "vfs.statfs"
	MethodRead     = "vfs.read"
	MethodWrite    = "vfs.write"
	MethodMkdir    = "vfs.mkdir"
	MethodMove     = "vfs.move"
	MethodRemove   = "vfs.remove"
	MethodTouch    = "vfs.touch"
	MethodVolumes  = "vfs.volumes"
	MethodCreate   = "vfs.create"
	MethodSetQuota = "vfs.set_quota"
)
