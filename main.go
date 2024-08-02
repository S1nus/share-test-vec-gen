package main

import (
	"fmt"

	sh "github.com/celestiaorg/go-square/v2/share"
)

func main() {
	zeroes472 := make([]byte, 472)
	for i := range zeroes472 {
		zeroes472[i] = 0
	}
	namespace, _ := sh.NewNamespace(0, []byte{1, 2, 3, 4, 5})
	blob, _ := sh.NewBlob(namespace, zeroes472, 0, nil)
	shares, _ := splitBlobs(blob)
	fmt.Println(shares)
}

// splitBlobs splits the provided blobs into shares.
func splitBlobs(blobs ...*sh.Blob) ([]sh.Share, error) {
	writer := sh.NewSparseShareSplitter()
	for _, blob := range blobs {
		if err := writer.Write(blob); err != nil {
			return nil, err
		}
	}
	return writer.Export(), nil
}
