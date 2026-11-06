package blorderdetails

import (
	"errors"
	"strings"

	"github.com/milbertk/shao/bussunessLedger/blstorage"
	order "github.com/milbertk/shao/dataBasesLedger/orders"
	"github.com/milbertk/shao/dataBasesLedger/storage"
)

// OrderDetailsView is the whole-order Details: comments + attachments.
type OrderDetailsView struct {
	OrderInternalID string                     `json:"orderInternalId"`
	Comments        []order.OrderCommentRow    `json:"comments"`
	Attachments     []order.OrderAttachmentRow `json:"attachments"`
	Contact         *order.OrderContactRow     `json:"contact,omitempty"`
}

func BSelectOrderDetails(companyID, orderInternalID string) (*OrderDetailsView, error) {
	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" || orderInternalID == "" {
		return nil, errors.New("companyId and orderInternalId are required")
	}
	comments, err := order.SelectOrderComments(companyID, orderInternalID)
	if err != nil {
		return nil, err
	}
	attachments, err := order.SelectOrderAttachments(companyID, orderInternalID)
	if err != nil {
		return nil, err
	}
	contact, err := order.SelectOrderContact(companyID, orderInternalID)
	if err != nil {
		return nil, err
	}
	return &OrderDetailsView{
		OrderInternalID: orderInternalID,
		Comments:        comments,
		Attachments:     attachments,
		Contact:         contact,
	}, nil
}

func BAddOrderComment(companyID, orderInternalID, text, userID, email string) (*OrderDetailsView, error) {
	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" || orderInternalID == "" {
		return nil, errors.New("companyId and orderInternalId are required")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("the comment text is required")
	}
	if _, err := order.InsertOrderComment(companyID, orderInternalID, text, userID, email); err != nil {
		return nil, err
	}
	return BSelectOrderDetails(companyID, orderInternalID)
}

func BDeleteOrderComment(companyID, orderInternalID, commentID string) (*OrderDetailsView, error) {
	if strings.TrimSpace(commentID) == "" {
		return nil, errors.New("the comment id is required")
	}
	affected, err := order.DeleteOrderComment(companyID, orderInternalID, commentID)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, errors.New("comment not found")
	}
	return BSelectOrderDetails(companyID, orderInternalID)
}

// BAddOrderAttachments records already-uploaded Firebase files (URL + metadata).
func BAddOrderAttachments(companyID, orderInternalID string, files []order.OrderAttachmentRow, userID, email string) (*OrderDetailsView, error) {
	companyID = strings.TrimSpace(companyID)
	orderInternalID = strings.TrimSpace(orderInternalID)
	if companyID == "" || orderInternalID == "" {
		return nil, errors.New("companyId and orderInternalId are required")
	}
	if len(files) == 0 {
		return nil, errors.New("no attachments were provided")
	}
	// Plan storage (2026-10-23): the new files must fit.
	var newBytes int64
	for _, f := range files {
		if f.FileSize > 0 {
			newBytes += f.FileSize
		}
	}
	if err := blstorage.BCheckCanStore(companyID, newBytes); err != nil {
		return nil, err
	}
	for _, f := range files {
		if strings.TrimSpace(f.URL) == "" {
			return nil, errors.New("each attachment needs a url")
		}
		// Only files of this company in our bucket (2026-10-25, security S13).
		if err := blstorage.ValidateCompanyFile(companyID, f.StoragePath, f.URL); err != nil {
			return nil, err
		}
	}
	for _, f := range files {
		if _, err := order.InsertOrderAttachment(companyID, orderInternalID, f, userID, email); err != nil {
			return nil, err
		}
		blstorage.TrackSavedFile(companyID, userID, f.StoragePath, f.FileSize, storage.KindOrderAttachment)
	}
	return BSelectOrderDetails(companyID, orderInternalID)
}

// BDeleteOrderAttachment removes the DB row and returns the storage path so the
// handler can delete the Firebase object.
func BDeleteOrderAttachment(companyID, orderInternalID, attachmentID string) (*OrderDetailsView, string, error) {
	if strings.TrimSpace(attachmentID) == "" {
		return nil, "", errors.New("the attachment id is required")
	}
	storagePath, affected, err := order.DeleteOrderAttachment(companyID, orderInternalID, attachmentID)
	if err != nil {
		return nil, "", err
	}
	if affected == 0 {
		return nil, "", errors.New("attachment not found")
	}
	blstorage.ForgetSavedFile(companyID, storagePath)
	view, err := BSelectOrderDetails(companyID, orderInternalID)
	if err != nil {
		return nil, storagePath, err
	}
	return view, storagePath, nil
}
