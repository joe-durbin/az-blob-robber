package azure

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	httpClient  *http.Client
	UserAgent   string
	AccessToken string    // Optional bearer token for authenticated requests
	DebugWriter io.Writer // Writer for curl command logging (if nil, logging is disabled)
}

const DefaultUserAgent = "az-blob-robber/1.0"

func NewClient() *Client {
	return NewClientWithToken("", nil, DefaultUserAgent)
}

func NewClientWithToken(token string, debugWriter io.Writer, userAgent string) *Client {
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	return &Client{
		httpClient: &http.Client{
			// Timeout: 0, // No timeout for large downloads
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
			},
		},
		UserAgent:   userAgent,
		AccessToken: token,
		DebugWriter: debugWriter,
	}
}

// logCurlCommand outputs a curl equivalent command to the debug writer
func (c *Client) logCurlCommand(req *http.Request) {
	if c.DebugWriter == nil {
		return
	}

	fmt.Fprintf(c.DebugWriter, "curl -X %s", req.Method)

	// Add headers
	for key, values := range req.Header {
		for _, value := range values {
			fmt.Fprintf(c.DebugWriter, " -H '%s: %s'", key, value)
		}
	}

	fmt.Fprintf(c.DebugWriter, " '%s'\n", req.URL.String())
}

// setAuthHeaders adds authentication and version headers to a request
func (c *Client) setAuthHeaders(req *http.Request) {
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("x-ms-version", "2019-12-12") // Older version works better for deleted blobs
	if c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	}
}

// CheckAccount checks if a storage account exists by trying to resolve its blob endpoint
// or making a root request.
// Returns true if it exists (even if private).
func (c *Client) CheckAccount(accountName string) bool {
	// A simple way is to check a standard endpoint.
	// We can try to HEAD the blob service root.
	// https://<account>.blob.core.windows.net/
	u := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	req, err := http.NewRequest("HEAD", u, nil)
	if err != nil {
		return false
	}
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// DNS error usually means it doesn't exist
		return false
	}
	defer resp.Body.Close()

	// If we get any response (even 400, 403, 404), the account exists (DNS resolved).
	c.logCurlCommand(req)
	return true
}

// CheckContainer checks if a container exists and is accessible.
// Returns:
// exists: true if we get a response indicating existence (200, 403, etc)
// public: true if we can list it (200)
func (c *Client) CheckContainer(accountName, containerName string) (exists bool, public bool) {
	u := fmt.Sprintf("https://%s.blob.core.windows.net/%s?restype=container&comp=list&maxresults=1",
		accountName, url.PathEscape(containerName))
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return false, false
	}
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		c.logCurlCommand(req)
		return true, true
	}
	// 403 means it exists but is private.
	// 404 means it doesn't exist.
	if resp.StatusCode == 403 {
		return true, false
	}

	return false, false
}

// ListBlobs returns a list of blobs in a container.
// It attempts to include versions first. If that fails, it falls back to a standard list.
func (c *Client) ListBlobs(accountName, containerName string) ([]Blob, error) {
	// Try with include=versions first (works better with deleted blobs on older API version)
	blobs, err := c.listBlobsInternal(accountName, containerName, "versions")
	if err == nil {
		return blobs, nil
	}
	// Fallback to standard list
	return c.listBlobsInternal(accountName, containerName, "")
}

func (c *Client) listBlobsInternal(accountName, containerName, include string) ([]Blob, error) {
	var allBlobs []Blob
	marker := ""

	for {
		u := fmt.Sprintf("https://%s.blob.core.windows.net/%s?restype=container&comp=list",
			accountName, url.PathEscape(containerName))
		if include != "" {
			u += "&include=" + include
		}
		if marker != "" {
			u += "&marker=" + url.QueryEscape(marker)
		}

		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}
		c.setAuthHeaders(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("failed to list blobs: status %d", resp.StatusCode)
		}

		c.logCurlCommand(req)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var result EnumerationResults
		if err := xml.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		allBlobs = append(allBlobs, result.Blobs.Blob...)

		marker = result.NextMarker
		if marker == "" {
			break
		}
	}

	return allBlobs, nil
}

// GetBlobVersions fetches all versions of a specific blob.
func (c *Client) GetBlobVersions(accountName, containerName, blobName string) ([]Blob, error) {
	// We list blobs with prefix=blobName and include=versions
	// This ensures we get all history for this specific blob.
	u := fmt.Sprintf("https://%s.blob.core.windows.net/%s?restype=container&comp=list&include=versions&prefix=%s",
		accountName, url.PathEscape(containerName), url.QueryEscape(blobName))

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get versions: status %d", resp.StatusCode)
	}

	c.logCurlCommand(req)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result EnumerationResults
	if err := xml.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Filter to ensure exact name match (prefix might match others)
	var versions []Blob
	for _, b := range result.Blobs.Blob {
		if b.Name == blobName {
			versions = append(versions, b)
		}
	}

	return versions, nil
}

// DownloadBlob downloads a blob from Azure Storage.
// accountName, containerName, and blobName are URL-encoded automatically.
// snapshotOrVersion can be a version ID (RFC3339 format) or snapshot timestamp.
func (c *Client) DownloadBlob(accountName, containerName, blobName, snapshotOrVersion string) (io.ReadCloser, error) {
	// URL encode path segments properly
	u := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		accountName,
		url.PathEscape(containerName),
		url.PathEscape(blobName))

	// Handle snapshots/versions
	// Snapshots use ?snapshot=<timestamp>
	// Versions use ?versionId=<timestamp>
	// Version IDs are RFC3339 format (e.g., 2025-08-07T21:08:03.6678148Z)
	// We detect version IDs by checking for the 'T' separator in ISO8601 format
	if snapshotOrVersion != "" {
		// Check if it looks like a version ID (RFC3339 format with 'T' separator)
		// Version IDs typically have 'T' around position 10 (YYYY-MM-DDT...)
		if len(snapshotOrVersion) > 10 && snapshotOrVersion[10] == 'T' {
			u += "?versionId=" + url.QueryEscape(snapshotOrVersion)
		} else {
			u += "?snapshot=" + url.QueryEscape(snapshotOrVersion)
		}
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download: %d", resp.StatusCode)
	}

	c.logCurlCommand(req)
	return resp.Body, nil
}
