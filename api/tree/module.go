/*
Package tree describes a module that adds a tree object store to the node.

Every node in the tree can hold an Object and can have named subnodes (both at the same time are allowed).
By default, all tree nodes are stored in the database. You can mount any Node at any valid path.

Paths begin with a slash and consist of segments separated by slashes, just like in a typical filesystem:

* /               - root node
* /path/to/a/node - a deeper node

Segments can contain any non-slash printable characters.

The default node implementation is a simple database store, but you can mount any implementation at any existing
path in the tree.
*/
package tree

const (
	MethodGet         = "tree.get"
	MethodSet         = "tree.set"
	MethodDelete      = "tree.delete"
	MethodList        = "tree.list"
	MethodMountRemote = "tree.mount_remote"
	MethodUnmount     = "tree.unmount"
)
