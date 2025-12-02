Message handling (where the CU-CP logic lives)
- RRC dispatcher: `rrc_gnb_task()` (`openair2/RRC/NR/rrc_gNB.c:2925+`) switch-cases ITTI messages. Key entries:
  - From F1AP: `F1AP_INITIAL_UL_RRC_MESSAGE` -> `rrc_gNB_process_initial_ul_rrc_message()`; `F1AP_UL_RRC_MESSAGE` -> `rrc_gNB_decode_dcch()`; UE context setup/mod/rel responses -> `rrc_CU_process_*`; DU lost connection -> `rrc_CU_process_f1_lost_connection()`.
  - From NGAP: NAS DL/UL and session control -> `rrc_gNB_process_NGAP_*` functions; UE context release commands; paging, HO requests/commands.
  - From E1AP: setup req -> `rrc_gNB_process_e1_setup_req()`; bearer context setup/mod/rel handlers (`rrc_gNB_process_e1_bearer_context_*`); lost connection -> `rrc_gNB_process_e1_lost_connection()`.
- F1 (CU side):
  - SCTP/task glue in `F1AP_CU_task()` (`openair2/F1AP/f1ap_cu_task.c`): SCTP_DATA_IND calls `f1ap_handle_message()`; RRC-triggered DL messages encoded via `CU_send_*` functions.
  - DU onboarding: `rrc_gNB_process_f1_setup_req()` (`rrc_gNB_du.c`) validates PLMN/cell, decodes MIB/SIB1/MTC, allocates `nr_rrc_du_container_t`, sends `f1_setup_response` or failure via `rrc->mac_rrc.*`.
  - UE lifecycle on F1: mapping stored in `f1ap_ids.c`; UE context release path uses `rrc_gNB_generate_RRCRelease()` -> F1 UE Context Release Command.
  - Loss handling: `rrc_CU_process_f1_lost_connection()`/`invalidate_du_connections()` drop UE contexts on a DU and trigger NGAP release + F1 Reset.
- E1 (CU-CP to CU-UP):
  - Transport callbacks set in `cucp_cuup_message_transfer_e1ap_init()` (CUCP sends `E1AP_BEARER_CONTEXT_*` ITTI).
  - CU-UP registration: `rrc_gNB_process_e1_setup_req()` (`rrc_gNB_cuup.c`) checks CU-UP ID/PLMN, stores in CU-UP tree, prunes UEs without CU-UP via `remove_unassociated_e1_connections()`, replies with `E1AP_SETUP_RESP`.
  - Bearer updates from RRC core (`rrc_gNB.c`): `trigger_bearer_setup()` (on PDU session setup), `e1_send_bearer_updates()` and `e1_send_bearer_modification_request()` (after F1 UE context setup or HO), `e1_notify_pdcp_status()` (DL RAN status transfer), `e1_request_pdcp_status()` for HO.
  - CU-UP loss: `rrc_gNB_process_e1_lost_connection()` clears UE->CUUP mapping and deletes CU-UP container; also triggers F1 resets for affected DUs.
- NGAP (AMF side):
  - AMF association/NG Setup: driven by `ngap_gNB_handle_register_gNB()` and `ngap_gNB_generate_ng_setup_request()` in `openair3/NGAP/ngap_gNB.c`; AMF contexts in RB tree keyed by assoc id.
  - UE context mgmt: NGAP handlers in `ngap_gNB_handlers.c`/`ngap_gNB_context_management_procedures.c` create/update `ngap_gNB_ue_context_t` (stored via `ngap_store_ue_context()`).
  - RRC->NGAP entry points in `rrc_gNB_NGAP.c`: NAS first req `rrc_gNB_send_NGAP_NAS_FIRST_REQ()`, Initial Context Setup response `rrc_gNB_send_NGAP_INITIAL_CONTEXT_SETUP_RESP()`, UL NAS transport `rrc_gNB_send_NGAP_UPLINK_NAS()`, PDU session setup/modify/release replies, UE capability indication, HO Required/Notify/Failure, etc.
- UE-facing RRC procedures (selected CU-CP handlers in `rrc_gNB.c`):
  - Initial attach: `rrc_handle_RRCSetupRequest()` -> `rrc_gNB_process_RRCSetupComplete()` -> triggers NGAP Initial UE message and security mode (`rrc_gNB_generate_SecurityModeCommand()`).
  - PDU session setup: `rrc_gNB_process_NGAP_PDUSESSION_SETUP_REQ()` -> `trigger_bearer_setup()` (E1) -> `rrc_gNB_generate_UeContextSetupRequest()` (F1) -> `rrc_gNB_encode_RRCReconfiguration()` for DL.
  - Reestablishment: `rrc_handle_RRCReestablishmentRequest()` with `handle_rrcReestablishmentComplete()`; notifies CU-UP via `cuup_notify_reestablishment()` (E1).
  - HO: mobility helpers in `rrc_gNB_mobility.c` (`nr_rrc_trigger_f1_ho()`, `nr_rrc_trigger_n2_ho()`, HO ack/success/cancel callbacks).
- Where to look next:
  - F1 message encoder/decoders: `openair2/F1AP/lib/*` and CU handlers `f1ap_cu_*`.
  - E1 bearer message helpers: `openair2/E1AP/lib/e1ap_bearer_context_management.*`.
  - NGAP procedure helpers: `openair3/NGAP/ngap_gNB_*_procedures.c`.

---

Recent survey + scaffolding notes (based on StormSIM patterns)
- SCTP wiring: added client/server helpers (ishidawataru/sctp) with NGAP/F1AP/E1AP PPIDs; transports now dial/listen via SCTP and push events into dispatcher with conn_id metadata for routing responses.
- NGAP decode: wrapper parses PDUs and tags key procedures (setup resp/failure, downlink NAS, initial context, UE release) to surface UE IDs to logic.
- RRC FSM: expanded states (idle/attaching/connected/connected_inactive/releasing) and events (setup req, context setup, release, suspend/resume, reestablish) to align with CU-CP flows.
- Logging: switched to zerolog wrapper for structured logs consistent with StormSIM style.
- Next: replace placeholder F1AP/E1AP event names with real message classification, wire UE ID/indexing per DU/AMF, and feed logic handlers for NAS/RRC procedures.

CU-CP implementation tasks (current focus)
- Transports: finalize SCTP client (NGAP) + server (F1AP/E1AP) wiring; add AMF reconnect/backoff and DU/CU-UP accept loop health metrics; validate PPID/stream config against config.yml.
- Context/indexes: finish UE/DU/CU-UP/AMF stores and secondary indexes (AMF UE NGAP ID, DU+RNTI); persist conn_id per SCTP assoc for response routing.
- NGAP logic: implement NG Setup (init/initiate + resp/fail handling), Initial UE flow (UL NAS -> InitialContextSetup -> DL NAS), UE Context Release, DL/UL NAS transport; mirror StormSIM handlers for ID mapping and slice/plmn capture.
- F1AP logic: decode Initial UL RRC -> RRC Setup -> UE Context Setup; handle DU F1 setup, UE context release/reset, DU loss; map DU assoc + UE IDs; carry NAS through RRC Reconfig.
- E1AP logic: CU-UP setup, bearer context setup/mod/rel; handle CU-UP loss by dropping PDCP bearers and triggering F1/NG release.
- RRC procedures: drive FSM on events (setup, context setup, release, suspend/resume, reestablish, HO) and integrate timers (inactivity, paging) using timer wheel.
- Observability/testing: propagate zerolog fields (ue/du/cuup/assoc), add metrics stubs, and build fuzz/integration harnesses (DU/AMF/CU-UP simulators) similar to StormSIM.
