package message

type SupervisorChannels struct {
	ToPeerWorkerToDownload map[[6]byte]chan<- DownloadRange
	FromPeerWorker         <-chan PeerMessage
	GetPeers               <-chan Peers
}

type TrackerChannels struct {
	SendPeers       chan<- Peers
	GetStatsChannel <-chan Stats
}

type PeerChannels struct {
	ToDownload         <-chan DownloadRange // need initialize with new peer
	PeerMessageChannel chan<- PeerMessage
	DownloadedChannel  chan<- Block
}

type PieceChannels struct {
	PostStatsChannel  chan<- Stats
	PeerHasDownloaded <-chan Block
	FileWorkerReady   <-chan Ready
	FileWorkerIsSaved <-chan IsRangeSaved // need initialize with new torrent
	FileWorkerToSave  chan<- SaveRange
}

type FileChannels struct {
	ReadyChannel  chan<- Ready
	ReportIsSaved map[[20]byte]chan<- IsRangeSaved
	ToSaveChannel <-chan SaveRange
}

func GetChannels() (SupervisorChannels, TrackerChannels, PeerChannels, PieceChannels, FileChannels) {
	var sup SupervisorChannels
	var tra TrackerChannels
	var peer PeerChannels
	var piece PieceChannels
	var file FileChannels

	sup.ToPeerWorkerToDownload = make(map[[6]byte]chan<- DownloadRange)
	peerMessage := make(chan PeerMessage)
	sup.FromPeerWorker = peerMessage
	peer.PeerMessageChannel = peerMessage
	peerListCh := make(chan Peers)
	sup.GetPeers = peerListCh
	tra.SendPeers = peerListCh

	stats := make(chan Stats)
	tra.GetStatsChannel = stats
	piece.PostStatsChannel = stats

	downloadedChannel := make(chan Block)
	peer.DownloadedChannel = downloadedChannel
	piece.PeerHasDownloaded = downloadedChannel

	fileWorkerReady := make(chan Ready)
	piece.FileWorkerReady = fileWorkerReady
	file.ReadyChannel = fileWorkerReady
	file.ReportIsSaved = make(map[[20]byte]chan<- IsRangeSaved)
	fileWorkerToSave := make(chan SaveRange)
	piece.FileWorkerToSave = fileWorkerToSave
	file.ToSaveChannel = fileWorkerToSave

	return sup, tra, peer, piece, file
}

func AddNewPeer(sup SupervisorChannels, peer PeerChannels, peerId [6]byte) PeerChannels {
	newPeer := peer
	newChannel := make(chan DownloadRange)
	newPeer.ToDownload = newChannel
	sup.ToPeerWorkerToDownload[peerId] = newChannel
	return newPeer
}
