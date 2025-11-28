package du

import (
	"github.com/JocelynWS/f1-gen/ies"
)

// HandleF1SetupRequest processes F1 Setup Request from DU
// Based on rrc_gNB_process_f1_setup_req from OAI
// Returns true if setup is accepted, false otherwise
func (du *GNBDU) HandleF1SetupRequest(setupReq *ies.F1SetupRequest) (bool, error) {
	// TODO: Type assert to actual F1SetupRequest when library is available
	// req := setupReq.(*ies.F1SetupRequest)

	// Guard: Check if assoc_id is valid
	// if du.AssocId == "" {
	// 	return false, fmt.Errorf("invalid association ID")
	// }

	// Extract information from setup request
	// TODO: When library available:
	// duId := req.GNBDUID
	// duName := string(req.GNBDUName)
	// transactionID := req.TransactionID

	// Validation checks (per duhandler.md):
	// 1. Reject if DU advertises != 1 cell
	// 2. PLMN mismatch vs CU config
	// 3. DU ID already used
	// 4. CellID/PCI clashes with existing DU

	// For now, accept all requests (validation to be implemented)
	du.State = DU_ACTIVE
	du.SetupReq = setupReq // Store deep copy

	// TODO: Decode MIB/SIB1/MTC from setup request
	// TODO: Store decoded system information

	return true, nil
}

// ValidateF1SetupRequest validates F1 Setup Request according to OAI rules
func ValidateF1SetupRequest(setupReq interface{}, cuPLMN string, existingDUs map[int64]*GNBDU) error {
	// TODO: Implement validation when F1AP library is available
	// Validation rules from duhandler.md:
	// - DU must advertise exactly 1 cell
	// - PLMN must match CU config
	// - DU ID must not be already used
	// - CellID/PCI must not clash with existing DUs

	_ = setupReq
	_ = cuPLMN
	_ = existingDUs
	return nil
}
