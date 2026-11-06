package ordertrackinginshao

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/milbertk/security"
	"github.com/milbertk/tracking"

	blordertracking "github.com/milbertk/shao/bussunessLedger/blordertracking"
)

// HSaveOrderTrackingStage creates/updates a stage (with its comments +
// attachments) and auto-syncs the order status. AUTHENTICATED + login tracking.
//
//	POST /companies/{companyId}/order-trackings/stages
func HSaveOrderTrackingStage(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HSaveOrderTrackingStage] Handler reached: method=%s path=%s", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeResponse(w, http.StatusMethodNotAllowed, "Use POST", nil)
		return
	}

	claims, ok := security.GetClaims(r)
	if !ok || claims == nil || strings.TrimSpace(claims.UID) == "" {
		writeResponse(w, http.StatusUnauthorized, "Missing user claims", nil)
		return
	}
	actorUserID := strings.TrimSpace(claims.UID)

	companyID, ok := security.GetActiveCompanyID(r)
	companyID = strings.TrimSpace(companyID)
	if !ok || companyID == "" {
		writeResponse(w, http.StatusForbidden, "Missing validated company context", nil)
		return
	}

	actorEmail, _ := security.GetActiveCompanyEmail(r)
	actorRole, _ := security.GetActiveCompanyRole(r)

	bundle, err := decodeSingleStageBundle(r)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	wasUpdate := strings.TrimSpace(bundle.Stage.OrderTrackingID) != ""

	result, err := blordertracking.BSaveStage(companyID, bundle)
	if err != nil {
		status := errorStatus(err)
		message := err.Error()
		if status == http.StatusInternalServerError {
			message = "An internal error occurred while saving the stage"
		}
		log.Printf("[HSaveOrderTrackingStage] Failed: status=%d error=%v", status, err)
		writeResponse(w, status, message, nil)
		return
	}

	insertStageTracking(r, actorUserID, actorEmail, companyID, actorRole, wasUpdate, result)

	log.Printf("[HSaveOrderTrackingStage] Saved OK: stageId=%s status=%s",
		result.Stage.OrderTrackingID, result.Stage.Status)
	writeResponse(w, http.StatusOK, "Stage saved successfully.", result)
}

func insertStageTracking(
	r *http.Request,
	actorUserID, actorEmail, activeCompanyID, activeCompanyRole string,
	wasUpdate bool,
	result *blordertracking.StageBundle,
) {
	actionName := "CreateOrderTrackingStage"
	if wasUpdate {
		actionName = "UpdateOrderTrackingStage"
	}

	payload := struct {
		CompanyID        string                        `json:"companyId"`
		ActorCompanyRole string                        `json:"actorCompanyRole"`
		Result           *blordertracking.StageBundle  `json:"result"`
	}{
		CompanyID:        activeCompanyID,
		ActorCompanyRole: activeCompanyRole,
		Result:           result,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[HSaveOrderTrackingStage] Error creating tracking payload: %v", err)
		return
	}

	lt := tracking.LoginTracking{
		UserID:         actorUserID,
		Email:          strings.TrimSpace(actorEmail),
		DateLocalAcces: time.Now().Format("2006-01-02 15:04:05"),
		IP:             readClientIP(r),
		Platform:       "Shao",
		MacAddress:     "0",
		Browser:        r.UserAgent(),
		CountryCode:    "",
		GMTTime:        time.Now().UTC().Format("2006-01-02 15:04:05"),
		Lang:           r.Header.Get("Accept-Language"),
		Action:         actionName,
		Jsonstring:     string(jsonPayload),
	}
	if err := lt.Insert(); err != nil {
		log.Printf("[HSaveOrderTrackingStage] Error inserting tracking: %v", err)
	}
}
