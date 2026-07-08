// objects-store stores an object in the local node's "local" repository and reads it back.
// Requires a running astrald node and ASTRALD_APPHOST_TOKEN.
package main

import (
	"fmt"
	"io"

	"github.com/cryptopunkscc/astral-go/api/objects"
	objectsClient "github.com/cryptopunkscc/astral-go/api/objects/client"
	"github.com/cryptopunkscc/astral-go/lib/astrald"
)

func main() {
	ctx := astrald.NewContext()
	client := objectsClient.Default()

	// every objects.Writer must end with Commit() or Discard()
	w, err := client.Create(ctx, objects.RepoLocal, 0)
	if err != nil {
		panic(err)
	}

	if _, err = w.Write([]byte("hello, objects")); err != nil {
		w.Discard()
		panic(err)
	}

	objectID, err := w.Commit()
	if err != nil {
		panic(err)
	}

	// limit 0 reads the whole object
	r, err := client.Read(ctx, objectID, 0, 0)
	if err != nil {
		panic(err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%v: %s\n", objectID, data)
}
