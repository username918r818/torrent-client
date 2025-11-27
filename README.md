# Torrent Client

A concurrent BitTorrent client implementation in Go featuring an actor-model architecture for efficient peer-to-peer file downloading.

## Features

- Pipelined block requests (5 concurrent requests per peer)
- Parallel peer connections
- Parallel disk I/O with multiple file workers
- Parallel piece validation and packing
- Multi-file torrent support

## Usage

```bash
go run . <torrent-file>
```

## Architecture

### Supervisor

The central coordinator that manages the entire download process:

- **Peer Management**: Maintains up to N active peer connections, queues additional peers
- **Task Assignment**: Allocates download tasks to available peers based on their bitfields
- **State Tracking**: Monitors peer states (choking, downloading, waiting, dead)
- **Message Routing**: Distributes channels between workers

The supervisor uses a state machine to track each peer:

- `PeerNotFound`: Peer not yet connected
- `PeerCouldBeAdded`: Ready to connect
- `PeerDead`: Connection failed or closed
- `PeerChoking`: Peer is choking us
- `PeerDownloading`: Actively downloading
- `PeerWaiting`: Ready for new task

### Tracker Worker

Communicates with the BitTorrent tracker:

- Sends announce requests with download progress
- Receives peer lists from tracker
- Periodically polls tracker for new peers
- Reports download statistics (uploaded, downloaded, left)

### Piece Workers (3 workers)

Handle piece validation and disk write coordination:

- **Download Tracking**: Uses efficient range structures to track which bytes are downloaded or saved
- **Piece Validation**: Validates complete pieces using SHA-1 hashes
- **Write Queue Management**: Queues validated pieces for disk writes
- **Memory Management**: Frees piece buffers after successful disk writes
- **Multi-file Support**: Maps piece offsets to correct file positions

### Peer Workers (1-50 workers per peer)

Each peer worker manages a single peer connection:

- **Handshake**: Performs BitTorrent protocol handshake
- **Message Protocol**: Implements BitTorrent wire protocol (choke, unchoke, interested, have, bitfield, request, piece)
- **Pipelining**: Maintains 5 concurrent block requests per peer for optimal throughput
- **Block Requests**: Requests 16 KB blocks sequentially within assigned pieces
- **Timeout Handling**: 3-minute read/write timeouts with automatic reconnection
- **Keep-alive**: Handles keep-alive messages during idle periods

The peer worker uses a request queue to pipeline downloads:

```
requestQueue < 5 → Send next request
Receive piece → requestQueue--
```

### File Workers (5 workers)

Handle all disk I/O operations independently of torrent logic:

- **Parallel Writes**: 5 concurrent workers for high-speed disk writes
- **File Allocation**: Pre-allocates files with correct sizes
- **Multi-file Support**: Handles torrents with multiple files and directory structures
- **Callback System**: Reports write success/failure back to piece workers
- **Error Recovery**: Failed writes are automatically retried

The file workers are torrent-agnostic and can handle multiple torrents simultaneously, making them a shared resource pool.

## Protocol Details

### Block Size

- Standard block size: 16 KB (16384 bytes)
- Last block in piece may be smaller

### Piece Size

- Typically 256 KB to 1 MB (torrent-dependent)
- Each piece is validated with SHA-1 hash

### Connection Limits

- Maximum concurrent peers: N
- Pipelined requests per peer: 5
- Total potential throughput: 250 concurrent block downloads

## Performance Optimizations

1. **Pipelined Requests**: Each peer maintains 5 concurrent block requests
2. **Parallel Peers**: Up to N simultaneous peer connections
3. **Parallel Disk I/O**: 5 file workers for concurrent writes
4. **Efficient Range Tracking**: O(log n) insertion and lookup for downloaded ranges
5. **Memory Management**: Piece data freed immediately after disk write
