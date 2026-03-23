package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// GetChannelsForUser returns all channels the user belongs to in the given team.
// GET /api/v4/users/{userID}/teams/{teamID}/channels
func (c *Client) GetChannelsForUser(userID, teamID string) ([]Channel, error) {
	path := fmt.Sprintf("/api/v4/users/%s/teams/%s/channels",
		url.PathEscape(userID),
		url.PathEscape(teamID),
	)
	data, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("GetChannelsForUser: %w", err)
	}
	var channels []Channel
	if err := json.Unmarshal(data, &channels); err != nil {
		return nil, fmt.Errorf("GetChannelsForUser: %w", err)
	}
	return channels, nil
}

// GetChannelMembersForUser returns channel membership records for all of the
// user's channels in the given team (unread counts, notify props, etc.)
// GET /api/v4/users/{userID}/teams/{teamID}/channels/members
func (c *Client) GetChannelMembersForUser(userID, teamID string) ([]ChannelMember, error) {
	path := fmt.Sprintf("/api/v4/users/%s/teams/%s/channels/members",
		url.PathEscape(userID),
		url.PathEscape(teamID),
	)
	data, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("GetChannelMembersForUser: %w", err)
	}
	var members []ChannelMember
	if err := json.Unmarshal(data, &members); err != nil {
		return nil, fmt.Errorf("GetChannelMembersForUser: %w", err)
	}
	return members, nil
}

// GetPreferences returns all preferences for the user.
// GET /api/v4/users/{userID}/preferences
func (c *Client) GetPreferences(userID string) ([]Preference, error) {
	path := fmt.Sprintf("/api/v4/users/%s/preferences",
		url.PathEscape(userID),
	)
	data, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("GetPreferences: %w", err)
	}
	var prefs []Preference
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, fmt.Errorf("GetPreferences: %w", err)
	}
	return prefs, nil
}

// MarkChannelRead marks the channel as read for the current user.
// POST /api/v4/channels/members/me/view
// Body: {"channel_id": "<channelID>", "prev_channel_id": ""}
func (c *Client) MarkChannelRead(channelID string) error {
	type markReadBody struct {
		ChannelID     string `json:"channel_id"`
		PrevChannelID string `json:"prev_channel_id"`
	}
	b, err := json.Marshal(markReadBody{ChannelID: channelID})
	if err != nil {
		return fmt.Errorf("mark channel read: marshal body: %w", err)
	}
	_, err = c.Post("/api/v4/channels/members/me/view", strings.NewReader(string(b)))
	if err != nil {
		return fmt.Errorf("MarkChannelRead: %w", err)
	}
	return nil
}

// CreateDirectChannel creates or returns the existing DM channel between the
// current user and otherUserID.
// POST /api/v4/channels/direct  body: ["myUserID","otherUserID"]
func (c *Client) CreateDirectChannel(otherUserID string) (Channel, error) {
	body, _ := json.Marshal([]string{c.userID, otherUserID})
	data, err := c.Post("/api/v4/channels/direct", bytes.NewReader(body))
	if err != nil {
		return Channel{}, fmt.Errorf("CreateDirectChannel: %w", err)
	}
	var ch Channel
	if err := json.Unmarshal(data, &ch); err != nil {
		return Channel{}, fmt.Errorf("CreateDirectChannel: %w", err)
	}
	return ch, nil
}

// JoinChannel adds the current user to a public channel.
// POST /api/v4/channels/{channelID}/members  body: {"user_id":"myUserID"}
func (c *Client) JoinChannel(channelID string) error {
	body, _ := json.Marshal(map[string]string{"user_id": c.userID})
	_, err := c.Post(
		fmt.Sprintf("/api/v4/channels/%s/members", url.PathEscape(channelID)),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("JoinChannel: %w", err)
	}
	return nil
}

// GetChannel fetches a single channel by ID.
// GET /api/v4/channels/{channelID}
func (c *Client) GetChannel(channelID string) (Channel, error) {
	data, err := c.Get(fmt.Sprintf("/api/v4/channels/%s", url.PathEscape(channelID)))
	if err != nil {
		return Channel{}, fmt.Errorf("GetChannel: %w", err)
	}
	var ch Channel
	if err := json.Unmarshal(data, &ch); err != nil {
		return Channel{}, fmt.Errorf("GetChannel: %w", err)
	}
	return ch, nil
}

// GetUsersByIDs fetches multiple users in a single request.
// Used for resolving DM channel display names.
// POST /api/v4/users/ids
// Body: JSON array of user ID strings
func (c *Client) GetUsersByIDs(userIDs []string) ([]User, error) {
	bodyBytes, err := json.Marshal(userIDs)
	if err != nil {
		return nil, fmt.Errorf("GetUsersByIDs: %w", err)
	}
	data, err := c.Post("/api/v4/users/ids", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("GetUsersByIDs: %w", err)
	}
	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("GetUsersByIDs: %w", err)
	}
	return users, nil
}
