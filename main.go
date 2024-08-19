package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	inc "github.com/celestiaorg/go-square/v2/inclusion"
	sh "github.com/celestiaorg/go-square/v2/share"
	"github.com/cometbft/cometbft/crypto/merkle"
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
	// Create a random namespace
	randomNamespaceId := make([]byte, 28)
	for i := 0; i < 18; i++ {
		randomNamespaceId[i] = 0
	}
	rand.Read(randomNamespaceId[18:])
	fmt.Println("Namespace ID:", randomNamespaceId)
	randomNamespace := sh.MustNewNamespace(0, randomNamespaceId)

	// Lowest application namespace
	// Namespaces below this value are not valid for application usage
	// 0x0000000000000000000000000000000000000000000000000000000100
	lowestNsBytes, err := hex.DecodeString("00000000000000000000000000000000000000000000000000000100")
	if err != nil {
		fmt.Println("Error decoding hex string:", err)
		return
	}
	lowestNamespace := sh.MustNewNamespace(0, lowestNsBytes)
	fmt.Println("Zero Namespace:", lowestNamespace)

	vecs := []testVector{}

	for _, n := range []sh.Namespace{randomNamespace, lowestNamespace} {
		// We choose lengths 478 and 479 because that is the boundary between needing 1 or 2 shares.
		zeroes478 := make([]byte, 478)
		for i := range zeroes478 {
			zeroes478[i] = 0
		}
		blob, _ := sh.NewBlob(n, zeroes478, 0, nil)
		cmmitment := inc.CreateCommitment(blob, merkle.HashFromByteSlices)
		shares, _ := splitBlobs(blob)
		out := ""
		for _, share := range shares {
			out = out + fmt.Sprintf("%x", share.ToBytes())
		}
		vecs = append(vecs, testVector{
			Namespace: fmt.Sprintf("%x", n.Bytes()),
			Data:      fmt.Sprintf("%x", zeroes478),
			Shares:    toHexString(shares),
		})

		zeroes479 := make([]byte, 479)
		for i := range zeroes479 {
			zeroes479[i] = 0
		}
		blob2, _ := sh.NewBlob(n, zeroes479, 0, nil)
		shares2, _ := splitBlobs(blob2)
		vecs = append(vecs, testVector{
			Namespace: fmt.Sprintf("%x", n.Bytes()),
			Data:      fmt.Sprintf("%x", zeroes479),
			Shares:    toHexString(shares2),
		})

		// We try prefix and suffix 1 values, to catch any weirdness that might occur at the boundary.
		prefix1_478 := make([]byte, 478)
		prefix1_478[0] = 1
		for i := range prefix1_478[1:] {
			prefix1_478[i] = 0
		}
		blob3, _ := sh.NewBlob(n, prefix1_478, 0, nil)
		shares3, _ := splitBlobs(blob3)
		vecs = append(vecs, testVector{
			Namespace: fmt.Sprintf("%x", n.Bytes()),
			Data:      fmt.Sprintf("%x", prefix1_478),
			Shares:    toHexString(shares3),
		})

		suffix1_478 := make([]byte, 478)
		for i := range suffix1_478[:477] {
			suffix1_478[i] = 0
		}
		suffix1_478[477] = 1
		blob4, _ := sh.NewBlob(n, suffix1_478, 0, nil)
		shares4, _ := splitBlobs(blob4)
		vecs = append(vecs, testVector{
			Namespace: fmt.Sprintf("%x", n.Bytes()),
			Data:      fmt.Sprintf("%x", suffix1_478),
			Shares:    toHexString(shares4),
		})

		suffix1_479 := make([]byte, 479)
		for i := range suffix1_479[:478] {
			suffix1_479[i] = 0
		}
		suffix1_479[478] = 1
		blob5, _ := sh.NewBlob(n, suffix1_479, 0, nil)
		shares5, _ := splitBlobs(blob5)
		vecs = append(vecs, testVector{
			Namespace: fmt.Sprintf("%x", n.Bytes()),
			Data:      fmt.Sprintf("%x", suffix1_479),
			Shares:    toHexString(shares5),
		})

		prefix1_479 := make([]byte, 479)
		prefix1_479[0] = 1
		for i := range prefix1_479[1:] {
			prefix1_479[i] = 0
		}
		blob6, _ := sh.NewBlob(n, prefix1_479, 0, nil)
		shares6, _ := splitBlobs(blob6)
		vecs = append(vecs, testVector{
			Namespace: fmt.Sprintf("%x", n.Bytes()),
			Data:      fmt.Sprintf("%x", prefix1_479),
			Shares:    toHexString(shares6),
		})

		randomKilobyte := make([]byte, 1024)
		rand.Read(randomKilobyte)
		blob7, _ := sh.NewBlob(n, randomKilobyte, 0, nil)
		shares7, _ := splitBlobs(blob7)
		vecs = append(vecs, testVector{
			Namespace: fmt.Sprintf("%x", n.Bytes()),
			Data:      fmt.Sprintf("%x", randomKilobyte),
			Shares:    toHexString(shares7),
		})

		random10kb := make([]byte, 1024*10)
		rand.Read(random10kb)
		blob8, _ := sh.NewBlob(n, random10kb, 0, nil)
		shares8, _ := splitBlobs(blob8)
		vecs = append(vecs, testVector{
			Namespace: fmt.Sprintf("%x", n.Bytes()),
			Data:      fmt.Sprintf("%x", random10kb),
			Shares:    toHexString(shares8),
		})

	}

	jsonData, _ := json.Marshal(vecs)
	file, _ := os.Create("testVectors.json")
	defer file.Close()
	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
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
