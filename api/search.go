package api

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SearchUsers searches for users matching term. Returns active users only.
func (c *Client) SearchUsers(term string) ([]User, error) {
	body, _ := json.Marshal(map[string]any{
		"term":           term,
		"allow_inactive": false,
	})
	data, err := c.Post("/api/v4/users/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	return users, nil
}

// SearchChannels searches for public channels in teamID matching term.
func (c *Client) SearchChannels(term, teamID string) ([]Channel, error) {
	body, _ := json.Marshal(map[string]any{"term": term})
	data, err := c.Post(fmt.Sprintf("/api/v4/teams/%s/channels/search", teamID), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var channels []Channel
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil, fmt.Errorf("search channels: %w", err)
	}
	return channels, nil
}
