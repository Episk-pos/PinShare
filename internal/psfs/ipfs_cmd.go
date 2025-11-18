package psfs

import (
	"context"
	"fmt"
	"os"
	"sync"

	shell "github.com/ipfs/go-ipfs-api"
)

var (
	ipfsClient *shell.Shell
	clientOnce sync.Once
	clientErr  error
)

// getIPFSClient returns a singleton IPFS HTTP API client
func getIPFSClient() (*shell.Shell, error) {
	clientOnce.Do(func() {
		apiURL := os.Getenv("IPFS_API")
		if apiURL == "" {
			apiURL = "http://localhost:5001"
		}

		// Create shell client
		ipfsClient = shell.NewShell(apiURL)
		if ipfsClient == nil {
			clientErr = fmt.Errorf("failed to create IPFS client for %s", apiURL)
		}
	})
	return ipfsClient, clientErr
}

// execute this cmd ipfs add  --cid-version 1 --raw-leaves gt256kb.txt -Q
func AddFileIPFS(ctx context.Context, filePath string) (string, error) {
	client, err := getIPFSClient()
	if err != nil {
		return "", err
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Add file to IPFS with options: CIDv1 and raw leaves
	cid, err := client.Add(file, shell.CidVersion(1), shell.RawLeaves(true))
	if err != nil {
		return "", fmt.Errorf("ipfs add failed: %w", err)
	}

	return cid, nil
}

func GetFileIPFS(cidStr string, filepath string) error {
	client, err := getIPFSClient()
	if err != nil {
		return err
	}

	// Get the file from IPFS
	err = client.Get(cidStr, filepath)
	if err != nil {
		return fmt.Errorf("ipfs get failed: %w", err)
	}

	return nil
}

func PinFileIPFS(cidStr string) error {
	client, err := getIPFSClient()
	if err != nil {
		return err
	}

	// Pin the file
	err = client.Pin(cidStr)
	if err != nil {
		return fmt.Errorf("ipfs pin add failed: %w", err)
	}

	return nil
}

func UnpinFileIPFS(cidStr string) error {
	client, err := getIPFSClient()
	if err != nil {
		return err
	}

	// Unpin the file
	err = client.Unpin(cidStr)
	if err != nil {
		return fmt.Errorf("ipfs pin rm failed: %w", err)
	}

	return nil
}
