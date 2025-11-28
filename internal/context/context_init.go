package context

import (
	"central-unit/internal/common/logger"
	"central-unit/pkg/model"
	"context"
)

func InitContext(amfs model.AMF) *CuCpContext {
	cuCtx := &CuCpContext{
		Logger:  logger.InitLogger("", map[string]string{"mod": "cucp"}),
		IsReady: make(chan bool),
		Close:   make(chan struct{}),
		Ctx:     context.Background(),
	}

	amf := cuCtx.newAmf(model.AMF{Ip: "127.0.0.1", Port: 8000})
	if err := cuCtx.initAmfConn(amf); err != nil {
		cuCtx.Fatal("Error in: %v", err)
	} else {
		cuCtx.Info("SCTP/NGAP service is running")
	}
	cuCtx.SendNgSetupRequest(amf)

	// Initialize F1AP SCTP server for DU connections
	if err := cuCtx.initF1APServer(); err != nil {
		cuCtx.Fatal("Error initializing F1AP server: %v", err)
	} else {
		cuCtx.Info("SCTP/F1AP server is running")
	}

	go func() {
		<-cuCtx.Ctx.Done()
		cuCtx.Terminate()
	}()

	return cuCtx
}
