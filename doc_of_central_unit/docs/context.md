CU-CP contexts in OAI (structures and fields, OAI names kept)

1) gNB_RRC_INST (openair2/RRC/NR/nr_rrc_defs.h:354+)
```
typedef struct gNB_RRC_INST_s {
  ngran_node_t          node_type;
  uint32_t              node_id;
  char                 *node_name;
  int                   module_id;
  eth_params_t          eth_params_s;
  uid_allocator_t       uid_allocator;
  RB_HEAD(rrc_nr_ue_tree_s, rrc_gNB_ue_context_s) rrc_ue_head;
  uint64_t              nr_cellid;
  gNB_RrcConfigurationReq configuration;
  seq_arr_t            *SIBs;
  instance_t            e1_inst;
  char                 *uecap_file;
  nr_security_configuration_t security;
  nr_mac_rrc_dl_if_t    mac_rrc;
  cucp_cuup_if_t        cucp_cuup;
  seq_arr_t            *neighbour_cell_configuration;
  nr_measurement_configuration_t measurementConfiguration;
  RB_HEAD(rrc_du_tree, nr_rrc_du_container_t) dus;
  size_t                num_dus;
  RB_HEAD(rrc_cuup_tree, nr_rrc_cuup_container_t) cuups;
  size_t                num_cuups;
  nr_pdcp_configuration_t pdcp_config;
  nr_rlc_configuration_t  rlc_config;
} gNB_RRC_INST;
```

2) rrc_gNB_UE_context_t / gNB_RRC_UE_t (openair2/RRC/NR/nr_rrc_defs.h:113+)
```
typedef struct gNB_RRC_UE_s {
  time_t                  last_seen;
  NR_DRB_ToReleaseList_t *DRB_ReleaseList;
  NR_SRB_INFO_TABLE_ENTRY Srb[NR_NUM_SRB];
  NR_MeasConfig_t        *measConfig;
  nr_handover_context_t  *ho_context;
  NR_MeasResults_t       *measResults;
  bool                    as_security_active;
  bool                    f1_ue_context_active;
  byte_array_t            ue_cap_buffer;
  NR_UE_NR_Capability_t  *UE_Capability_nr;
  int                     UE_Capability_size;
  NR_UE_MRDC_Capability_t *UE_Capability_MRDC;
  int                     UE_MRDC_Capability_size;
  NR_CellGroupConfig_t   *masterCellGroup;
  NR_RadioBearerConfig_t *rb_config;
  uint8_t                 kgnb[32];
  int8_t                  kgnb_ncc;
  uint8_t                 nh[32];
  int8_t                  nh_ncc;
  NR_CipheringAlgorithm_t ciphering_algorithm;
  e_NR_IntegrityProtAlgorithm integrity_algorithm;
  rnti_t                  rnti;
  uint64_t                random_ue_identity;
  NR_UE_S_TMSI            Initialue_identity_5g_s_TMSI;
  uint64_t                ng_5G_S_TMSI_Part1;
  NR_EstablishmentCause_t establishment_cause;
  uint64_t                nr_cellid;
  uint32_t                rrc_ue_id;
  uint64_t                amf_ue_ngap_id;
  nr_guami_t              ue_guami;
  ngap_security_capabilities_t security_capabilities;
  sctp_assoc_t            x2_target_assoc;
  int                     MeNB_ue_x2_id;
  int                     nb_of_e_rabs;
  nr_e_rab_param_t        e_rab[NB_RB_MAX];
  uint32_t                nsa_gtp_teid[S1AP_MAX_E_RAB];
  transport_layer_addr_t  nsa_gtp_addrs[S1AP_MAX_E_RAB];
  rb_id_t                 nsa_gtp_ebi[S1AP_MAX_E_RAB];
  rb_id_t                 nsa_gtp_psi[S1AP_MAX_E_RAB];
  seq_arr_t               pduSessions;
  seq_arr_t               drbs;
  rrc_action_t            xids[NR_RRC_TRANSACTION_IDENTIFIER_NUMBER];
  uint8_t                 e_rab_release_command_flag;
  uint32_t                ue_rrc_inactivity_timer;
  uint32_t                ue_reestablishment_counter;
  uint32_t                ue_reconfiguration_counter;
  bool                    ongoing_reconfiguration;
  bool                    an_release;
  int                     n_initial_pdu;
  pdusession_t           *initial_pdus;
  byte_array_t            nas_pdu;
  int                     max_delays_pdu_session;
  bool                    ongoing_pdusession_setup_request;
  nr_redcap_ue_cap_t     *redcap_cap;
} gNB_RRC_UE_t;
```
Wrapper:
```
typedef struct rrc_gNB_ue_context_s {
  RB_ENTRY(rrc_gNB_ue_context_s) entries;
  struct gNB_RRC_UE_s            ue_context;
} rrc_gNB_ue_context_t;
```

Supporting types (nr_rrc_defs.h:90+)
```
typedef enum pdu_session_satus_e { PDU_SESSION_STATUS_NEW, PDU_SESSION_STATUS_DONE,
  PDU_SESSION_STATUS_ESTABLISHED, PDU_SESSION_STATUS_REESTABLISHED,
  PDU_SESSION_STATUS_TOMODIFY, PDU_SESSION_STATUS_FAILED,
  PDU_SESSION_STATUS_TORELEASE, PDU_SESSION_STATUS_RELEASED } pdu_session_status_t;

typedef struct pdusession_s {
  int                   pdusession_id;
  byte_array_t          nas_pdu;
  seq_arr_t             qos;
  pdu_session_type_t    pdu_session_type;
  gtpu_tunnel_t         n3_outgoing;
  gtpu_tunnel_t         n3_incoming;
  nssai_t               nssai;
  nr_sdap_configuration_t sdap_config;
} pdusession_t;

typedef struct pdu_session_param_s {
  pdusession_t       param;
  pdu_session_status_t status;
  uint8_t            xid;
  ngap_cause_t       cause;
} rrc_pdu_session_param_t;

typedef struct drb_s {
  int                    status;
  int                    drb_id;
  int                    pdusession_id;
  gtpu_tunnel_t          du_tunnel_config;
  gtpu_tunnel_t          cuup_tunnel_config;
  nr_pdcp_configuration_t pdcp_config;
} drb_t;
```

3) DU context (nr_rrc_defs.h:336+; rrc_gNB_du.c)
```
typedef struct nr_rrc_du_container_t {
  RB_ENTRY(nr_rrc_du_container_t) entries;
  sctp_assoc_t            assoc_id;
  f1ap_setup_req_t       *setup_req;
  NR_MIB_t               *mib;
  NR_SIB1_t              *sib1;
  NR_MeasurementTimingConfiguration_t *mtc;
} nr_rrc_du_container_t;
```

4) CU-UP context (nr_rrc_defs.h:346+; rrc_gNB_cuup.c)
```
typedef struct nr_rrc_cuup_container_t {
  RB_ENTRY(nr_rrc_cuup_container_t) entries;
  e1ap_setup_req_t      *setup_req;
  sctp_assoc_t           assoc_id;
} nr_rrc_cuup_container_t;
```

5) F1 UE mapping (openair2/F1AP/f1ap_ids.c)
```
typedef struct {
  uint32_t     secondary_ue;
  sctp_assoc_t du_assoc_id;
  sctp_assoc_t e1_assoc_id;
} f1_ue_data_t;
```
Helpers: cu_add/update/get/remove_f1_ue_data(), stored in hashtable.

6) NGAP AMF association (openair3/NGAP/ngap_gNB_defs.h:63+)
```
typedef struct ngap_gNB_amf_data_s {
  RB_ENTRY(ngap_gNB_amf_data_s) entry;
  char                *amf_name;
  net_ip_address_t     amf_s1_ip;
  struct served_guamis_s served_guami;
  struct plmn_supports_s plmn_supports;
  uint8_t              relative_amf_capacity;
  ngap_overload_state_t overload_state;
  ngap_gNB_state_t     state;
  int32_t              nextstream;
  uint16_t             in_streams;
  uint16_t             out_streams;
  uint16_t             cnx_id;
  sctp_assoc_t         assoc_id;
  uint8_t              broadcast_plmn_num;
  uint8_t              broadcast_plmn_index[PLMN_LIST_MAX_SIZE];
  struct ngap_gNB_instance_s *ngap_gNB_instance;
} ngap_gNB_amf_data_t;
```

Supporting NGAP lists (ngap_gNB_defs.h):
- served_guami_s contains lists of plmn_identity_s, served_region_id_s, amf_set_id_s, amf_pointer_s.
- plmn_support_s contains plmn_identity + slice_support list.

7) NGAP instance (ngap_gNB_defs.h:96+)
```
typedef struct ngap_gNB_instance_s {
  STAILQ_ENTRY(ngap_gNB_instance_s) ngap_gNB_entries;
  uint32_t ngap_amf_nb;
  uint32_t ngap_amf_pending_nb;
  uint32_t ngap_amf_associated_nb;
  RB_HEAD(ngap_amf_map, ngap_gNB_amf_data_s) ngap_amf_head;
  instance_t instance;
  char      *gNB_name;
  uint32_t   gNB_id;
  enum cell_type_e cell_type;
  uint32_t   tac;
  net_ip_address_t gNB_ng_ip;
  uint8_t    num_plmn;
  ngap_plmn_t plmn[PLMN_LIST_MAX_SIZE];
  ngap_paging_drx_t default_drx;
} ngap_gNB_instance_t;
```

8) NGAP UE context (ngap_gNB_ue_context.h:30+)
```
typedef struct ngap_gNB_ue_context_s {
  RB_ENTRY(ngap_gNB_ue_context_s) entries;
  uint32_t         gNB_ue_ngap_id;
  uint64_t         amf_ue_ngap_id;
  int32_t          tx_stream;
  int32_t          rx_stream;
  ngap_ue_state    ue_state;
  struct ngap_gNB_amf_data_s *amf_ref;
  plmn_id_t        selected_plmn_identity;
  ngap_gNB_instance_t *gNB_instance;
} ngap_gNB_ue_context_t;
```

9) F1 endpoint state (f1ap_cudu_inst_t in openair2/F1AP/f1ap_common.h:414+)
```
typedef struct f1ap_cudu_inst_s {
  f1ap_setup_req_t setupReq;
  f1ap_net_config_t net_config;
  struct { sctp_assoc_t assoc_id; } du;
  uint16_t sctp_in_streams;
  uint16_t sctp_out_streams;
  instance_t gtpInst;
} f1ap_cudu_inst_t;
```

Key supporting types (OAI names preserved)
- nssai_t (common/5g_platform_types.h:34)
  ```
  typedef struct nssai_s { uint8_t sst; uint32_t sd; } nssai_t;
  ```
- nr_guami_t (common/5g_platform_types.h:38)
  ```
  typedef struct nr_guami_s {
    uint16_t mcc; uint16_t mnc; uint8_t mnc_len;
    uint8_t amf_region_id; uint16_t amf_set_id; uint8_t amf_pointer;
  } nr_guami_t;
  ```
- gtpu_tunnel_t (common/platform_types.h:235)
  ```
  typedef struct { uint32_t teid; transport_layer_addr_t addr; } gtpu_tunnel_t;
  ```
- transport_layer_addr_t (common/platform_types.h:228)
  ```
  typedef struct { uint8_t length; uint8_t buffer[20]; } transport_layer_addr_t;
  ```
- nr_sdap_configuration_t (openair2/SDAP/nr_sdap/nr_sdap_configuration.h)
  ```
  typedef struct { bool header_dl_absent; bool header_ul_absent; } nr_sdap_configuration_t;
  ```
- nr_pdcp_configuration_t (openair2/LAYER2/nr_pdcp/nr_pdcp_configuration.h:62)
  ```
  typedef struct { struct { int sn_size; int t_reordering; int discard_timer; } drb; } nr_pdcp_configuration_t;
  ```
- nr_rlc_configuration_t (openair2/LAYER2/nr_rlc/nr_rlc_configuration.h:150)
  ```
  typedef struct {
    struct { int t_poll_retransmit; int t_reassembly; int t_status_prohibit;
             int poll_pdu; int poll_byte; int max_retx_threshold; int sn_field_length; } drb_am;
    struct { int t_reassembly; int sn_field_length; } drb_um;
  } nr_rlc_configuration_t;
  ```
- nr_security_configuration_t (openair2/RRC/NR/nr_rrc_defs.h:270)
  ```
  typedef struct {
    int ciphering_algorithms[4]; int ciphering_algorithms_count;
    int integrity_algorithms[4]; int integrity_algorithms_count;
    int do_drb_ciphering; int do_drb_integrity;
  } nr_security_configuration_t;
  ```
