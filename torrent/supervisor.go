package torrent

import (
	"context"
	"log/slog"
	"sync"

	"github.com/username918r818/torrent-client/file"
	"github.com/username918r818/torrent-client/message"
)

func StartSupervisor(ctx context.Context, torrentFile TorrentFile, port int) {
	ch, traCh, peerCh, pieceCh, fileCh := message.GetChannels()

	trackerSession := &TrackerSession{}

	trackerSession.TorrentFile = &torrentFile
	trackerSession.Port = port
	trackerSession.Left = torrentFile.Files[0].Length

	var wgTracker, wgFiles, wgPiece, wgPeers sync.WaitGroup
	wgTracker.Go(func() { StartWorkerTracker(ctx, trackerSession, traCh) })

	for range 2 {
		wgFiles.Go(func() { file.StartPieceWorker(ctx, fileCh) })
	}

	var totalBytes int64

	fileMap, err := file.Alloc(torrentFile.Files)
	if err != nil {
		slog.ErrorContext(ctx, "Supervisor: "+err.Error())
		return
	}

	for _, v := range torrentFile.Files {
		totalBytes += v.Length

	}

	pieceFile := make(chan message.IsRangeSaved)
	pieceCh.FileWorkerIsSaved = pieceFile

	pieceArray := InitPieceArray(totalBytes, torrentFile.PieceLength)
	pieceCh.FileWorkerIsSaved = make(<-chan message.IsRangeSaved)

	for range 3 {
		wgFiles.Go(func() { StartPieceWorker(ctx, &pieceArray, &torrentFile, fileMap, pieceCh) })
	}

	_, _, _, _, _, _, _, _ = ch, traCh, peerCh, pieceCh, fileCh, &wgPiece, &wgPeers, fileMap
}
