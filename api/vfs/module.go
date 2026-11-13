// Package vfs holds the wire surface of the vfs module: the entry and info
// types, the three actions, the error values, the operation names and the
// canonical directory encoder (VFS 6, sections 4, 5, 7 and 8).
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

	MethodSetEvaluator = "vfs.set_evaluator"

	// MethodAuthorize is the op an evaluating app serves, not the node. The
	// app reads one mod.vfs.read_action or mod.vfs.write_action and answers
	// ack to allow; any other answer refuses.
	MethodAuthorize = "vfs.authorize"
)
