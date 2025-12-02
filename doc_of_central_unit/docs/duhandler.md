# this is note for handler DU of Open Air Interface

# CU-CP DU handling (what, where, and steps)

## Storage + helpers (`openair2/RRC/NR/rrc_gNB_du.c`)
- DUs live in RB tree `rrc_du_tree` keyed by `assoc_id`, each node `nr_rrc_du_container_t` holds `assoc_id`, copied `f1ap_setup_req`, decoded `mib`/`sib1`, `mtc`.
- Lookups: `get_du_by_assoc_id()`, `get_du_by_cell_id()`, `get_du_for_ue()` (resolve via `f1_ue_data`), `find_target_du()` (pick another DU), `get_cell_information_by_phycellId()`; printer `dump_du_info()`.
- Neighbour prep: `label_intra_frequency_neighbours()` marks intra-frequency neighbours using `get_ssb_arfcn()`/`get_ssb_scs()` and neighbour config; `valid_du_in_neighbour_configs()` sanity-checks DU against configured neighbours.
- UE cleanup helper: `invalidate_du_connections()` walks UE tree; for UEs on the lost DU, updates `f1_ue_data` and triggers NG release (SA) or removes NSA user context; returns count removed.

## DU attach (F1 Setup Request) — `rrc_gNB_process_f1_setup_req(req, assoc_id)`
- Guard assoc_id; log DU ID/name; prebuild failure PDU with `F1AP_get_next_transaction_identifier`.
- Reject with failure if: DU advertises !=1 cell; PLMN mismatch vs CU config; DU ID already used; cellID/PCI clashes with existing DU.
- Decode MeasurementTimingConfiguration (tolerated if missing); validate neighbour-config consistency; on error send Setup Failure.
- Decode system info (MIB mandatory; SIB1 optional in NSA) via `extract_sys_info`; on decode failure send Setup Failure.
- On accept: allocate `nr_rrc_du_container_t`, store assoc_id; deep-copy setup request (`cp_f1ap_setup_request`); store decoded MIB/SIB1/MTC; insert into DU tree; increment `num_dus`.
- Build SI to send: iterate CU SIBs (SIB2 handled) to fill `served_cells_to_activate`; label intra-frequency neighbours if MIB+SIB1 present.
- Send `f1_setup_response` (with RRC version and optional CU name); free response resources. For PHY-test/do-ra modes, call `rrc_add_nsa_user()` to create a test UE.

## DU configuration update — `rrc_gNB_process_f1_du_configuration_update(conf_up, assoc_id)`
- Fetch DU by assoc_id; ensure cell count/PLMN match. Cells-to-add/delete not supported (warn and ignore).
- For modify (supports one cell): verify old cell ID and PLMN, then `update_cell_info()` (overwrite served cell info, re-decode MTC). If sys_info provided, overwrite MIB (mandatory) and SIB1 (optional). Relabel intra-frequency neighbours when both MIB+SIB1 present.
- Log cell service states from `conf_up->status`; send `gnb_du_configuration_update_acknowledge`.

## DU loss/reset
- `rrc_CU_process_f1_lost_connection(lc, assoc_id)`: find DU; log release; free stored MIB/SIB1/MTC; free copied setup request; remove DU from tree, decrement `num_dus`; call `invalidate_du_connections()` to release/clean UEs tied to this DU; log UE count lost.
- `trigger_f1_reset(rrc, du_assoc_id)`: build F1 RESET-ALL with transport cause and send via `mac_rrc.f1_reset` (used to recover DU).



# F1AP procedures the CU‑CP handles with the DU, grouped by direction (CU↔DU) and where they are implemented.

  DU → CU (decoded/handled)

  - F1 Setup Request → CU_handle_F1_SETUP_REQUEST in openair2/F1AP/f1ap_cu_interface_management.c
  - gNB-DU Configuration Update → CU_handle_gNB_DU_CONFIGURATION_UPDATE in f1ap_cu_interface_management.c
  - Reset Acknowledge → CU_handle_RESET_ACKNOWLEDGE in f1ap_cu_interface_management.c
  - gNB-CU Configuration Update Acknowledge → CU_handle_gNB_CU_CONFIGURATION_UPDATE_ACKNOWLEDGE in f1ap_cu_interface_management.c
  - UE Context Setup Response → CU_handle_UE_CONTEXT_SETUP_RESPONSE in f1ap_cu_ue_context_management.c
  - UE Context Setup Failure → CU_handle_UE_CONTEXT_SETUP_FAILURE in f1ap_cu_ue_context_management.c
  - UE Context Release Request → CU_handle_UE_CONTEXT_RELEASE_REQUEST in f1ap_cu_ue_context_management.c
  - UE Context Release Complete → CU_handle_UE_CONTEXT_RELEASE_COMPLETE in f1ap_cu_ue_context_management.c
  - UE Context Modification Response → CU_handle_UE_CONTEXT_MODIFICATION_RESPONSE in f1ap_cu_ue_context_management.c
  - UE Context Modification Failure → CU_handle_UE_CONTEXT_MODIFICATION_FAILURE in f1ap_cu_ue_context_management.c
  - UE Context Modification Required (DU-initiated change) → CU_handle_UE_CONTEXT_MODIFICATION_REQUIRED in f1ap_cu_ue_context_management.c
  - Initial UL RRC Message Transfer → CU_handle_INITIAL_UL_RRC_MESSAGE_TRANSFER in f1ap_cu_rrc_message_transfer.c
  - UL RRC Message Transfer → CU_handle_UL_RRC_MESSAGE_TRANSFER in f1ap_cu_rrc_message_transfer.c

  CU → DU (encoded/sent)

  - F1 Setup Response → CU_send_F1_SETUP_RESPONSE in f1ap_cu_interface_management.c
  - F1 Setup Failure → CU_send_F1_SETUP_FAILURE in f1ap_cu_interface_management.c
  - gNB-CU Configuration Update → CU_send_gNB_CU_CONFIGURATION_UPDATE in f1ap_cu_interface_management.c
  - gNB-DU Configuration Update Acknowledge → CU_send_gNB_DU_CONFIGURATION_UPDATE_ACKNOWLEDGE in f1ap_cu_interface_management.c
  - Reset → CU_send_RESET in f1ap_cu_interface_management.c
  - (Reset Acknowledge send side is stubbed AssertFatal in CU_send_RESET_ACKNOWLEDGE)
  - DL RRC Message Transfer → CU_send_DL_RRC_MESSAGE_TRANSFER in f1ap_cu_rrc_message_transfer.c
  - UE Context Setup Request → CU_send_UE_CONTEXT_SETUP_REQUEST in f1ap_cu_task.c
  - UE Context Modification Request → CU_send_UE_CONTEXT_MODIFICATION_REQUEST in f1ap_cu_task.c
  - UE Context Release Command → CU_send_UE_CONTEXT_RELEASE_COMMAND in f1ap_cu_task.c
  - UE Context Modification Confirm / Refuse → CU_send_UE_CONTEXT_MODIFICATION_CONFIRM / CU_send_UE_CONTEXT_MODIFICATION_REFUSE in f1ap_cu_task.c
  - Paging → CU_send_Paging in f1ap_cu_task.c (built in f1ap_cu_paging.c)
  - DL RRC Message Transfer already noted; all SCTP dispatch for outgoing ITTI IDs is in f1ap_cu_task.c switch.
