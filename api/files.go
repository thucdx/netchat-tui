package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// GetFileInfo returns metadata for the given file ID.
func (c *Client) GetFileInfo(fileID string) (FileInfo, error) {
	path := "/api/v4/files/" + url.PathEscape(fileID) + "/info"
	data, err := c.Get(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("GetFileInfo: %w", err)
	}
	var fi FileInfo
	if err := json.Unmarshal(data, &fi); err != nil {
		return FileInfo{}, fmt.Errorf("GetFileInfo: %w", err)
	}
	return fi, nil
}

// DownloadFileThumbnail returns the raw bytes of the thumbnail for an image file.
// Returns an error for non-image files (Mattermost returns 404 in that case).
func (c *Client) DownloadFileThumbnail(fileID string) ([]byte, error) {
	path := "/api/v4/files/" + url.PathEscape(fileID) + "/thumbnail"
	data, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("DownloadFileThumbnail: %w", err)
	}
	return data, nil
}

// DownloadFile returns the raw bytes of the full file.
func (c *Client) DownloadFile(fileID string) ([]byte, error) {
	path := "/api/v4/files/" + url.PathEscape(fileID)
	data, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("DownloadFile: %w", err)
	}
	return data, nil
}
