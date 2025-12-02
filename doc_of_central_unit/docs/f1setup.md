Guide to handle f1setup request from DU and send f1setup response

```c
void rrc_gNB_process_f1_setup_req(f1ap_setup_req_t *req, sctp_assoc_t assoc_id)
{
  AssertFatal(assoc_id != 0, "illegal assoc_id == 0: should be -1 (monolithic) or >0 (split)\n");
  gNB_RRC_INST *rrc = RC.nrrrc[0];
  DevAssert(rrc);

  LOG_I(NR_RRC, "Received F1 Setup Request from gNB_DU %lu (%s) on assoc_id %d\n", req->gNB_DU_id, req->gNB_DU_name, assoc_id);
  // pre-fill F1 Setup Failure message
  f1ap_setup_failure_t fail = {.transaction_id = F1AP_get_next_transaction_identifier(0, 0)};

  // check:
  // - it is one cell
  // - PLMN and Cell ID matches
  // - no previous DU with the same ID
  // else reject
  if (req->num_cells_available != 1) {
    LOG_E(NR_RRC, "can only handle on DU cell, but gNB_DU %ld has %d\n", req->gNB_DU_id, req->num_cells_available);
    fail.cause = F1AP_CauseRadioNetwork_gNB_CU_Cell_Capacity_Exceeded;
    rrc->mac_rrc.f1_setup_failure(assoc_id, &fail);
    return;
  }
  f1ap_served_cell_info_t *cell_info = &req->cell[0].info;
  if (!rrc_gNB_plmn_matches(rrc, cell_info)) {
    LOG_E(NR_RRC,
          "PLMN mismatch: CU %03d.%0*d, DU %03d%0*d\n",
          rrc->configuration.plmn[0].mcc,
          rrc->configuration.plmn[0].mnc_digit_length,
          rrc->configuration.plmn[0].mnc,
          cell_info->plmn.mcc,
          cell_info->plmn.mnc_digit_length,
          cell_info->plmn.mnc);
    fail.cause = F1AP_CauseRadioNetwork_plmn_not_served_by_the_gNB_CU;
    rrc->mac_rrc.f1_setup_failure(assoc_id, &fail);
    return;
  }
  nr_rrc_du_container_t *it = NULL;
  RB_FOREACH(it, rrc_du_tree, &rrc->dus) {
    if (it->setup_req->gNB_DU_id == req->gNB_DU_id) {
      LOG_E(NR_RRC,
            "gNB-DU ID: existing DU %s on assoc_id %d already has ID %ld, rejecting requesting gNB-DU\n",
            it->setup_req->gNB_DU_name,
            it->assoc_id,
            it->setup_req->gNB_DU_id);
      fail.cause = F1AP_CauseMisc_unspecified;
      rrc->mac_rrc.f1_setup_failure(assoc_id, &fail);
      return;
    }
    // note: we assume that each DU contains only one cell; otherwise, we would
    // need to check every cell in the requesting DU to any existing cell.
    const f1ap_served_cell_info_t *exist_info = &it->setup_req->cell[0].info;
    const f1ap_served_cell_info_t *new_info = &req->cell[0].info;
    if (exist_info->nr_cellid == new_info->nr_cellid || exist_info->nr_pci == new_info->nr_pci) {
      LOG_E(NR_RRC,
            "existing DU %s on assoc_id %d already has cellID %ld/physCellId %d, rejecting requesting gNB-DU with cellID %ld/physCellId %d\n",
            it->setup_req->gNB_DU_name,
            it->assoc_id,
            exist_info->nr_cellid,
            exist_info->nr_pci,
            new_info->nr_cellid,
            new_info->nr_pci);
      fail.cause = F1AP_CauseMisc_unspecified;
      rrc->mac_rrc.f1_setup_failure(assoc_id, &fail);
      return;
    }
  }

  // MTC is mandatory, but some DUs don't send it in the F1 Setup Request, so
  // "tolerate" this behavior, despite it being mandatory
  NR_MeasurementTimingConfiguration_t *mtc =
      extract_mtc(cell_info->measurement_timing_config, cell_info->measurement_timing_config_len);

  if (rrc->neighbour_cell_configuration
      && !valid_du_in_neighbour_configs(rrc->neighbour_cell_configuration, cell_info, ssb_arfcn_mtc(mtc))) {
    LOG_E(NR_RRC, "problem with DU %ld in neighbor configuration, rejecting DU\n", req->gNB_DU_id);
    f1ap_setup_failure_t fail = {.cause = F1AP_CauseMisc_unspecified};
    rrc->mac_rrc.f1_setup_failure(assoc_id, &fail);
    ASN_STRUCT_FREE(asn_DEF_NR_MeasurementTimingConfiguration, mtc);
    return;
  }

  const f1ap_gnb_du_system_info_t *sys_info = req->cell[0].sys_info;
  NR_MIB_t *mib = NULL;
  NR_SIB1_t *sib1 = NULL;

  if (sys_info != NULL && sys_info->mib != NULL && !(sys_info->sib1 == NULL && IS_SA_MODE(get_softmodem_params()))) {
    if (!extract_sys_info(sys_info, &mib, &sib1)) {
      LOG_W(NR_RRC, "rejecting DU ID %ld\n", req->gNB_DU_id);
      fail.cause = F1AP_CauseProtocol_semantic_error;
      rrc->mac_rrc.f1_setup_failure(assoc_id, &fail);
      ASN_STRUCT_FREE(asn_DEF_NR_MeasurementTimingConfiguration, mtc);
      return;
    }
  }
  LOG_I(NR_RRC, "Accepting DU %ld (%s), sending F1 Setup Response\n", req->gNB_DU_id, req->gNB_DU_name);
  LOG_I(NR_RRC, "DU uses RRC version %u.%u.%u\n", req->rrc_ver[0], req->rrc_ver[1], req->rrc_ver[2]);

  // we accept the DU
  nr_rrc_du_container_t *du = calloc(1, sizeof(*du));
  AssertFatal(du, "out of memory\n");
  du->assoc_id = assoc_id;

  /* ITTI will free the setup request message via free(). So the memory
   * "inside" of the message will remain, but the "outside" container no, so
   * allocate memory and copy it in */
  du->setup_req = calloc(1,sizeof(*du->setup_req));
  AssertFatal(du->setup_req, "out of memory\n");
  // Copy F1AP message
  *du->setup_req = cp_f1ap_setup_request(req);
  // MIB can be null and configured later via DU Configuration Update
  du->mib = mib;
  du->sib1 = sib1;
  du->mtc = mtc;
  RB_INSERT(rrc_du_tree, &rrc->dus, du);
  rrc->num_dus++;

  served_cells_to_activate_t cell = {
      .plmn = cell_info->plmn,
      .nr_cellid = cell_info->nr_cellid,
      .nrpci = cell_info->nr_pci,
      .num_SI = 0,
  };

  // Encode CU SIBs and configure setup response with sysinfo
  seq_arr_t *sibs = rrc->SIBs;
  if (sibs) {
    for (int i = 0; i < sibs->size; i++) {
      nr_SIBs_t *sib = (nr_SIBs_t *)seq_arr_at(sibs, i);
      switch (sib->SIB_type) {
        case 2: {
          NR_SSB_MTC_t *ssbmtc = get_ssb_mtc(mtc);
          sib->SIB_size = do_SIB2_NR(&sib->SIB_buffer, ssbmtc);
          cell.SI_msg[cell.num_SI].SI_container = sib->SIB_buffer;
          cell.SI_msg[cell.num_SI].SI_container_length = sib->SIB_size;
          cell.SI_msg[cell.num_SI].SI_type = sib->SIB_type;
          cell.num_SI++;
        } break;
        default:
          AssertFatal(false, "SIB%d not handled yet\n", sib->SIB_type);
      }
    }
  }

  if (du->mib != NULL && du->sib1 != NULL)
    label_intra_frequency_neighbours(rrc, du, cell_info);

  f1ap_setup_resp_t resp = {.transaction_id = req->transaction_id,
                            .num_cells_to_activate = 1,
                            .cells_to_activate[0] = cell};
  int num = read_version(TO_STRING(NR_RRC_VERSION), &resp.rrc_ver[0], &resp.rrc_ver[1], &resp.rrc_ver[2]);
  AssertFatal(num == 3, "could not read RRC version string %s\n", TO_STRING(NR_RRC_VERSION));
  if (rrc->node_name != NULL)
    resp.gNB_CU_name = strdup(rrc->node_name);
  rrc->mac_rrc.f1_setup_response(assoc_id, &resp);
  free_f1ap_setup_response(&resp);

  /* we need to setup one default UE for phy-test and do-ra modes in the MAC */
  if (get_softmodem_params()->phy_test > 0 || get_softmodem_params()->do_ra > 0)
    rrc_add_nsa_user(rrc, NULL, assoc_id);
}



int CU_send_F1_SETUP_RESPONSE(sctp_assoc_t assoc_id, f1ap_setup_resp_t *f1ap_setup_resp)
{
  uint8_t  *buffer=NULL;
  uint32_t len = 0;

  /* Encode F1 Setup Response */
  F1AP_F1AP_PDU_t *pdu = encode_f1ap_setup_response(f1ap_setup_resp);
  /* Free after encode */
  free_f1ap_setup_response(f1ap_setup_resp);

  /* encode */
  if (f1ap_encode_pdu(pdu, &buffer, &len) < 0) {
    LOG_E(F1AP, "Failed to encode F1 setup response\n");
    ASN_STRUCT_FREE(asn_DEF_F1AP_F1AP_PDU, pdu);
    return -1;
  }
  ASN_STRUCT_FREE(asn_DEF_F1AP_F1AP_PDU, pdu);
  f1ap_itti_send_sctp_data_req(assoc_id, buffer, len);
  return 0;
}
```