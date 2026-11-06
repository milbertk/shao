package blreports

import (
	"errors"
	"strconv"
	"strings"

	reports "github.com/milbertk/shao/dataBasesLedger/reportes"
)

// BAgingOrders — open orders (not done/cancelled), oldest first, with age.
//
//	statusKey: "all" or a frontend order key (placed/inProgress/partialDone).
//	minDaysStr: optional "?minDays=" value; "" means no minimum.
func BAgingOrders(companyID, statusKey, minDaysStr string) (*reports.AgingOrdersReport, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	if statusKey == "all" {
		statusKey = ""
	}
	minDays := 0
	if s := strings.TrimSpace(minDaysStr); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 {
			return nil, errors.New("invalid minDays: must be a non-negative number")
		}
		minDays = n
	}
	return reports.AgingOrders(companyID, statusKey, minDays)
}
