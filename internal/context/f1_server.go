package context

import (
	"fmt"
	"io"
	"net"
	"syscall"

	"github.com/ishidawataru/sctp"
)

const F1AP_PPID uint32 = 62

// initF1APServer initializes the SCTP server for F1AP (DU connections)
func (cu *CuCpContext) initF1APServer() error {
	// Resolve IP address
	netAddr, err := net.ResolveIPAddr("ip", cu.ControlInfo.f1_gnbIp)
	if err != nil {
		return fmt.Errorf("resolve IP: %w", err)
	}

	// Create SCTP address
	addr := &sctp.SCTPAddr{
		IPAddrs: []net.IPAddr{*netAddr},
		Port:    cu.ControlInfo.f1_gnbPort,
	}

	// Create socket configuration
	config := sctp.SocketConfig{
		InitMsg: sctp.InitMsg{
			NumOstreams:    2,
			MaxInstreams:   2,
			MaxAttempts:    2,
			MaxInitTimeout: 2,
		},
	}

	// Listen
	listener, err := config.Listen("sctp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	cu.F1APListener = listener
	cu.f1apStop = make(chan struct{})

	cu.Info("F1AP server listening on %s", listener.Addr().String())

	// Start accepting connections in a goroutine
	go cu.f1apAcceptLoop()

	return nil
}

func (cu *CuCpContext) f1apAcceptLoop() {
	for {
		select {
		case <-cu.f1apStop:
			return
		default:
			conn, err := cu.F1APListener.AcceptSCTP()
			if err != nil {
				if err == syscall.EINTR || err == syscall.EAGAIN {
					continue
				}
				cu.Error("Accept error: %v", err)
				continue
			}

			if conn == nil {
				continue
			}

			cu.Info("New connection from %s", conn.RemoteAddr().String())

			// Configure the accepted connection
			if err := cu.configureF1APConnection(conn); err != nil {
				cu.Error("Failed to configure connection: %v", err)
				conn.Close()
				continue
			}

			// Handle connection in goroutine
			go cu.handleF1APConnection(conn)
		}
	}
}

func (cu *CuCpContext) configureF1APConnection(conn *sctp.SCTPConn) error {
	// Subscribe to events
	events := sctp.SCTP_EVENT_DATA_IO | sctp.SCTP_EVENT_SHUTDOWN | sctp.SCTP_EVENT_ASSOCIATION
	if err := conn.SubscribeEvents(events); err != nil {
		return fmt.Errorf("subscribe events: %w", err)
	}

	// Set default PPID
	info := &sctp.SndRcvInfo{PPID: 62}
	if err := conn.SetDefaultSentParam(info); err != nil {
		return fmt.Errorf("set default sent param: %w", err)
	}

	// Set read buffer
	if err := conn.SetReadBuffer(8192); err != nil {
		return fmt.Errorf("set read buffer: %w", err)
	}

	return nil
}

func (cu *CuCpContext) handleF1APConnection(conn *sctp.SCTPConn) {
	defer conn.Close()

	cu.Info("New DU connection")
	cu.TempDuConn = conn

	buf := make([]byte, 8192)
	remoteAddr := conn.RemoteAddr().String()

	cu.Info("Handling F1AP connection from %s", remoteAddr)

	for {
		n, info, err := conn.SCTPRead(buf)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				cu.Info("Connection %s closed", remoteAddr)
				return
			}
			if err == syscall.EAGAIN || err == syscall.EINTR {
				continue
			}
			cu.Error("Read error: %v", err)
			return
		}

		if info == nil {
			cu.Warn("Received nil info")
			continue
		}

		if info.PPID != F1AP_PPID {
			cu.Warn("Wrong PPID %d, expected %d", info.PPID, F1AP_PPID)
			continue
		}

		// Copy and dispatch message
		rawMsg := make([]byte, n)
		copy(rawMsg, buf[:n])

		go cu.dispatchF1(rawMsg)
	}
}
