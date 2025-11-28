package transport

import (
	"central-unit/internal/common/logger"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/alitto/pond/v2"
	"github.com/ishidawataru/sctp"
)

const (
	payloadSize                      = 65535
	requestTimeout                   = 2 * time.Second
	sctpDefaultNumOstreams           = 5
	sctpDefaultMaxInstreams          = 3
	sctpDefaultMaxAttempts           = 2
	sctpDefaultMaxInitTimeout        = 2
	defaultChannelBuffer             = 5000
	NGAP_PPID                 uint32 = 60
)

var bufferPool = sync.Pool{
	New: func() any {
		return make([]byte, payloadSize)
	},
}

type SctpConn struct {
	gnbId      string
	localAddr  string
	remoteAddr string
	conn       *sctp.SCTPConn

	// Channels for reading & Worker configuration
	ReadCh       chan []byte
	readWorkers  int
	writeWorkers pond.Pool

	// Control
	*logger.Logger
	Timeout time.Duration
	ctx     context.Context
	wg      sync.WaitGroup

	// Mutexes to protect SCTP connection calls
	readMutex  sync.Mutex
	writeMutex sync.Mutex
}

func NewSctpConn(gnbid, localAddr, remoteAddr string, ctx context.Context) *SctpConn {
	if gnbid == "" || localAddr == "" || remoteAddr == "" {
		return nil
	}

	return &SctpConn{
		gnbId:      gnbid,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,

		ReadCh:       make(chan []byte, defaultChannelBuffer),
		readWorkers:  100,
		writeWorkers: pond.NewPool(100),

		Logger:  logger.InitLogger("", map[string]string{"mod": "sctp"}),
		Timeout: requestTimeout,
		ctx:     ctx,
	}
}

func (sc *SctpConn) Connect() error {
	var laddr, raddr *sctp.SCTPAddr
	var err error

	if sc.localAddr != "" {
		laddr, err = sctp.ResolveSCTPAddr("sctp", sc.localAddr)
		if err != nil {
			return fmt.Errorf("resolve local addr: %v", err)
		}
	}

	raddr, err = sctp.ResolveSCTPAddr("sctp", sc.remoteAddr)
	if err != nil {
		return fmt.Errorf("resolve remote addr: %v", err)
	}

	sc.conn, err = sctp.DialSCTPExt("sctp", laddr, raddr, sctp.InitMsg{
		NumOstreams:    sctpDefaultNumOstreams,
		MaxInstreams:   sctpDefaultMaxInstreams,
		MaxAttempts:    sctpDefaultMaxAttempts,
		MaxInitTimeout: sctpDefaultMaxInitTimeout,
	})
	if err != nil {
		return fmt.Errorf("dial SCTP: %v", err)
	}
	sc.conn.SubscribeEvents(sctp.SCTP_EVENT_DATA_IO)

	// Start workers
	sc.startWorkers()

	return nil
}

// read and write workers
func (sc *SctpConn) startWorkers() {
	// Multiple read workers for parallel receiving
	for i := 0; i < sc.readWorkers; i++ {
		sc.wg.Add(1)
		go sc.readWorker()
	}

	// Multiple write workers for parallel sending
	// for i := 0; i < sc.writeWorkers; i++ {
	// 	sc.wg.Add(1)
	// 	go sc.writeWorker()
	// }
}

func (sc *SctpConn) readWorker() {
	defer sc.wg.Done()

	// Each worker needs its own buffer to avoid race conditions
	buf := make([]byte, payloadSize)

	for {
		select {
		case <-sc.ctx.Done():
			return
		default:
			// Only ONE worker should call SCTPRead at a time
			// Need to serialize this call
			n, _, err := sc.readFromConn(buf)
			if err != nil {
				continue
			}

			// sc.Info("[SCTP] Received message on stream %d", info.Stream)

			// Copy data to avoid buffer reuse issues
			data := make([]byte, n)
			copy(data, buf[:n])

			select {
			case sc.ReadCh <- data:
			default: // Drop if channel full
				sc.Warn("Drop received msg, cause of full queue.")
				//logger.SctpConnStats[sc.gnbId].DLmessagesDropped.Add(1)
			}
		}
	}
}

func (sc *SctpConn) Send(data []byte) {
	streamID := uint16(0)
	sc.writeWorkers.Submit(func() {
		err := sc.writeToConn(data, streamID)
		if err != nil {
			sc.Error("[SCTP] Write error: %v\n", err)
			//logger.SctpConnStats[sc.gnbId].ULmessagesDropped.Add(1)
		} else {
			//logger.SctpConnStats[sc.gnbId].ULmessages.Add(1)
		}
	})
	streamID = (streamID + 1) % sctpDefaultNumOstreams
}

func (sc *SctpConn) Read() <-chan []byte {
	return sc.ReadCh
}

func (sc *SctpConn) Close() error {
	if sc.conn != nil {
		sc.conn.Close()
	}
	sc.wg.Wait()
	return nil
}

// Thread-safe SCTP read with mutex protection
func (sc *SctpConn) readFromConn(buf []byte) (int, *sctp.SndRcvInfo, error) {
	sc.readMutex.Lock()
	defer sc.readMutex.Unlock()
	return sc.conn.SCTPRead(buf)
}

// Thread-safe SCTP write with mutex protection
func (sc *SctpConn) writeToConn(data []byte, streamID uint16) error {
	sc.writeMutex.Lock()
	defer sc.writeMutex.Unlock()

	sc.conn.SetWriteDeadline(time.Now().Add(requestTimeout))

	info := &sctp.SndRcvInfo{
		Stream: streamID,
		PPID:   NGAP_PPID,
	}

	_, err := sc.conn.SCTPWrite(data, info)
	return err
}
