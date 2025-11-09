## Summary
PinShare currently interacts with the local IPFS daemon using shell CLI commands (e.g., \`exec.Command(\"ipfs\", \"add\", filepath)\` in \`internal/psfs/ipfs_cmd.go\`). This works well for local/single-node setups but limits flexibility for remote IPFS nodes, distributed environments, or containerized deployments where the CLI binary might not be available or performant. Switching to the IPFS HTTP API (port 5001) would enable remote daemon support, better error handling (JSON responses), and eliminate shell dependencies.

This is a medium-priority enhancement to improve portability and align with IPFS best practices (e.g., using libraries like \`github.com/ipfs/go-ipfs-api\` for direct HTTP calls).

## Current State
- **CLI Usage**: PinShare shells out to the \`ipfs\` binary for key operations:
  - \`addFileIPFS\` and \`pinFileIPFS\` in \`internal/psfs/ipfs_cmd.go\` use \`exec.Command\` to run \`ipfs add\`, \`ipfs pin add\`, etc.
  - Output parsing is manual (e.g., grep for CID from stdout).
  - Banset routine (\`internal/app\`) uses \`ipfs pin rm\` via CLI.
- **Dependencies**: Relies on IPFS CLI in PATH (works in Docker image but brittle for custom setups).
- **Limitations**:
  - Local-only: Can't easily connect to remote IPFS nodes (e.g., public gateways or clusters).
  - Error Handling: Stderr parsing is error-prone (e.g., no structured JSON).
  - Performance: Shell overhead for each call in the watcher loop (\`internal/app/startFileWatcher\`).
  - Portability: Assumes CLI installed; fails if missing (e.g., minimal containers).

From code review (\`internal/psfs/ipfs_cmd.go\`):
\`\`\`go
cmd := exec.Command(\"ipfs\", \"add\", filepath)
output, err := cmd.Output()
if err != nil {
  return \"\", fmt.Errorf(\"ipfs add failed:
\`\`\`

## Proposed Solution
Replace CLI calls with HTTP API using \`github.com/ipfs/go-ipfs-api\`. This library provides typed methods for add/pin operations with JSON responses.

### Implementation Steps
1. **Add Dependency**:
   - \`go get github.com/ipfs/go-ipfs-api\`
   - Update \`go.mod\`

2. **Refactor \`internal/psfs/ipfs_cmd.go\`**:
   - Import: \`import ipfsapi \"github.com/ipfs/go-ipfs-api\"\`
   - Create client: \`client := ipfsapi.NewApi(\"http://localhost:5001\")\`
   - Replace \`exec.Command(\"ipfs\", \"add\", filepath)\` with:
     \`\`\`go
nd, err := client.Unixfs().AddFile(ctx, filepath)
if err != nil {
  return \"\", err
}
cid := nd.Cid().String()
\`\`\`
   - For pinning: \`client.Pin().Add(ctx, cid)\`
   - Handle errors with structured JSON (e.g., client errors are typed).

3. **Update Banset Routine** (\`internal/app/app.go\`):
   - Use \`client.Pin().Rm(ctx, cid)\` instead of CLI \`ipfs pin rm\`.

4. **Configurable Endpoint**:
   - Add \`ipfsApiUrl: \"http://localhost:5001\"` to \`config/config.yaml\`.
   - Load in app: \`client := ipfsapi.NewApi(config.IPFSApiUrl)\`
   - Default to local; env var for remote (e.g., public gateway).

5. **Error Handling**:
   - Wrap API errors (e.g., 5001 not ready → retry with backoff).
   - Log JSON responses for debugging.

6. **Testing**:
   - Unit: Mock client for add/pin.
   - Integration: Test with local IPFS daemon.
   - Remote: Config for public API (e.g., ipfs.infura.io).

### Benefits
- **Remote Support**: Connect to any IPFS node/cluster (e.g., Filecoin, Pinata).
- **Structured Data**: JSON CIDs/errors (no parsing stdout).
- **Performance**: No shell overhead; direct HTTP.
- **Portability**: No CLI dep; works in minimal envs.

### Risks/Migration
- **Dependency**: Add go-ipfs-api (lightweight, ~1MB).
- **Local Daemon**: Still needs IPFS running (CLI or API).
- **Fallback**: Keep CLI as optional for legacy.

### Related
- Code: \`internal/psfs/ipfs_cmd.go\`
- TODO: Remote IPFS support (per \`docs/planning/todo.md\`).

Assign to @bryanchriswhite? Label: enhancement, help wanted.