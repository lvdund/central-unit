package transport

import (
	"central-unit/internal/common/logger"
	"context"
	"fmt"
	"io"
	"sync"
	"syscall"
	"time"

	"github.com/ishidawataru/sctp"
)

const (
	NGAP_PPID            uint32 = 60
	readBufferSize              = 8192
	defaultChannelBuffer        = 5000
	requestTimeout              = 2 * time.Second
)

type SctpConn struct {
	gnbId      string
	localAddr  string
	remoteAddr string
	conn       *sctp.SCTPConn

	// Channel for reading
	ReadCh chan []byte

	// Control
	*logger.Logger
	ctx context.Context
	wg  sync.WaitGroup
}

func NewSctpConn(gnbid, localAddr, remoteAddr string, ctx context.Context) *SctpConn {
	if gnbid == "" || localAddr == "" || remoteAddr == "" {
		return nil
	}

	return &SctpConn{
		gnbId:      gnbid,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		ReadCh:     make(chan []byte, defaultChannelBuffer),
		Logger:     logger.InitLogger("", map[string]string{"mod": "sctp"}),
		ctx:        ctx,
	}
}

func (sc *SctpConn) Connect() error {
	var laddr, raddr *sctp.SCTPAddr
	var err error

	if sc.localAddr != "" {
		laddr, err = sctp.ResolveSCTPAddr("sctp", sc.localAddr)
		if err != nil {
			return fmt.Errorf("resolve local addr: %w", err)
		}
	}

	raddr, err = sctp.ResolveSCTPAddr("sctp", sc.remoteAddr)
	if err != nil {
		return fmt.Errorf("resolve remote addr: %w", err)
	}

	sc.conn, err = sctp.DialSCTPExt("sctp", laddr, raddr, sctp.InitMsg{
		NumOstreams:    5,
		MaxInstreams:   3,
		MaxAttempts:    2,
		MaxInitTimeout: 2,
	})
	if err != nil {
		return fmt.Errorf("dial SCTP: %w", err)
	}

	events := sctp.SCTP_EVENT_DATA_IO | sctp.SCTP_EVENT_SHUTDOWN | sctp.SCTP_EVENT_ASSOCIATION
	if err := sc.conn.SubscribeEvents(events); err != nil {
		return fmt.Errorf("subscribe events: %w", err)
	}

	info := &sctp.SndRcvInfo{PPID: NGAP_PPID}
	if err := sc.conn.SetDefaultSentParam(info); err != nil {
		return fmt.Errorf("set default sent param: %w", err)
	}

	if err := sc.conn.SetReadBuffer(readBufferSize); err != nil {
		return fmt.Errorf("set read buffer: %w", err)
	}

	sc.Info("SCTP connection established with PPID=%d", NGAP_PPID)

	// Start read loop
	sc.wg.Add(1)
	go sc.readLoop()

	return nil
}

func (sc *SctpConn) readLoop() {
	defer sc.wg.Done()
	defer close(sc.ReadCh)

	buf := make([]byte, readBufferSize)

	for {
		// Check context cancellation before blocking read
		select {
		case <-sc.ctx.Done():
			return
		default:
		}

		n, info, err := sc.conn.SCTPRead(buf)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				sc.Error("Connection closed by peer")
				return
			}
			if err == syscall.EAGAIN || err == syscall.EINTR {
				continue
			}
			sc.Error("Read error: %v", err)
			return
		}

		if info == nil {
			sc.Error("Received nil info")
			continue
		}

		if info.PPID != NGAP_PPID {
			sc.Error("Wrong PPID: %d, expected %d", info.PPID, NGAP_PPID)
			continue
		}

		sc.Info("Received %d bytes (PPID=%d, Stream=%d)", n, info.PPID, info.Stream)

		// Copy data to avoid buffer reuse issues
		data := make([]byte, n)
		copy(data, buf[:n])

		select {
		case sc.ReadCh <- data:
		case <-sc.ctx.Done():
			return
		default:
			sc.Warn("Drop received msg, cause of full queue")
		}
	}
}

func (sc *SctpConn) Send(data []byte) error {
	if sc.conn == nil {
		return fmt.Errorf("SCTP connection not established")
	}

	info := &sctp.SndRcvInfo{
		PPID:   NGAP_PPID,
		Stream: 0,
	}

	_, err := sc.conn.SCTPWrite(data, info)
	if err != nil {
	}

	return nil
}

func (sc *SctpConn) Read() <-chan []byte {
	return sc.ReadCh
}

func (sc *SctpConn) Close() error {
	if sc.conn != nil {
		if err := sc.conn.Close(); err != nil {
			return err
		}
	}
	sc.wg.Wait()
	return nil
}
