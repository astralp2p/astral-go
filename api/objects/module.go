package objects

import (
	"github.com/cryptopunkscc/astral-go/astral"
)

const (
	ModuleName   = "objects"
	MethodCreate = "objects.create"

	MethodNew               = "objects.new"
	MethodLoad              = "objects.load"
	MethodStore             = "objects.store"
	MethodDelete            = "objects.delete"
	MethodPurge             = "objects.purge"
	MethodContains          = "objects.contains"
	MethodScan              = "objects.scan"
	MethodSearch            = "objects.search"
	MethodDescribe          = "objects.describe"
	MethodFind              = "objects.find"
	MethodRegisterSearcher  = "objects.register_searcher"
	MethodRegisterDescriber = "objects.register_describer"
	MethodRegisterFinder    = "objects.register_finder"
	MethodRegisterBlueprint = "objects.register_blueprint"
	MethodProbe             = "objects.probe"
	MethodRead              = "objects.read"
	MethodGetType           = "objects.get_type"
	MethodPush              = "objects.push"
	MethodNewMem            = "objects.new_mem"
	MethodRepositories      = "objects.repositories"
	MethodRemoveRepository  = "objects.remove_repository"
	MethodBlueprints        = "objects.blueprints"
	MethodEcho              = "objects.echo"

	RepoMain      = "main"      // everything
	RepoDevice    = "device"    // device: memory, local, removable
	RepoMemory    = "memory"    // memcache repos
	RepoLocal     = "local"     // local storage
	RepoRemovable = "removable" // removable storage
	RepoVirtual   = "virtual"   // virtual repos (archives, encryption, chunks)
	RepoNetwork   = "network"   // network repos
	RepoSystem    = "system"
)

type Describer interface {
	DescribeObject(*astral.Context, *astral.ObjectID) (<-chan *Descriptor, error)
}
