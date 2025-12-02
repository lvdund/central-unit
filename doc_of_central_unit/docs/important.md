# do not edit any thing in folder [StormSIM](../StormSIM/). Just read!

# you can use some module in [StormSIM](../StormSIM/) like: 
* [sctp](../StormSIM/internal/transport/sctpngap/)
* [state-machine engine](../StormSIM/internal/common/fsm/)
* [logger](../StormSIM/internal/common/logger/logger.go)
* ngap handler in gnb + context inside gnb (uegnb, amdgnb, gnb, pdu session). like:
    * [NGSetupRequest](../StormSIM/internal/core/gnbcontext/trigger.go#L208)
    * [NGSetupResponse](../StormSIM/internal/core/gnbcontext/ngap_handler.go#L328)
    * ... all logic handler of ngap msg + context inside gnb (uegnb, amdgnb, gnb, pdu session) i folder [gnbcontext](../StormSIM/internal/core/gnbcontext/)

# but carefull (read [docs](../StormSIM/docs/) and [readme](../StormSIM/README.md)), because StormSIM simulate both UE + gnb (CU+DU), abstract RRC, F1 signaling procedure (mean directly send nas msg into gnb by go-channel)
