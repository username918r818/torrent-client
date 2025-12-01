package file

import (
	"context"
	"log/slog"
	"os"
)

type ATask struct {
	Callback chan<- AReport
	Path     string
	Length   int64
}

type WTask struct {
	Callback chan<- WReport
	Id       any
	File     *os.File
	Offset   int64
	Length   int64
	Data     []byte
}
type DTask struct {
	Callback chan<- DReport
	Id       any
	File     *os.File
}

type AReport struct {
	Ok   bool
	Path string
}
type WReport struct {
	Ok bool
	Id any
}
type DReport struct {
	Ok bool
	Id any
}

type FileChannels struct {
	Ready <-chan struct{}
	ATask chan<- ATask
	WTask chan<- WTask
	DTask chan<- DTask
}

type channels struct {
	ready chan<- struct{}
	aTask <-chan ATask
	wTask <-chan WTask
	dTask <-chan DTask
}

func Init(ctx context.Context, n int) FileChannels {
	r := make(chan struct{}, n)
	a := make(chan ATask)
	w := make(chan WTask)
	d := make(chan DTask)

	outer := FileChannels{}
	inner := channels{}

	outer.Ready = r
	outer.ATask = a
	outer.WTask = w
	outer.DTask = d

	inner.ready = r
	inner.aTask = a
	inner.wTask = w
	inner.dTask = d

	start(ctx, inner)

	return outer
}

func start(ctx context.Context, ch channels) {
	ch.ready <- struct{}{}
	for {
		select {
		case msg := <-ch.wTask:
			slog.Debug("file worker: received wTask")

			err := writeChunk(msg.File, msg.Offset, msg.Data)
			r := WReport{}
			r.Id = msg.Id
			if err != nil {
				slog.Error("file Worker: " + err.Error())
				r.Ok = false
				msg.Callback <- r
			} else {
				r.Ok = true
				msg.Callback <- r
			}

			ch.ready <- struct{}{}
		case <-ctx.Done():
			return
		}
	}

}
