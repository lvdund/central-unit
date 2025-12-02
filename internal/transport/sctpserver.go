package transport

// import (
// 	"io"
// 	"net"
// 	"sync"
// 	"syscall"

// 	"github.com/ishidawataru/sctp"

// 	"github.com/reogac/utils"
// 	"github.com/sirupsen/logrus"
// )

// const F1AP_PPID uint32 = 62

// // F1APServer represents an SCTP server for F1AP (DU connections)
// type F1APServer struct {
// 	*logrus.Entry
// 	connections sync.Map
// 	listener    *sctp.SCTPListener
// 	wg          sync.WaitGroup
// 	ipList      []string
// 	port        int
// 	done        chan bool
// 	onNewConn   func(*sctp.SCTPConn) // Callback for new connections
// }

// const f1apReadBufSize uint32 = 8192

// // set default read timeout to 2 seconds
// var f1apReadTimeout syscall.Timeval = syscall.Timeval{Sec: 2, Usec: 0}

// var f1apSctpConfig sctp.SocketConfig = sctp.SocketConfig{
// 	InitMsg: sctp.InitMsg{
// 		NumOstreams:    2,
// 		MaxInstreams:   2,
// 		MaxAttempts:    2,
// 		MaxInitTimeout: 2,
// 	},
// }

// // NewF1APServer creates a new F1AP SCTP server
// func NewF1APServer(ipList []string, port int, onNewConn func(*sctp.SCTPConn)) *F1APServer {
// 	logger := logrus.New()
// 	logger.SetFormatter(&logrus.TextFormatter{})
// 	entry := logger.WithFields(logrus.Fields{
// 		"mod": "f1ap",
// 	})

// 	return &F1APServer{
// 		Entry:     entry,
// 		done:      make(chan bool),
// 		ipList:    ipList,
// 		port:      port,
// 		onNewConn: onNewConn,
// 	}
// }

// // Run starts the F1AP SCTP server
// func (s *F1APServer) Run() (err error) {
// 	if s.listener != nil {
// 		return nil
// 	}

// 	ips := []net.IPAddr{}
// 	var netAddr *net.IPAddr
// 	for _, addr := range s.ipList {
// 		if netAddr, err = net.ResolveIPAddr("ip", addr); err != nil {
// 			return utils.WrapError("Resolving binding address", err)
// 		} else {
// 			s.Debugf("Resolved address '%s' to %s\n", addr, netAddr)
// 			ips = append(ips, *netAddr)
// 		}
// 	}

// 	addr := &sctp.SCTPAddr{
// 		IPAddrs: ips,
// 		Port:    s.port,
// 	}

// 	if s.listener, err = f1apSctpConfig.Listen("sctp", addr); err != nil {
// 		return utils.WrapError("Server start listening", err)
// 	}
// 	s.Infof("F1AP server listening on %s", s.listener.Addr())
// 	go s.loop()
// 	return
// }

// func (s *F1APServer) loop() {
// 	s.wg.Add(1)
// 	defer s.wg.Done()
// 	s.Infof("Waiting for F1AP connections")
// 	for {
// 		select {
// 		case <-s.done:
// 			return
// 		default:
// 			newConn, err := s.listener.AcceptSCTP()
// 			if err != nil {
// 				s.Errorf("Receive an error while waiting for new connections: %+v", err)
// 				switch err {
// 				case syscall.EINTR, syscall.EAGAIN:
// 					s.Debugf("EINTR or EAGAIN error type")
// 				default:
// 				}
// 				continue
// 			} else if newConn != nil {
// 				s.Debugf("New connection from %s:%s", newConn.RemoteAddr().Network(), newConn.RemoteAddr().String())
// 				var info *sctp.SndRcvInfo
// 				if infoTmp, err := newConn.GetDefaultSentParam(); err != nil {
// 					s.Errorf("Get default sent param error: %+v, accept failed", err)
// 					if err = newConn.Close(); err != nil {
// 						s.Errorf("Close error: %+v", err)
// 					}
// 					continue
// 				} else {
// 					info = infoTmp
// 					s.Tracef("Get default sent param[value: %+v]", info)
// 				}

// 				info.PPID = F1AP_PPID
// 				if err := newConn.SetDefaultSentParam(info); err != nil {
// 					s.Errorf("Set default sent param error: %+v, accept failed", err)
// 					if err = newConn.Close(); err != nil {
// 						s.Errorf("Close error: %+v", err)
// 					}
// 					continue
// 				} else {
// 					s.Tracef("Set default sent param[value: %+v]", info)
// 				}

// 				events := sctp.SCTP_EVENT_DATA_IO | sctp.SCTP_EVENT_SHUTDOWN | sctp.SCTP_EVENT_ASSOCIATION
// 				if err := newConn.SubscribeEvents(events); err != nil {
// 					s.Errorf("Failed to subscribe events: %+v", err)
// 					if err = newConn.Close(); err != nil {
// 						s.Errorf("Close error: %+v", err)
// 					}
// 					continue
// 				} else {
// 					s.Tracef("Subscribe SCTP event[DATA_IO, SHUTDOWN_EVENT, ASSOCIATION_CHANGE]")
// 				}

// 				if err := newConn.SetReadBuffer(int(f1apReadBufSize)); err != nil {
// 					s.Errorf("Set read buffer error: %+v, accept failed", err)
// 					if err = newConn.Close(); err != nil {
// 						s.Errorf("Close error: %+v", err)
// 					}
// 					continue
// 				} else {
// 					s.Tracef("Set read buffer to %d bytes", f1apReadBufSize)
// 				}

// 				// Note: SetReadTimeout may not be available in all SCTP implementations
// 				// Skipping timeout setting for now

// 				s.Infof("SCTP Accept from: %s", newConn.RemoteAddr().String())
// 				s.connections.Store(newConn, newConn)

// 				// Call callback for new connection
// 				if s.onNewConn != nil {
// 					s.onNewConn(newConn)
// 				}

// 				go s.handleConnection(newConn, f1apReadBufSize)
// 			} else {
// 				s.Tracef("Listener timeouted")
// 			}
// 		}
// 	}
// }

// // Stop stops the F1AP SCTP server
// func (s *F1APServer) Stop() {
// 	if s.listener == nil {
// 		s.Warnf("F1AP SCTP server not running")
// 		return
// 	}

// 	s.Debugf("Close F1AP SCTP listener...")
// 	if err := s.listener.Close(); err != nil {
// 		s.Errorf("F1AP SCTP listener may not close normally: %+v", err)
// 	}
// 	close(s.done)
// 	s.connections.Range(func(key, value interface{}) bool {
// 		conn := value.(net.Conn)
// 		if err := conn.Close(); err != nil {
// 			s.Error("Close connection returns error: %+v", err)
// 		}
// 		return true
// 	})
// 	s.wg.Wait()
// 	s.Infof("F1AP SCTP server closed")
// }

// // handleConnection handles messages from a single DU connection
// func (s *F1APServer) handleConnection(conn *sctp.SCTPConn, bufsize uint32) {
// 	s.wg.Add(1)
// 	defer func() {
// 		// in case calling Stop(), conn.Close() will return EBADF because
// 		// conn has been already closed
// 		if err := conn.Close(); err != nil && err != syscall.EBADF {
// 			s.Errorf("close connection error: %+v", err)
// 		}
// 		s.connections.Delete(conn)
// 		s.wg.Done()
// 	}()

// 	remoteaddr := conn.RemoteAddr().String()
// 	s.Infof("Handling F1AP connection from %s", remoteaddr)

// 	buf := make([]byte, bufsize)
// 	for {
// 		n, info, err := conn.SCTPRead(buf)
// 		if err != nil {
// 			switch err {
// 			case io.EOF, io.ErrUnexpectedEOF:
// 				s.Errorf("Connection %s closed", remoteaddr)
// 				return
// 			case syscall.EAGAIN:
// 				s.Trace("SCTP read timeout")
// 				continue
// 			case syscall.EINTR:
// 				s.Debugf("SCTPRead: %+v", err)
// 				continue
// 			default:
// 				s.Errorf("Handle connection[addr: %+v] error: %+v", conn.RemoteAddr(), err)
// 				return
// 			}
// 		}

// 		if info == nil || info.PPID != F1AP_PPID {
// 			s.Warnf("Received SCTP PPID != %d, discard this packet", F1AP_PPID)
// 			continue
// 		}

// 		// Message will be handled by the callback (onNewConn) which sets up
// 		// the DU context and message reading loop
// 		// This handleConnection is mainly for connection lifecycle management
// 		_ = n // Message already handled by callback
// 	}
// }

// // handleSCTPNotification handles SCTP notifications (placeholder for future use)
// func (s *F1APServer) handleSCTPNotification(conn *sctp.SCTPConn, notification interface{}) {
// 	s.Debugf("Received SCTP notification from %s: %+v", conn.RemoteAddr().String(), notification)
// }

// // Send sends data through a specific connection
// func (s *F1APServer) Send(conn *sctp.SCTPConn, data []byte) error {
// 	info := &sctp.SndRcvInfo{
// 		PPID:   F1AP_PPID,
// 		Stream: 0,
// 	}
// 	_, err := conn.SCTPWrite(data, info)
// 	return err
// }
