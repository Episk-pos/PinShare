# PinShare

PinShare is a Decentralised Pinning Service for IPFS, built on libp2p to assist the clustering of IPFS Content for curators building a library or knowledge stacks and as a basis to build advanced data pipelines from. It's a peer-to-peer application designed to securely share files by vetting them through The Security Consensus before their metadata is advertised to other peers on the network. PinShare can be customised to serve different community needs or data pools as seperate libraries, simply by changing the config and running seperate instances of PinShare. 

## Core Features

-   **P2P Networking**: Built on `go-libp2p` for robust and decentralized communication.
-   **Automated File Ingestion**: Watches a designated folder for new files to process automatically.
-   **Security First**: Integrates with VirusTotal to scan files before they are shared. It checks for existing reports by hash and can submit new files for analysis.
-   **File Type Validation**: A configurable allowlist ensures only permitted file types are processed.
-   **Metadata Propagation**: Uses libp2p PubSub to efficiently broadcast metadata of safe files to all connected peers.
-   **REST API**: Exposes an API for programmatic interaction, defined with OpenAPI.

## How It Works

The lifecycle of a file in PinShare follows these steps:

1.  **File Drop**: A user places a file into the configured `upload` directory.
2.  **Detection**: The application's file watcher detects the new file.
3.  **Validation**: The file's extension is checked against a list of allowed types. If not allowed, it's rejected.
4.  **Hashing**: A SHA256 hash of the file is computed.
5.  **VirusTotal Check**:
    -   PinShare first queries VirusTotal with the file's hash to check for a pre-existing scan report.
    -   If a report exists and shows zero detections, the file is considered safe.
    -   If no report exists, PinShare uses a headless browser (`chromedp`) to upload the file to VirusTotal for a new scan.
6.  **Verdict & Action**:
    -   **Safe**: If the file is cleared by VirusTotal (0 detections), its metadata is added to the local `metadata.json` store.
    -   **Unsafe/Rejected**: If the file is flagged as malicious or is an invalid type, it is moved to the `reject` folder.
7.  **Metadata Sharing**: Once a file is confirmed safe, its metadata is broadcast over the configured libp2p PubSub topic.
8.  **Peer Action**: Other peers in the network receive this metadata and can use it to fetch the file from the advertising peer.

## Getting Started

### Quickstart

- Preview Release is v0.1.2

- The docker image contains all dependancies required to run PinShare. 
- `docker run -it -v $(pwd):/opt/pinshare/data ghcr.io/cypherpunk-labs/pinshare:latest`

### Prerequisites

-   Go (latest version recommended)
-   Docker/Podman (latest version recommended)
-   IPFS Desktop (latest version recommended)
-   A local installation of Google Chrome or Chromium (required for VirusTotal integration).

### Installation

1.  Clone the repository:
    ```bash
    git clone https://github.com/cypherpunk-labs/PinShare.git
    cd PinShare
    ```

2.  Build the application:
    ```bash
    go build -o pinshare ./cmd/pinshare
    ```

### Configuration

PinShare is configured via `config/config.yaml`. You can copy the example file and modify it to suit your needs.

Key configuration settings:

-   `uploadFolder`: Directory to watch for new files (e.g., `./data/uploads`).
-   `cacheFolder`: Directory for temporary/cached files (e.g., `./data/cache`).
-   `rejectFolder`: Directory where unsafe or invalid files are moved (e.g., `./data/rejects`).
-   `metaDataFile`: Path to the JSON file storing metadata of safe files (e.g., `./data/metadata.json`).
-   `identityKeyFile`: Path to store the libp2p node's private key (e.g., `./data/identity.key`).
-   `libp2pPort`: Port for the libp2p host to listen on.
-   `metadataTopicID`: The PubSub topic name for sharing metadata.

### Running PinShare
 
#### Docker (Recommended)
 
The Docker image bundles IPFS, so everything runs in one container. For minimal access (IPFS Web UI and Gateway), expose only essential ports:

```bash
docker run -d --name pinshare \
  -v $(pwd):/opt/pinshare/data \
  -p 5001:5001 \
  -p 8080:8080 \
  ghcr.io/cypherpunk-labs/pinshare:latest
```
 
This starts the IPFS daemon internally, waits for it to initialize, then launches PinShare. Access:
- IPFS Web UI: http://localhost:5001/webui
- IPFS Gateway: http://localhost:8080 (for content previews)

Optional ports (uncomment for advanced use):
- `-p 9090:9090` for PinShare API (e.g., custom tools).
- `-p 4001:4001 -p 4001:4001/udp` for IPFS Swarm (P2P connectivity).
- `-p 50001:50001` for libp2p (metadata gossip).

For interactive mode (foreground logs), add `-it` and optional ports as needed.

To run just the IPFS daemon from the same image (e.g., for separate management or testing):
 
```bash
docker run -d --name ipfs-daemon \
  -v $(pwd)/ipfs-data:/data/ipfs \
  -p 5001:5001 \
  -p 8080:8080 \
  ghcr.io/cypherpunk-labs/pinshare:latest \
  /usr/local/bin/start_ipfs daemon
```

This exposes the IPFS API (5001 with Web UI at http://localhost:5001/webui) and gateway (8080). Optional: Add `-p 4001:4001 -p 4001:4001/udp` for Swarm. Note: PinShare expects IPFS to be local; for a separate PinShare container, use host networking (`--network host`) or mount the IPFS API socket.

#### Docker Compose (Full Stack with UI)
 
For a complete setup including the React UI, use docker-compose.yml (includes PinShare + built UI):

```bash
docker-compose up -d --build
```

This starts:
- PinShare with IPFS (ports 5001, 8080).

Access:
- IPFS Web UI: http://localhost:5001/webui
- IPFS Gateway: http://localhost:8080

Optional: Uncomment P2P ports in compose for network connectivity. Logs: `docker-compose logs -f`.

#### Non-Docker
 
For non-Docker runs, first ensure the IPFS daemon is running (PinShare interacts with it via CLI for file adding and pinning). You can use the Docker command above to host IPFS in a container, or install/run IPFS natively:
 
```bash
ipfs daemon &
```
 
This starts the IPFS node on port 5001. If using IPFS Desktop, ensure it's running and the daemon is active.
 
To start the service, simply run the compiled binary:

```bash
go build -o pinshare .
./pinshare
```

This will start the libp2p node, initialize the file watcher, and launch the API server. You will see logs indicating the node's status and peer connections.

## Command-Line Interface (CLI)

PinShare includes subcommands for specific tasks and debugging.

-   Run `./pinshare --help` for a full list of available commands.

-   **Example**: Test the headless browser integration by checking a known hash on VirusTotal.
    ```bash
    # The hash must exist on VirusTotal for this to work
    ./pinshare testcdp 275a021bbfb6489e54d471899f7db9d1663fc695ec2fe2a2c4538aabf651fd0f
    ```

## API

The service exposes a RESTful API for management and queries. The API is defined using the OpenAPI specification.

-   **API Specification**: See `docs/spec/basemetadata.openapi.spec.yaml` for the full contract.
-   The API server starts automatically when you run the main application.

## Security Considerations

The integration with VirusTotal currently relies on **web scraping** using `chromedp`. This approach is inherently fragile and may break if VirusTotal changes its website's HTML structure or selectors. This is a known risk and a more robust API-based integration is a future goal.
