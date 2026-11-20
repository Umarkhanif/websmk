package state

import (
	"errors"
	"testing"
	"time"
)

const documentState = `{
  "studentDocuments": [{"id": "doc-1", "studentId": "std-1", "status": "MENUNGGU_CEK"}]
}`

func TestUploadStudentDocumentRequiresServerBlob(t *testing.T) {
	doc := mustDoc(t, documentState)
	err := UploadStudentDocument(doc, "std-1", "doc-1", DocumentUpload{FileName: "kk.pdf"}, time.Now())
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid when blobId/size missing", err)
	}
}

func TestUploadStudentDocumentStoresValidatedMetadata(t *testing.T) {
	doc := mustDoc(t, documentState)
	upload := DocumentUpload{
		FileName:  "kk.pdf",
		MimeType:  "application/pdf",
		BlobID:    "11111111-2222-3333-4444-555555555555",
		SizeBytes: 2048,
		MagicHex:  "25504446",
	}
	if err := UploadStudentDocument(doc, "std-1", "doc-1", upload, time.Now()); err != nil {
		t.Fatalf("upload: %v", err)
	}
	record := asMap(asList(doc["studentDocuments"])[0])
	if text(record["blobId"]) != upload.BlobID {
		t.Fatalf("blobId = %q, want %q", text(record["blobId"]), upload.BlobID)
	}
	if text(record["status"]) != "MENUNGGU_CEK" {
		t.Fatalf("status = %q, want MENUNGGU_CEK", text(record["status"]))
	}
	size, _ := number(record["fileSizeKb"])
	if size != 2 {
		t.Fatalf("fileSizeKb = %v, want 2", size)
	}
	if text(record["mimeHeaderHex"]) != "25504446" {
		t.Fatalf("magic = %q", text(record["mimeHeaderHex"]))
	}
}

func TestUploadStudentDocumentRejectsForeignRecord(t *testing.T) {
	doc := mustDoc(t, documentState)
	upload := DocumentUpload{BlobID: "blob-1", SizeBytes: 100}
	if err := UploadStudentDocument(doc, "std-2", "doc-1", upload, time.Now()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}
