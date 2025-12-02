# UE initial connection: CU-CP handled messages and handlers

Source flow from `doc/RRC/rrc-dev.md` (initial connection diagram). Messages below are only those the CU-CP handles; DU/UE PHY/MAC steps (Msg1/2) are omitted.

## Access + RRC establishment
- DU → CU: `F1AP Initial UL RRC Message Transfer` (carries `RRCSetupRequest` in Msg3)  
  Handler: `CU_handle_INITIAL_UL_RRC_MESSAGE_TRANSFER()` in `openair2/F1AP/f1ap_cu_rrc_message_transfer.c` → ITTI to RRC → `rrc_gNB_process_initial_ul_rrc_message()` → `rrc_handle_RRCSetupRequest()` in `openair2/RRC/NR/rrc_gNB.c`.
- CU → UE (via DU): `F1AP DL RRC Message Transfer` with `RRCSetup` (Msg4)  
  Sent from `rrc_gNB_generate_RRCSetup()` (RR context) → `CU_send_DL_RRC_MESSAGE_TRANSFER()` in `f1ap_cu_rrc_message_transfer.c`.
- UE → CU: `F1AP UL RRC Message Transfer` with `RRCSetupComplete`  
  Handler: `CU_handle_UL_RRC_MESSAGE_TRANSFER()` in `f1ap_cu_rrc_message_transfer.c` → `rrc_gNB_decode_dcch()` → `handle_rrcSetupComplete()` in `openair2/RRC/NR/rrc_gNB.c`.

## NAS forwarding to AMF
- CU → AMF: `NGAP Initial UE Message` (NAS Registration/Service Req)  
  Built in `rrc_gNB_send_NGAP_NAS_FIRST_REQ()` in `openair2/RRC/NR/rrc_gNB_NGAP.c` after `RRCSetupComplete`.
- AMF → CU: `NGAP Initial Context Setup Request`  
  Handler: `rrc_gNB_process_NGAP_INITIAL_CONTEXT_SETUP_REQ()` in `rrc_gNB_NGAP.c`.

## AS security + UE caps
- CU → UE: `F1AP DL RRC Message Transfer` with `SecurityModeCommand`  
  Built in `rrc_gNB_generate_SecurityModeCommand()` (called from the Initial Context Setup path) → sent via `CU_send_DL_RRC_MESSAGE_TRANSFER()`.
- UE → CU: `F1AP UL RRC Message Transfer` with `SecurityModeComplete`  
  Processed in `rrc_gNB_decode_dcch()` → `securityModeComplete` branch in `rrc_gNB.c` (enables PDCP security, may trigger UE Cap Enquiry or bearer setup).
- Optional UE capabilities: DL `UECapabilityEnquiry` and UL `UECapabilityInformation` go through the same DL/UL RRC transfer handlers above; CU-side logic in `rrc_gNB_generate_UECapabilityEnquiry()` / `handle_ueCapabilityInformation()` in `rrc_gNB.c`.

## PDU session setup path (if AMF asked for resources)
- CU → CU-UP: `E1AP Bearer Context Setup Request`  
  Sent by `trigger_bearer_setup()`/`cucp_cuup` interface from `rrc_gNB.c` (E1 sender `cucp_cuup_e1ap.c`).
- CU-UP → CU: `E1AP Bearer Context Setup Response`  
  Handler: `e1apCUCP_handle_BEARER_CONTEXT_SETUP_RESPONSE()` in `openair2/E1AP/e1ap.c`, dispatched to RRC via ITTI → `rrc_gNB_process_e1_bearer_context_setup_resp()` in `rrc_gNB_cuup.c`.
- CU → DU: `F1AP UE Context Setup Request` (configures DRBs, CellGroup)  
  Built in `rrc_gNB_generate_f1_ue_context_setup_req()` (RRC) and sent via `CU_send_UE_CONTEXT_SETUP_REQUEST()` in `f1ap_cu_task.c`.
- DU → CU: `F1AP UE Context Setup Response`  
  Handler: `CU_handle_UE_CONTEXT_SETUP_RESPONSE()` in `f1ap_cu_ue_context_management.c` → `rrc_CU_process_ue_context_setup_response()` in `rrc_gNB.c`; triggers `e1_send_bearer_updates()` if needed.
- CU → UE: `F1AP DL RRC Message Transfer` with `RRCReconfiguration` (activates DRBs)  
  Generated in `rrc_gNB_generate_RRCReconfiguration()` / `e1_send_bearer_updates()` path and sent via `CU_send_DL_RRC_MESSAGE_TRANSFER()`.
- UE → CU: `F1AP UL RRC Message Transfer` with `RRCReconfigurationComplete`  
  Handled in `rrc_gNB_decode_dcch()` → `handle_rrcReconfigurationComplete()` in `rrc_gNB.c`.

## Completion toward AMF
- CU → AMF: `NGAP Initial Context Setup Response` (after RRC Reconfiguration Complete / bearer setup)  
  Built in `rrc_gNB_send_NGAP_INITIAL_CONTEXT_SETUP_RESP()` in `openair2/RRC/NR/rrc_gNB_NGAP.c` (also forwards NAS Registration/Service Complete).
