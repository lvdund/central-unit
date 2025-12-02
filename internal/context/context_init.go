package context

import (
	"central-unit/internal/common/logger"
	"central-unit/pkg/config"
	"central-unit/pkg/model"
	"context"
)

func InitContext(amfs model.AMF, cfg config.Config) *CuCpContext {
	cuCtx := &CuCpContext{
		Logger:      logger.InitLogger("", map[string]string{"mod": "cucp"}),
		IsReadyNgap: make(chan bool),
		Close:       make(chan struct{}),
		Ctx:         context.Background(),
	}

	// Set control info from config
	cuCtx.ControlInfo.ng_gnbId = cfg.NGAP.GnbId
	cuCtx.ControlInfo.ng_gnbIp = cfg.NGAP.LocalAddress
	cuCtx.ControlInfo.ng_gnbPort = cfg.NGAP.LocalPort
	cuCtx.ControlInfo.f1_gnbIp = cfg.F1AP.LocalAddress
	cuCtx.ControlInfo.f1_gnbPort = cfg.F1AP.LocalPort
	cuCtx.ControlInfo.f1_gnbId = cfg.CUCP.NodeID
	cuCtx.ControlInfo.mcc = cfg.CUCP.PLMN.MCC
	cuCtx.ControlInfo.mnc = cfg.CUCP.PLMN.MNC
	cuCtx.ControlInfo.tac = cfg.CUCP.TAC
	

	// Set slice info from config
	if len(cfg.CUCP.Slices) > 0 {
		cuCtx.SetSliceInfoFromConfig(
			cfg.CUCP.Slices[0].SST,
			cfg.CUCP.Slices[0].SD,
		)
	}

	amf := cuCtx.newAmf(model.AMF{Ip: cfg.NGAP.AMFAddress, Port: cfg.NGAP.AMFPort})
	if err := cuCtx.initAmfConn(amf); err != nil {
		cuCtx.Fatal("Error in: %v", err)
	} else {
		cuCtx.Info("SCTP/NGAP service is running")
	}
	cuCtx.SendNgSetupRequest(amf)

	<-cuCtx.IsReadyNgap

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
