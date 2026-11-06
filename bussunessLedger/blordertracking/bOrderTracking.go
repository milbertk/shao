package blordertracking

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	blnotifications "github.com/milbertk/shao/bussunessLedger/blnotifications"
	"github.com/milbertk/shao/bussunessLedger/blstorage"
	ordertracking "github.com/milbertk/shao/dataBasesLedger/orderTracking"
	"github.com/milbertk/shao/dataBasesLedger/storage"
)

// ErrStageNotFound distinguishes a missing stage.
var ErrStageNotFound = errors.New("order tracking stage not found")

// StageBundle groups a stage with its comments + attachments.
type StageBundle struct {
	Stage       ordertracking.OrderTrackingStageS        `json:"stage"`
	Comments    []ordertracking.OrderTrackingCommentS    `json:"comments"`
	Attachments []ordertracking.OrderTrackingAttachmentS `json:"attachments"`
}

// OrderTrackingView is the full view for one order: all its stages (each with
// its comments + attachments).
type OrderTrackingView struct {
	OrderInternalID string        `json:"orderInternalId"`
	Stages          []StageBundle `json:"stages"`
}

// BSelectStageFull returns one stage + its children.
func BSelectStageFull(companyID, stageID string) (*StageBundle, error) {
	companyID = strings.TrimSpace(companyID)
	stageID = strings.TrimSpace(stageID)
	if companyID == "" || stageID == "" {
		return nil, errors.New("companyId and stageId are required")
	}

	stage, err := ordertracking.SelectStageByID(companyID, stageID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStageNotFound
		}
		return nil, fmt.Errorf("failed to select stage: %w", err)
	}
	comments, err := ordertracking.SelectCommentsByStage(companyID, stageID)
	if err != nil {
		return nil, err
	}
	attachments, err := ordertracking.SelectAttachmentsByStage(companyID, stageID)
	if err != nil {
		return nil, err
	}
	return &StageBundle{Stage: *stage, Comments: comments, Attachments: attachments}, nil
}

// BSelectOrderTracking returns all stages of an order, each with its children.
func BSelectOrderTracking(companyID, orderInternalID string) (*OrderTrackingView, error) {
	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" || orderInternalID == "" {
		return nil, errors.New("companyId and orderInternalId are required")
	}

	stages, err := ordertracking.SelectStagesByOrder(companyID, orderInternalID)
	if err != nil {
		return nil, err
	}

	view := &OrderTrackingView{OrderInternalID: orderInternalID, Stages: make([]StageBundle, 0, len(stages))}
	for i := range stages {
		st := stages[i]
		comments, err := ordertracking.SelectCommentsByStage(companyID, st.OrderTrackingID)
		if err != nil {
			return nil, err
		}
		attachments, err := ordertracking.SelectAttachmentsByStage(companyID, st.OrderTrackingID)
		if err != nil {
			return nil, err
		}
		view.Stages = append(view.Stages, StageBundle{Stage: st, Comments: comments, Attachments: attachments})
	}
	return view, nil
}

// BSaveStage creates or updates one stage (create if no id, update if id exists),
// rewrites its comments + attachments, and then SYNCS the order's status to the
// most recent stage's status.
func BSaveStage(companyID string, input StageBundle) (*StageBundle, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, errors.New("companyId is required")
	}
	input.Stage.CompanyID = companyID
	if strings.TrimSpace(input.Stage.OrderInternalID) == "" {
		return nil, errors.New("orderInternalId is required")
	}
	if strings.TrimSpace(input.Stage.Stage) == "" {
		return nil, errors.New("stage is required")
	}

	// Order status before this stage (to notify only real changes).
	previousStatus := blnotifications.CurrentOrderStatus(companyID, input.Stage.OrderInternalID)

	stageID := strings.TrimSpace(input.Stage.OrderTrackingID)
	stageCreated := stageID == "" // a new stage is announced to the customer (2026-10-30)
	if stageID == "" {
		if err := input.Stage.Insert(); err != nil {
			return nil, fmt.Errorf("failed to create stage: %w", err)
		}
	} else {
		_, err := ordertracking.SelectStageByID(companyID, stageID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				input.Stage.OrderTrackingID = ""
				stageCreated = true
				if err := input.Stage.Insert(); err != nil {
					return nil, fmt.Errorf("failed to create stage: %w", err)
				}
			} else {
				return nil, fmt.Errorf("failed to validate stage: %w", err)
			}
		} else {
			if err := input.Stage.Update(); err != nil {
				return nil, fmt.Errorf("failed to update stage: %w", err)
			}
		}
	}

	newStageID := input.Stage.OrderTrackingID

	// Rewrite comments.
	if err := ordertracking.DeleteCommentsByStage(companyID, newStageID); err != nil {
		return nil, fmt.Errorf("failed to clear stage comments: %w", err)
	}
	for i := range input.Comments {
		c := &input.Comments[i]
		c.OrderTrackingID = newStageID
		c.CompanyID = companyID
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		if c.CommentOrder < 1 {
			c.CommentOrder = i + 1
		}
		if err := c.Insert(); err != nil {
			return nil, fmt.Errorf("failed to insert stage comment: %w", err)
		}
	}

	// Rewrite attachments (Firebase refs).
	if err := ordertracking.DeleteAttachmentsByStage(companyID, newStageID); err != nil {
		return nil, fmt.Errorf("failed to clear stage attachments: %w", err)
	}
	for i := range input.Attachments {
		a := &input.Attachments[i]
		a.OrderTrackingID = newStageID
		a.CompanyID = companyID
		if strings.TrimSpace(a.URL) == "" {
			continue
		}
		// Only files of this company in our bucket (2026-10-25, security S13).
		path := ""
		if a.StoragePath != nil {
			path = *a.StoragePath
		}
		if err := blstorage.ValidateCompanyFile(companyID, path, a.URL); err != nil {
			return nil, err
		}
		if err := a.Insert(); err != nil {
			return nil, fmt.Errorf("failed to insert stage attachment: %w", err)
		}
		// Count it in the company storage (2026-10-23; never blocks the stage).
		if a.StoragePath != nil {
			blstorage.TrackSavedFile(companyID, "", *a.StoragePath, a.FileSize, storage.KindTrackingAttachment)
		}
	}

	// AUTO-SYNC: set the order's status to the most recent stage's status.
	recentStatus, err := ordertracking.MostRecentStageStatus(companyID, input.Stage.OrderInternalID)
	if err == nil && recentStatus != "" {
		_ = ordertracking.UpdateOrderStatus(companyID, input.Stage.OrderInternalID, recentStatus)
	}
	// Tell the customer (2026-10-30): a new stage (name + status, one email),
	// otherwise only a real order status change.
	switch {
	case stageCreated:
		blnotifications.NotifyOrderStage(companyID, input.Stage.OrderInternalID, input.Stage.Stage, input.Stage.Status)
	case err == nil && recentStatus != "" && previousStatus != "" && !strings.EqualFold(previousStatus, recentStatus):
		blnotifications.NotifyOrder(companyID, input.Stage.OrderInternalID, blnotifications.EventStatusChanged)
	}

	return BSelectStageFull(companyID, newStageID)
}
