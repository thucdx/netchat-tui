package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// GetCustomEmojiList fetches one page of custom emojis from the server.
// page is 0-indexed; perPage is the number of results per page (max 200).
// Returns an empty slice (no error) when there are no more results.
func (c *Client) GetCustomEmojiList(page, perPage int) ([]CustomEmoji, error) {
	path := fmt.Sprintf("/api/v4/emoji?page=%d&per_page=%d&sort=name", page, perPage)
	data, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("GetCustomEmojiList: %w", err)
	}
	var emojis []CustomEmoji
	if err := json.Unmarshal(data, &emojis); err != nil {
		return nil, fmt.Errorf("GetCustomEmojiList: %w", err)
	}
	return emojis, nil
}

// GetCustomEmojiImage downloads the raw image bytes for a custom emoji by ID.
func (c *Client) GetCustomEmojiImage(emojiID string) ([]byte, error) {
	path := "/api/v4/emoji/" + url.PathEscape(emojiID) + "/image"
	data, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("GetCustomEmojiImage: %w", err)
	}
	return data, nil
}
