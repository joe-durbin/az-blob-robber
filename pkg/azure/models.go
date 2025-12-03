package azure

import "encoding/xml"

// EnumerationResults is the top-level XML response for listing blobs.
type EnumerationResults struct {
	XMLName       xml.Name `xml:"EnumerationResults"`
	ContainerName string   `xml:"ContainerName,attr"`
	Blobs         Blobs    `xml:"Blobs"`
	NextMarker    string   `xml:"NextMarker"`
}

type Blobs struct {
	Blob []Blob `xml:"Blob"`
}

type Blob struct {
	Name             string     `xml:"Name"`
	Snapshot         string     `xml:"Snapshot"`
	VersionId        string     `xml:"VersionId"`
	IsCurrentVersion *bool      `xml:"IsCurrentVersion"` // Pointer to detect absence
	Properties       Properties `xml:"Properties"`
	Url              string     `xml:"Url"`
	Deleted          bool       `xml:"Deleted"` // Try at top level first
}

type Properties struct {
	LastModified           string `xml:"Last-Modified"`
	Etag                   string `xml:"Etag"`
	ContentLength          int64  `xml:"Content-Length"`
	ContentType            string `xml:"Content-Type"`
	BlobType               string `xml:"BlobType"`
	LeaseStatus            string `xml:"LeaseStatus"`
	CreationTime           string `xml:"Creation-Time"`
	DeletedTime            string `xml:"DeletedTime"`            // Azure puts this in Properties
	RemainingRetentionDays int    `xml:"RemainingRetentionDays"` // Indicates soft-deleted
}

// IsDeleted checks if the blob is marked as deleted
// Azure can indicate deletion in multiple ways
func (b *Blob) IsDeleted() bool {
	// Check if Deleted field is set at blob level
	if b.Deleted {
		return true
	}
	// Check if DeletedTime exists in Properties (soft-delete indicator)
	if b.Properties.DeletedTime != "" {
		return true
	}
	// Check if this is a versioned blob without IsCurrentVersion
	// (means the current version has been deleted)
	if b.VersionId != "" && (b.IsCurrentVersion == nil || !*b.IsCurrentVersion) {
		return true
	}
	return false
}

// ContainerEnumerationResults is the XML response for listing containers (if we were to support that, though usually we brute force names).
// But for brute forcing, we just check 200 vs 404.
