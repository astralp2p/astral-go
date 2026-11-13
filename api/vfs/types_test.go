package vfs

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

func TestEntryRoundTrip(t *testing.T) {
	in := Entry{Name: "sub", ID: fileID("x"), Flags: FlagFolder, ModTime: vectorTime, CreationTime: vectorCreationTime}

	var buf bytes.Buffer
	if _, err := in.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	var out Entry
	if _, err := out.ReadFrom(&buf); err != nil {
		t.Fatal(err)
	}
	if out.Name != in.Name || !out.ID.IsEqual(&in.ID) || out.Flags != in.Flags ||
		!out.ModTime.Time().Equal(in.ModTime.Time()) ||
		!out.CreationTime.Time().Equal(in.CreationTime.Time()) || !out.IsFolder() {
		t.Fatalf("round trip changed the entry: %+v", out)
	}

	if _, err := json.Marshal(in); err != nil {
		t.Fatal(err)
	}
}

func TestInfoRoundTrip(t *testing.T) {
	in := Info{
		VFS: astral.GenerateIdentity(), Root: fileID("r"), Usage: 1, Quota: 2, Inodes: 3, InodeLimit: 4,
		Evaluator: "astral://app:vfs.authorize",
	}

	var buf bytes.Buffer
	if _, err := in.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	var out Info
	if _, err := out.ReadFrom(&buf); err != nil {
		t.Fatal(err)
	}
	if !out.VFS.IsEqual(in.VFS) || !out.Root.IsEqual(&in.Root) ||
		out.Usage != 1 || out.Quota != 2 || out.Inodes != 3 || out.InodeLimit != 4 ||
		out.Evaluator != in.Evaluator {
		t.Fatalf("round trip changed the info: %+v", out)
	}

	j, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var fromJSON Info
	if err := json.Unmarshal(j, &fromJSON); err != nil {
		t.Fatal(err)
	}
	if fromJSON.Evaluator != in.Evaluator {
		t.Fatalf("json round trip changed the evaluator: %q", fromJSON.Evaluator)
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
		"vfs.set_evaluator", "vfs.authorize",
	}
	got := []string{
		MethodLs, MethodStat, MethodStatfs, MethodRead, MethodWrite, MethodMkdir,
		MethodMove, MethodRemove, MethodTouch, MethodVolumes, MethodCreate, MethodSetQuota,
		MethodSetEvaluator, MethodAuthorize,
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got %q, want %q", got[i], want[i])
		}
	}
}
