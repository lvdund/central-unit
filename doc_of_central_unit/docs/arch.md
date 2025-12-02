High-level CU-CP architecture in OAI (code inspection)
- Main control thread `rrc_gnb_task()` (`openair2/RRC/NR/rrc_gNB.c`) is single-threaded and processes ITTI messages from F1AP, E1AP, NGAP (and timers). It drives UE procedures and orchestrates CU-UP/DU/AMF interactions.
- F1 control: `F1AP_CU_task()` (`openair2/F1AP/f1ap_cu_task.c`) runs an ITTI thread handling SCTP events. It decodes F1 PDUs via `f1ap_handle_message()` and forwards ITTI messages (e.g., `F1AP_SETUP_REQ`, `F1AP_UL_RRC_MESSAGE`) to RRC; downlink messages from RRC are encoded and sent over SCTP here.
- E1 interface (CU-CP <-> CU-UP): CU-CP registers callbacks in `cucp_cuup_e1ap_init()` (`openair2/RRC/NR/cucp_cuup_e1ap.c`) so RRC can push bearer context setup/mod/rel ITTI messages to the E1 task (`TASK_CUCP_E1`). CU-UP signalling is handled by RRC functions in `rrc_gNB_cuup.c`.
- NG interface (CU-CP <-> AMF): NGAP module in `openair3/NGAP` runs `ngap_gNB_task()` which manages SCTP to AMF and exchanges NGAP ITTI messages with RRC (NAS transport, context setup/release, HO).
- Data-plane split awareness: CU-CP keeps DU list (F1-C), CU-UP list (E1), and UE mappings to DU/CU-UP (`f1ap_ids.c`). GTP-U is started in `F1AP_CU_task` only when running integrated CU (no E1); otherwise CU-UP handles UP.
- Reference docs: `doc/F1AP/F1-design.md` (thread model, F1 message routing), `doc/E1AP/E1-design.md` (CUCP<->CUUP procedures, callbacks), `doc/RRC/rrc-dev.md` (RRC procedures from CU-CP view).



# Map of CU-CP

```mermaid
flowchart LR
  subgraph CU_CP[RRC of CU-CP core\nopenair2/RRC/NR/rrc_gNB.c]
    RRC_TASK[RRC task\nrrc_gnb_task]
  end

  subgraph F1CU[F1AP CU task\nopenair2/F1AP/f1ap_cu_task.c]
    F1_SCTP[SCTP data ind/resp\ncu_task_handle_*]
    F1_ENC[CU_send_* encoders\nF1AP lib]
  end

  subgraph E1CU[E1AP CU-CP side\nopenair2/RRC/NR/cucp_cuup_e1ap.c]
    E1_CB[callbacks -> TASK_CUCP_E1\nbearer setup/mod/rel]
  end

  subgraph NGAP[NGAP task\nopenair3/NGAP/ngap_gNB.c]
    NG_TASK[ngap_gNB_task\nAMF SCTP]
  end

  subgraph DU[Distributed Unit\nF1-C peer]
    DU_F1[F1AP DU task]
  end

  subgraph CUUP[CU-UP\nE1 peer]
    CUUP_E1[E1AP CU-UP task]
  end

  AMF[(AMF\nNGAP peer)]
  UE[(UE)]

  UE ---|RRC| DU_F1
  DU_F1 <-->|F1-C SCTP| F1_SCTP
  F1_ENC -->|ITTI msgs F1AP_*| RRC_TASK
  RRC_TASK -->|DL RRC / UE Ctxt| F1_ENC
  RRC_TASK <-->|ITTI NGAP_*| NG_TASK
  NG_TASK <-->|SCTP NG| AMF
  RRC_TASK <-->|ITTI E1AP_*| E1_CB
  E1_CB <-->|SCTP E1| CUUP_E1
  RRC_TASK ---|config/state| RC[RC.nrrrc & context trees\nUE/DU/CU-UP maps]
```

Component notes
- RRC task: single-threaded CU-CP brain; dispatches F1AP/NGAP/E1AP ITTI, maintains UE/DU/CU-UP trees and triggers RRC/PDCP/HO procedures.
- F1AP CU task: SCTP listener + encoder/decoder; translates between SCTP and ITTI (`F1AP_*`) for DU signalling; starts GTP-U only in integrated mode.
- E1AP CU-CP side: callback shim registering `bearer_context_*` senders into E1 task so RRC can drive CU-UP bearer lifecycle.
- NGAP task: manages AMF associations, NG Setup, NAS transport, context setup/release, and HO messaging; exchanges ITTI with RRC.
- Context storage: global `RC.nrrrc[0]` holds RRC instance with trees for UE contexts, DUs, CU-UPs; `f1ap_ids.c` maps CU UE IDs to DU/CU-UP assoc IDs and DU UE IDs.
