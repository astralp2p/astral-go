package objects

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/astralp2p/astral-go/astral"
)

// sampleRepositoryInfos covers a leaf, a sequential group, a concurrent group,
// and an empty sequential group that must stay distinct from the leaf.
func sampleRepositoryInfos() map[string]RepositoryInfo {
	return map[string]RepositoryInfo{
		"leaf": {
			Name: "mem0", Label: "Default memory", Free: 67108864,
			Kind: RepositoryKindRepository, Children: []astral.String8{},
		},
		"sequential group": {
			Name: "main", Label: "World",
			Kind: RepositoryKindGroup, Children: []astral.String8{"device", "virtual", "network"},
		},
		"concurrent group": {
			Name: "local", Label: "Local storage",
			Kind: RepositoryKindGroup, Children: []astral.String8{"fs1", "fs0"}, Concurrent: true,
		},
		"empty sequential group": {
			Name: "memory", Label: "In-memory repos",
			Kind: RepositoryKindGroup, Children: []astral.String8{},
		},
	}
}

func TestRepositoryInfo_BinaryRoundTrip(t *testing.T) {
	for name, src := range sampleRepositoryInfos() {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := src.WriteTo(&buf); err != nil {
				t.Fatal(err)
			}

			var dst RepositoryInfo
			if _, err := dst.ReadFrom(&buf); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(src, dst) {
				t.Fatalf("want %+v, got %+v", src, dst)
			}
		})
	}
}

func TestRepositoryInfo_JSONRoundTrip(t *testing.T) {
	for name, src := range sampleRepositoryInfos() {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(src)
			if err != nil {
				t.Fatal(err)
			}

			var dst RepositoryInfo
			if err := json.Unmarshal(data, &dst); err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(src, dst) {
				t.Fatalf("want %+v, got %+v from %s", src, dst, data)
			}
		})
	}
}

// The field order and value forms are the agreed objects.repositories shape.
func TestRepositoryInfo_MarshalJSON_Shape(t *testing.T) {
	info := sampleRepositoryInfos()["sequential group"]

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}

	want := `{"Name":"main","Label":"World","Free":0,"Kind":"group","Children":["device","virtual","network"],"Concurrent":false}`
	if string(data) != want {
		t.Fatalf("want %s, got %s", want, data)
	}
}

func TestRepositoryInfo_MarshalJSON_NilChildrenIsEmptyArray(t *testing.T) {
	data, err := json.Marshal(RepositoryInfo{Name: "mem0", Kind: RepositoryKindRepository})
	if err != nil {
		t.Fatal(err)
	}

	want := `{"Name":"mem0","Label":"","Free":0,"Kind":"repository","Children":[],"Concurrent":false}`
	if string(data) != want {
		t.Fatalf("want %s, got %s", want, data)
	}
}
