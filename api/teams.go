package api

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// GetTeamsForUser returns all teams the given user belongs to.
func (c *Client) GetTeamsForUser(userID string) ([]Team, error) {
	path := "/api/v4/users/" + url.PathEscape(userID) + "/teams"

	data, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get teams for user: %w", err)
	}

	var teams []Team
	if err := json.Unmarshal(data, &teams); err != nil {
		return nil, fmt.Errorf("get teams for user: parse response: %w", err)
	}

	return teams, nil
}

// GetFirstTeam returns the first team for the user, or an error if none found.
// Returns the first team (the app uses single-team mode).
func (c *Client) GetFirstTeam(userID string) (Team, error) {
	teams, err := c.GetTeamsForUser(userID)
	if err != nil {
		return Team{}, err
	}

	if len(teams) == 0 {
		return Team{}, fmt.Errorf("no teams found for user")
	}

	return teams[0], nil
}
