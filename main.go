package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	inc "github.com/celestiaorg/go-square/v2/inclusion"
	sh "github.com/celestiaorg/go-square/v2/share"
)

/*
	The purpose of this program is to generate test vectors which we will use to ensure correctness
	of the Solidity and Rust libraries, against the go-square package reference implementation.

	These test vectors are designed to cover edge cases and potential issues such as off-by-1 errors.

	TODO: check the values of the shares, not just the lengths.
*/

type testVector struct {
	Namespace string `json:"namespace"`
	Data      string `json:"data"`
	Shares    string `json:"shares"`
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
	for i, testVector := range testVectors {
		//namespaceBytes, err := hex.DecodeString(testVector.Namespace)
		//if err != nil {
		//	log.Fatalf("Failed to decode namespace: %s", err)
		//}
		//namespace := sh.MustNewV0Namespace(namespaceBytes)
		fmt.Println("vec :", i)
		sharesData, err := hex.DecodeString(testVector.Shares)
		if err != nil {
			log.Fatalf("Failed to decode shares: %s", err)
		}
		subtreeWidth := inc.SubTreeWidth(len(sharesData)/512, 64)
		//fmt.Println("Subtree width: ", subtreeWidth)
		treeSizes, err := inc.MerkleMountainRangeSizes(uint64(len(sharesData)/512), uint64(subtreeWidth))
		if err != nil {
			log.Fatalf("Failed to get tree sizes: %s", err)
		}
		fmt.Println("len tree sizes: ", len(treeSizes))
		for i, treeSize := range treeSizes {
			fmt.Println("Tree size: ", i, " size: ", treeSize)
		}
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
