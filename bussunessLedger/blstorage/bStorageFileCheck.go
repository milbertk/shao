package blstorage

import (
	"net/url"
	"os"
	"strings"

	"github.com/milbertk/shao/dataBasesLedger/configuration"
)

// Files are uploaded by the browser straight to Firebase Storage, so shao must
// not trust the url / storagePath it receives (2026-10-25, security S13):
//   - the storage path must be inside the company folder
//     ("companies/{companyId}/...") and cannot contain "..";
//   - the URL must be an https Firebase Storage download URL of YOUR bucket
//     (FIREBASE_STORAGE_BUCKET environment variable, or tbl_configuration
//     FIREBASE_STORAGE_BUCKET; when neither is set any Firebase bucket is
//     accepted, but the company folder is still required) and must point to
//     the same file as the storage path.

const firebaseDownloadHost = "firebasestorage.googleapis.com"

// storageBucket returns the configured Firebase bucket ("" = not configured).
func storageBucket() string {
	if bucket := strings.TrimSpace(os.Getenv("FIREBASE_STORAGE_BUCKET")); bucket != "" {
		return bucket
	}
	bucket, _ := configuration.GetParameter("FIREBASE_STORAGE_BUCKET")
	return strings.TrimSpace(bucket)
}

// companyFolder is the folder every file of the company must live in.
func companyFolder(companyID string) string {
	return "companies/" + strings.TrimSpace(companyID) + "/"
}

// ValidateCompanyStoragePath checks that a storage path belongs to the company.
func ValidateCompanyStoragePath(companyID, storagePath string) error {
	storagePath = strings.TrimSpace(storagePath)
	if strings.TrimSpace(companyID) == "" {
		return invalid("companyId is required")
	}
	if storagePath == "" {
		return nil
	}
	if !strings.HasPrefix(storagePath, companyFolder(companyID)) ||
		strings.Contains(storagePath, "..") || strings.Contains(storagePath, "\\") {
		return invalid("storagePath must be inside the company folder")
	}
	return nil
}

// ValidateCompanyFile checks the download URL and the storage path of a file
// the app uploaded (either may be empty; empty values are not checked).
func ValidateCompanyFile(companyID, storagePath, fileURL string) error {
	if err := ValidateCompanyStoragePath(companyID, storagePath); err != nil {
		return err
	}
	fileURL = strings.TrimSpace(fileURL)
	if fileURL == "" {
		return nil
	}
	objectName, ok := firebaseObjectName(fileURL, storageBucket())
	if !ok {
		return invalid("the file URL must be a Firebase Storage URL of this application")
	}
	if !strings.HasPrefix(objectName, companyFolder(companyID)) || strings.Contains(objectName, "..") {
		return invalid("the file URL must point to the company folder")
	}
	if p := strings.TrimSpace(storagePath); p != "" && p != objectName {
		return invalid("the file URL and storagePath do not match")
	}
	return nil
}

// firebaseObjectName extracts the object name from a download URL like
// https://firebasestorage.googleapis.com/v0/b/<bucket>/o/<escaped path>?alt=media&token=...
// When bucket is not empty the URL must use that bucket.
func firebaseObjectName(fileURL, bucket string) (string, bool) {
	parsed, err := url.Parse(fileURL)
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, firebaseDownloadHost) {
		return "", false
	}
	rest, found := strings.CutPrefix(parsed.Path, "/v0/b/")
	if !found {
		return "", false
	}
	urlBucket, objectName, found := strings.Cut(rest, "/o/")
	if !found || urlBucket == "" || objectName == "" {
		return "", false
	}
	if bucket != "" && !strings.EqualFold(urlBucket, bucket) {
		return "", false
	}
	return objectName, true
}
