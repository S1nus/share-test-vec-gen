package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	inc "github.com/celestiaorg/go-square/v2/inclusion"
	sh "github.com/celestiaorg/go-square/v2/share"
	"github.com/celestiaorg/nmt"
	"github.com/cometbft/cometbft/crypto/merkle"
)

/*
	The purpose of this program is to generate test vectors which we will use to ensure correctness
	of the Solidity and Rust libraries, against the go-square package reference implementation.

	These test vectors are designed to cover edge cases and potential issues such as off-by-1 errors.

	TODO: check the values of the shares, not just the lengths.
*/

type testVector struct {
	Namespace  string `json:"namespace"`
	Data       string `json:"data"`
	Shares     string `json:"shares"`
	Commitment string `json:"commitment"`
}

func main() {
	jsonFile, err := os.Open("testVectors.json")
	if err != nil {
		log.Fatalf("Failed to open file: %s", err)
	}
	defer jsonFile.Close()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("Failed to read file: %s", err)
	}

	var testVectors []testVector
	err = json.Unmarshal(byteValue, &testVectors)
	if err != nil {
		log.Fatalf("Failed to unmarshal json: %s", err)
	}
	for i, testVector := range testVectors[:1] {

		namespaceBytes, err := hex.DecodeString(testVector.Namespace)
		if err != nil {
			log.Fatalf("Failed to decode namespace: %s", err)
		}

		namespace, err := sh.NewNamespaceFromBytes(namespaceBytes)
		if err != nil {
			log.Fatalf("Failed to create namespace: %s", err)
		}

		dataBytes, err := hex.DecodeString(testVector.Data)
		if err != nil {
			log.Fatalf("Failed to decode data: %s", err)
		}

		blob, err := sh.NewBlob(namespace, dataBytes, 0, nil)
		if err != nil {
			log.Fatalf("Failed to create blob: %s", err)
		}

		shares, err := splitBlobs(blob)
		if err != nil {
			log.Fatalf("Failed to split blobs: %s", err)
		}
		if toHexString(shares) != testVector.Shares {
			fmt.Println("vec :", i, " shares do not match")
		} else {
			fmt.Println("vec :", i)
			fmt.Println("commitment: ", testVector.Commitment)
		}

		// defaultSubtreeRootThreshold is 64
		subtreeWidth := inc.SubTreeWidth(len(shares), 64)
		treeSizes, err := inc.MerkleMountainRangeSizes(uint64(len(shares)), uint64(subtreeWidth))
		if err != nil {
			log.Fatalf("Failed to get merkle mountain range sizes: %s", err)
		}
		leafSets := make([][][]byte, len(treeSizes))
		cursor := uint64(0)
		for i, treeSize := range treeSizes {
			leafSets[i] = sh.ToBytes(shares[cursor : cursor+treeSize])
			cursor += treeSize
		}

		// create the commitments by pushing each leaf set onto an NMT
		subTreeRoots := make([][]byte, len(leafSets))
		for i, set := range leafSets {
			// Create the NMT. TODO: use NMT wrapper.
			tree := nmt.New(sha256.New(), nmt.NamespaceIDSize(sh.NamespaceSize), nmt.IgnoreMaxNamespace(true))
			for _, leaf := range set {
				// the namespace must be added again here even though it is already
				// included in the leaf to ensure that the hash will match that of
				// the NMT wrapper (pkg/wrapper). Each namespace is added to keep
				// the namespace in the share, and therefore the parity data, while
				// also allowing for the manual addition of the parity namespace to
				// the parity data.
				nsLeaf := make([]byte, 0)
				nsLeaf = append(nsLeaf, namespace.Bytes()...)
				nsLeaf = append(nsLeaf, leaf...)

				err = tree.Push(nsLeaf)
				if err != nil {
					log.Fatalf("Failed to push leaf: %s", err)
				}
			}
			// add the root
			root, err := tree.Root()
			fmt.Println("the root: ", hex.EncodeToString(root))
			if err != nil {
				log.Fatalf("Failed to get root: %s", err)
			}
			subTreeRoots[i] = root
		}

		commitment := merkle.HashFromByteSlices(subTreeRoots)
		fmt.Println("got commitment: ", hex.EncodeToString(commitment))
		fmt.Println("expected commitment: ", testVector.Commitment)
	}

}

func toHexString(shares []sh.Share) string {
	var result string
	for _, share := range shares {
		result += fmt.Sprintf("%x", share.ToBytes())
	}
	return result
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
