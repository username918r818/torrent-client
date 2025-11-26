package file

import (
	"context"

	"github.com/username918r818/torrent-client/message"
)

func StartPieceWorker(ctx context.Context, ch message.FileChannels) {
	go func() {
		for {
			select {
			case rangeToSave, ok := (<-ch.ToSaveChannel): // TODO
				_, _ = rangeToSave, ok
			case <-ctx.Done():
				return
			}
		}
	}()
}
