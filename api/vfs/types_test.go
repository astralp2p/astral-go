package vfs

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func TestEntryRoundTrip(t *testing.T) {
	in := Entry{Name: "sub", ID: fileID("x"), Flags: FlagFolder, ModTime: vectorTime}

	var buf bytes.Buffer
	if _, err := in.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	var out Entry
	if _, err := out.ReadFrom(&buf); err != nil {
		t.Fatal(err)
	}
	if out.Name != in.Name || !out.ID.IsEqual(&in.ID) || out.Flags != in.Flags ||
		!out.ModTime.Time().Equal(in.ModTime.Time()) || !out.IsFolder() {
		t.Fatalf("round trip changed the entry: %+v", out)
	}

	if _, err := json.Marshal(in); err != nil {
		t.Fatal(err)
	}
}

func TestInfoRoundTrip(t *testing.T) {
	in := Info{VFS: astral.GenerateIdentity(), Root: fileID("r"), Usage: 1, Quota: 2, Entries: 3, EntryLimit: 4}

	var buf bytes.Buffer
	if _, err := in.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	var out Info
	if _, err := out.ReadFrom(&buf); err != nil {
		t.Fatal(err)
	}
	if !out.VFS.IsEqual(in.VFS) || !out.Root.IsEqual(&in.Root) ||
		out.Usage != 1 || out.Quota != 2 || out.Entries != 3 || out.EntryLimit != 4 {
		t.Fatalf("round trip changed the info: %+v", out)
	}
}

func TestTypesRegistered(t *testing.T) {
	for _, name := range []string{
		"mod.vfs.entry", "mod.vfs.info",
		"mod.vfs.read_action", "mod.vfs.write_action", "mod.vfs.manage_action",
	} {
		if astral.New(name) == nil {
			t.Errorf("%s is not registered", name)
		}
	}
}

func TestMethodNames(t *testing.T) {
	want := []string{
		"vfs.ls", "vfs.stat", "vfs.statfs", "vfs.read", "vfs.write", "vfs.mkdir",
		"vfs.move", "vfs.remove", "vfs.touch", "vfs.volumes", "vfs.create", "vfs.set_quota",
	}
	got := []string{
		MethodLs, MethodStat, MethodStatfs, MethodRead, MethodWrite, MethodMkdir,
		MethodMove, MethodRemove, MethodTouch, MethodVolumes, MethodCreate, MethodSetQuota,
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %q, want %q", got[i], want[i])
		}
	}
}
