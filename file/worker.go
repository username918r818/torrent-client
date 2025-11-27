package file

import (
	"context"
	"log/slog"

	"github.com/username918r818/torrent-client/message"
)

func StartFileWorker(ctx context.Context, ch message.FileChannels) {
	for {
		select {
		case msg := <-ch.ToSaveChannel: // TODO fix offset
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

			err := WriteChunk(msg.File, msg.FileOffset, data)

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
