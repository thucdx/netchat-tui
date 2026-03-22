package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// GetPostsForChannel returns a page of posts for the given channel.
// page is 0-indexed; perPage is typically 30 or 60.
func (c *Client) GetPostsForChannel(channelID string, page, perPage int) (PostList, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))

	path := "/api/v4/channels/" + url.PathEscape(channelID) + "/posts?" + q.Encode()

	data, err := c.Get(path)
	if err != nil {
		return PostList{}, fmt.Errorf("GetPostsForChannel: %w", err)
	}

	var pl PostList
	if err := json.Unmarshal(data, &pl); err != nil {
		return PostList{}, fmt.Errorf("GetPostsForChannel: %w", err)
	}

	return pl, nil
}

// GetPostsSince returns all posts in the channel created after the given
// Unix millisecond timestamp. Used for incremental refresh.
func (c *Client) GetPostsSince(channelID string, since int64) (PostList, error) {
	if since <= 0 {
		return PostList{}, fmt.Errorf("GetPostsSince: since must be > 0, got %d", since)
	}

	q := url.Values{}
	q.Set("since", strconv.FormatInt(since, 10))

	path := "/api/v4/channels/" + url.PathEscape(channelID) + "/posts?" + q.Encode()

	data, err := c.Get(path)
	if err != nil {
		return PostList{}, fmt.Errorf("GetPostsSince: %w", err)
	}

	var pl PostList
	if err := json.Unmarshal(data, &pl); err != nil {
		return PostList{}, fmt.Errorf("GetPostsSince: %w", err)
	}

	return pl, nil
}

// CreatePost sends a new message to the given channel.
// Returns the created Post on success.
func (c *Client) CreatePost(channelID, message string) (Post, error) {
	body := struct {
		ChannelID string `json:"channel_id"`
		Message   string `json:"message"`
	}{
		ChannelID: channelID,
		Message:   message,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return Post{}, fmt.Errorf("CreatePost: %w", err)
	}

	data, err := c.Post("/api/v4/posts", bytes.NewReader(raw))
	if err != nil {
		return Post{}, fmt.Errorf("CreatePost: %w", err)
	}

	var p Post
	if err := json.Unmarshal(data, &p); err != nil {
		return Post{}, fmt.Errorf("CreatePost: %w", err)
	}

	return p, nil
}
