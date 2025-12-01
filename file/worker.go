package file

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/username918r818/torrent-client/message"
)

func StartFileWorker(ctx context.Context, ch message.FileChannels) {
	ch.ReadyChannel <- true
	for {
		select {
		case msg := <-ch.ToSaveChannel:
			if msg.Length == -1 {
				time.Sleep(time.Second * 10)
				ch.ReadyChannel <- true
				break
			}
			slog.Info("File worker: received msg: " + fmt.Sprintf("%d", msg.Length))
			data := make([]byte, msg.Length)
			var index int64
			pieceIndex := msg.Offset / msg.PieceLength
			startPiece := msg.Offset % msg.PieceLength
			copy(data, msg.Pieces[pieceIndex][startPiece:])
			index = msg.PieceLength - startPiece

			for index < msg.Length {
				pieceIndex++
				copy(data[index:], msg.Pieces[pieceIndex])
				index += msg.PieceLength
			}

			err := writeChunk(msg.File, msg.FileOffset, data)

			if err != nil {
				slog.Error("File Worker: " + err.Error())
				msg.Callback <- message.IsRangeSaved{}
			} else {
				msg.Callback <- message.IsRangeSaved{IsSaved: true, Offset: msg.Offset, Length: msg.Length}
			}

			ch.ReadyChannel <- true
		case <-ctx.Done():
			return
		}
	}

}
