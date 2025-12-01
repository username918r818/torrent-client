package piece

import (
	// "context"
	"context"
	"crypto/sha1"
	"os"
	"sync"
	"sync/atomic"

	// "errors"
	// "log/slog"
	// "os"
	// "strings"
	// "sync"
	// "github.com/username918r818/torrent-client/message"
	"github.com/username918r818/torrent-client/file"
	"github.com/username918r818/torrent-client/util"
)

// type PieceState int

// const (
// 	NotStarted PieceState = iota
// 	InProgress
// 	Downloaded
// 	Validated
// 	Saving
// 	Saved
// 	Corrupted
// )

// type Piece struct {
// 	state      PieceState
// 	data       []byte
// 	downloaded *util.List[util.Pair[int64, int64]] // downloaded ranges in byte
// }

// type PieceArray struct {
// 	pieces          []Piece
// 	pieceLength     int64
// 	lastPieceLength int64
// 	dataLocks       []sync.Mutex                        // for writing into pieces
// 	listDLocks      []sync.Mutex                        // locks for downloaded
// 	listTLock       sync.Mutex                          // locks for toSave
// 	toSave          *util.List[util.Pair[int64, int64]] // used to know ranges in piece indicies of downloaded but not saved yet data
// 	listSLock       sync.Mutex                          // locks for Saved
// 	Saved           *util.List[util.Pair[int64, int64]] // used to know ranges of saved data
// }

// var ErrValidatedPiece = errors.New("piece already validated")

type Piece struct {
	Index int64
	Data  []byte
}

type StatDiff struct {
	ToDownload int64
	Validated  int64
	Saving     int64
	Saved      int64
}

type toDeleteEnum = int

const (
	toDelete toDeleteEnum = iota
	deleting
	toRestore
)

type pieceSession struct {
	listDlock  sync.Mutex // lock for downloaded
	listTSLock sync.Mutex // lock for toSave
	listSLock  sync.Mutex // lock for saving
	listSDLock sync.Mutex // lock for saved

	downloaded, toSave, saving, saved util.RangeSet

	pieceMutex   sync.RWMutex       // lock for pieceMutexes
	pieceMutexes map[int]sync.Mutex // need pieceMutex be acquired for use // TODO check if need
	pieces       map[int][]byte

	pieceLength int64

	readyFileWorkers atomic.Int64

	fileMutex       sync.Mutex
	fileRanges      util.RangeSet
	rangeFile       map[util.Range]string
	fileRange       map[string]util.Range
	filePath        map[string][]string // clear after alloc
	filePointer     map[string]*os.File
	filesToAllocate map[string]struct{}
	filesToDelete   map[string]toDeleteEnum
	ch              Channels
	cb              cbChannels
}

type Channels struct {
	FileReady   <-chan struct{}
	Save        chan<- file.WTask
	Delete      chan<- file.DTask
	Alloc       chan<- file.ATask
	Piece       <-chan Piece
	PieceReport chan<- util.Pair[int, bool]
	Report      chan<- StatDiff
	ToCreate    <-chan map[string]util.Range
	ToDelete    <-chan map[string]util.Range
	ToDownload  chan<- []int
	NewFile     <-chan NewFileTask
	DeleteFile  <-chan DeleteFileTask
	saved       <-chan file.WReport
	deleted     <-chan file.DReport
	allocated   <-chan file.AReport
}

type cbChannels struct {
	saved     chan<- file.WReport
	deleted   chan<- file.DReport
	allocated chan<- file.AReport
}

type NewFileTask struct {
	Path           []string
	Offset, Length int64
}

type DeleteFileTask struct {
	SlicePath  []string
	StringPath string
}

func Validate(data []byte, hash [20]byte) bool {
	return sha1.Sum(data) == hash
}

func Init(ctx context.Context, amount int, ch Channels, pieceLength int64) {
	ps := pieceSession{}
	sc := make(chan file.WReport)
	dc := make(chan file.DReport)
	ac := make(chan file.AReport)
	ch.saved = sc
	ch.deleted = dc
	ch.allocated = ac
	ps.cb.saved = sc
	ps.cb.deleted = dc
	ps.cb.allocated = ac

	ps.pieceLength = pieceLength

	helperReportCh := make(chan StatDiff)
	helperToDownloadCh := make(chan []int)

	go helperStats(ctx, helperReportCh, ch.Report)
	go helperToDownload(ctx, helperToDownloadCh, ch.ToDownload)

	for range amount {
		go ps.start(ctx)
	}
}

// TODO
func (ps *pieceSession) start(ctx context.Context) {}

// sends to file worker, removes from toSave, adds to saving
// need to be locked fileRange, osFile
func (ps *pieceSession) save(r util.Range, fs string) {
	fr := ps.fileRange[fs]
	osFile := ps.filePointer[fs]
	task := file.WTask{Callback: ps.cb.saved, Id: r, File: osFile}

	task.Offset = r.First - fr.First
	task.Length = r.Second - r.First
	task.Data = make([]byte, task.Length)

	ps.pieceMutex.RLock()
	curPiece := r.First / ps.pieceLength
	curIndex := r.First % ps.pieceLength
	dataIndex := int64(0)

	for dataIndex < task.Length {
		copy(task.Data[dataIndex:], ps.pieces[int(curPiece)][curIndex:])
		dataIndex += ps.pieceLength - curIndex
		curIndex = 0
		curPiece++
	}

	ps.ch.Save <- task
}

// sets range saved and remove Downloaded and saving, checks if need to delete
func (ps *pieceSession) setSaved(report file.WReport) {
	r, ok := report.Id.(util.Range)
	if !ok {
		panic("piece worker: file.WReport.Id: wrong type ")
	}
	ps.listDlock.Lock()
	ps.listSLock.Lock()
	ps.listSDLock.Lock()
	ps.downloaded.Extract(r)
	ps.saving.Extract(r)
	ps.saved.Insert(r)

	curPiece := r.First / ps.pieceLength
	if r.First%ps.pieceLength != 0 {
		curPiece++
	}

	ps.pieceMutex.Lock()

	for curPiece*ps.pieceLength <= r.Second {
		delete(ps.pieces, int(curPiece))
		curPiece++
	}
	ps.pieceMutex.Unlock()

	ps.fileMutex.Lock()

	// it should return file range
	fr := ps.fileRanges.FindIntersections(r)
	if len(fr) == 0 {
		panic("piece: setSeved(): len(fr) == 0")
	}
	f, ok := ps.rangeFile[fr[0]]
	if !ok {
		panic("piece: setSeved(): file not found")
	}

	if ps.filesToDelete[f] == toDelete {
		ps.fileMutex.Unlock()
		ps.tryToDelete()
	}
}

// sets range toSave, e.g. if file write fails or received a new file range to download
func (ps *pieceSession) setToSave(report file.WReport) {}

// receives index and adds it to Downloaded, also report
func (ps *pieceSession) validate(index int, piece, hash []byte) bool

// adds file and inserts ranges to toSave range, if deleting, set to restore, else put in queue to allocate, also toDownload(...)
func (ps *pieceSession) addFile(t NewFileTask) {}

// removes file and remove ranges from toSave range, if file is saving or saved, marks to delete
func (ps *pieceSession) removeFile(t DeleteFileTask) {}

// try to allocate
func (ps *pieceSession) tryToAllocate() {}

// allocs file
func (ps *pieceSession) alloc(f string) {}

// checks if it has ready file workers and filesToDelete not saving and saved and delete(file)
func (ps *pieceSession) tryToDelete() {}

// if file is not saving and saved then delete (and remove from saved range and set status deleting), else marks file to delete if saving
func (ps *pieceSession) delete(f string) {}

// checks if it has ready file workers and ranges and file is allocated then save(range)
func (ps *pieceSession) tryToSave()

// sends indicies of pieces to download
func (ps *pieceSession) toDownload(toDownload []int)

// creates helper stats foroutine
func helperStats(ctx context.Context, in <-chan StatDiff, out chan<- StatDiff) {
	cur := StatDiff{}
	for {
		select {
		case new := <-in:
			cur.ToDownload += new.ToDownload
			cur.Validated += new.Validated
			cur.Saving += new.Saving
			cur.Saved += new.Saved
		case out <- cur:
			cur = StatDiff{}
		case <-ctx.Done():
			return
		}
	}
}

// creates helper toDownload foroutine
func helperToDownload(ctx context.Context, in <-chan []int, out chan<- []int) {
	cur := make([]int, 0)
	unique := make(map[int]struct{})
	for {
		select {
		case new := <-in:
			for _, v := range new {
				if _, ok := unique[v]; !ok {
					cur = append(cur, v)
					unique[v] = struct{}{}
				}
			}
		case out <- cur:
			cur = make([]int, 0)
			unique = make(map[int]struct{})
		case <-ctx.Done():
			return
		}
	}
}

// func InitPieceArray(totalBytes, pieceLength int64) (a PieceArray) {
// 	arrLength := totalBytes / pieceLength
// 	a.lastPieceLength = totalBytes % pieceLength
// 	if a.lastPieceLength > 0 {
// 		arrLength++
// 	} else {
// 		a.lastPieceLength = pieceLength
// 	}
// 	a.pieces = make([]Piece, arrLength)
// 	a.dataLocks = make([]sync.Mutex, arrLength)
// 	a.listDLocks = make([]sync.Mutex, arrLength)
// 	a.pieceLength = pieceLength
// 	return
// }

// func UpdatePiece(pieceIndex int, a *PieceArray) ([]byte, error) {
// 	a.dataLocks[pieceIndex].Lock()
// 	defer a.dataLocks[pieceIndex].Unlock()
// 	if a.pieces[pieceIndex].state != NotStarted && a.pieces[pieceIndex].state != InProgress {
// 		return nil, ErrValidatedPiece
// 	}
// 	if a.pieces[pieceIndex].data == nil {
// 		a.pieces[pieceIndex].state = InProgress
// 		newLength := a.pieceLength
// 		if pieceIndex == len(a.pieces)-1 {
// 			newLength = a.lastPieceLength
// 		}
// 		a.pieces[pieceIndex].data = make([]byte, newLength)
// 		return a.pieces[pieceIndex].data, nil
// 	}

// 	return a.pieces[pieceIndex].data, nil
// }

// func StartPieceWorker(ctx context.Context, pieces *PieceArray, tf *TorrentFile, fileMap map[string]*os.File, ch message.PieceChannels) {
// 	for {
// 		select {
// 		case newBlock := <-ch.PeerHasDownloaded:
// 			// slog.Info("Piece worker: received new block")
// 			pieceIndex := newBlock.Offset / pieces.pieceLength
// 			pieceLowerBound := pieceIndex * pieces.pieceLength
// 			pieceUpperBound := pieceLowerBound + pieces.pieceLength
// 			if int(pieceIndex) == len(pieces.pieces)-1 {
// 				pieceUpperBound = pieceLowerBound + pieces.lastPieceLength
// 			}

// 			pieces.listDLocks[pieceIndex].Lock()
// 			pieces.pieces[pieceIndex].downloaded = util.InsertRange(pieces.pieces[pieceIndex].downloaded, newBlock.Offset, newBlock.Offset+newBlock.Length)
// 			checkRange := util.Contains(pieces.pieces[pieceIndex].downloaded, pieceLowerBound, pieceUpperBound)
// 			pieces.listDLocks[pieceIndex].Unlock()

// 			ns, sai := -newBlock.Length, newBlock.Length
// 			validated := false
// 			if checkRange {
// 				pieces.dataLocks[pieceIndex].Lock()
// 				if Validate(pieces.pieces[pieceIndex].data, tf.Pieces[pieceIndex]) {
// 					validated = true
// 					pieces.pieces[pieceIndex].state = Validated
// 					pieces.dataLocks[pieceIndex].Unlock()
// 					pieces.listTLock.Lock()
// 					pieces.toSave = util.InsertRange(pieces.toSave, pieceLowerBound, pieceUpperBound)
// 					pieces.listTLock.Unlock()
// 				} else {
// 					pieces.dataLocks[pieceIndex].Unlock()
// 					ns += pieceUpperBound - pieceLowerBound
// 				}
// 				sai += pieceLowerBound - pieceUpperBound
// 				pieces.listDLocks[pieceIndex].Lock()
// 				pieces.pieces[pieceIndex].downloaded = nil
// 				pieces.listDLocks[pieceIndex].Unlock()
// 			}
// 			msg := message.StatDiff{NotStarted: ns, Downloaded: sai}
// 			if validated {
// 				msg[Validated] = pieceUpperBound - pieceLowerBound
// 			}
// 			ch.PostStatsChannel <- msg

// 		case ready := (<-ch.FileWorkerReady):
// 			slog.Info("Piece worker: received new ready")
// 			if !ready {
// 				break
// 			}
// 			pieces.listTLock.Lock()
// 			firstRange := pieces.toSave
// 			if pieces.toSave != nil {
// 				pieces.toSave = pieces.toSave.Next
// 			}
// 			pieces.listTLock.Unlock()
// 			if firstRange == nil {
// 				msg := message.SaveRange{}
// 				msg.Length = -1
// 				ch.FileWorkerToSave <- msg
// 				break
// 			}
// 			totalOffset := firstRange.Value.First
// 			length := firstRange.Value.Second - firstRange.Value.First
// 			var f *os.File
// 			var fileStartOffset, fLength int64
// 			var currentOffset int64
// 			for _, curFile := range tf.Files {
// 				if currentOffset+curFile.Length > totalOffset {
// 					if len(curFile.Path) > 1 {
// 						f = fileMap[strings.Join(curFile.Path, "/")]
// 					} else {
// 						f = fileMap[curFile.Path[0]]
// 					}
// 					fileStartOffset = currentOffset
// 					fLength = curFile.Length
// 					break
// 				}
// 				currentOffset += curFile.Length
// 			}

// 			fileOffsetInFile := totalOffset - fileStartOffset
// 			remainingInFile := fLength - fileOffsetInFile
// 			if length > remainingInFile {
// 				length = remainingInFile
// 				pieces.listTLock.Lock()
// 				pieces.toSave = util.InsertRange(pieces.toSave, firstRange.Value.First+length, firstRange.Value.Second)
// 				pieces.listTLock.Unlock()
// 			}

// 			firstPiece := totalOffset / pieces.pieceLength
// 			lastPiece := (totalOffset + length - 1) / pieces.pieceLength

// 			dataToSend := make([][]byte, lastPiece+1)
// 			for i := firstPiece; i <= lastPiece; i++ {
// 				dataToSend[i] = pieces.pieces[i].data
// 			}

// 			msg := message.SaveRange{}
// 			msg.InfoHash = tf.InfoHash
// 			msg.Pieces = dataToSend
// 			msg.PieceLength = pieces.pieceLength
// 			msg.Offset = totalOffset
// 			msg.FileOffset = totalOffset - fileOffsetInFile
// 			msg.Length = length
// 			msg.File = f
// 			msg.Callback = ch.CallBack

// 			msgStats := message.StatDiff{Saving: length}
// 			ch.PostStatsChannel <- msgStats
// 			ch.FileWorkerToSave <- msg

// 		case isSaved, ok := (<-ch.FileWorkerIsSaved):
// 			slog.Info("Piece worker: received new saved")
// 			if !ok {
// 				break
// 			}
// 			msgStats := message.StatDiff{Saving: -isSaved.Length}
// 			if !isSaved.IsSaved {
// 				ch.PostStatsChannel <- msgStats
// 				pieces.listTLock.Lock()
// 				pieces.toSave = util.InsertRange(pieces.toSave, isSaved.Offset, isSaved.Offset+isSaved.Length)
// 				pieces.listTLock.Unlock()
// 				break
// 			}

// 			msgStats[Saved] += isSaved.Length
// 			ch.PostStatsChannel <- msgStats

// 			pieces.listSLock.Lock()
// 			pieces.Saved = util.InsertRange(pieces.Saved, isSaved.Offset, isSaved.Offset+isSaved.Length)
// 			pieces.listSLock.Unlock()

// 			firstPiece := isSaved.Offset / pieces.pieceLength
// 			lastPiece := (isSaved.Offset + isSaved.Length) / pieces.pieceLength

// 			for i := firstPiece; i <= lastPiece; i++ {
// 				if i == firstPiece || i == lastPiece {
// 					lw, up := i*pieces.pieceLength, (i+1)*pieces.pieceLength
// 					pieces.listSLock.Lock()
// 					checkRange := util.Contains(pieces.Saved, lw, up)
// 					pieces.listSLock.Unlock()
// 					if !checkRange {
// 						continue
// 					}
// 				}
// 				pieces.pieces[i].state = Saved
// 				pieces.pieces[i].data = nil
// 			}

// 		case <-ctx.Done():
// 			return
// 		}
// 	}

// }
