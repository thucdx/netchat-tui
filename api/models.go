package api

// User represents a Mattermost user.
type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	DeleteAt  int64  `json:"delete_at"`
}

// Team represents a Mattermost team.
type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// Channel represents a Mattermost channel.
// Type: "D" = Direct Message, "G" = Group Message, "O" = Open/Public, "P" = Private
type Channel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
	Type          string `json:"type"`
	TeamID        string `json:"team_id"`
	TotalMsgCount int64  `json:"total_msg_count"`
	LastPostAt    int64  `json:"last_post_at"`
	Header        string `json:"header"`
	Purpose       string `json:"purpose"`
	DeleteAt      int64  `json:"delete_at"`
}

// IsDeleted returns true if the channel has been deleted.
func (ch Channel) IsDeleted() bool {
	return ch.DeleteAt > 0
}

// NotifyProps holds per-channel notification preferences for a member.
type NotifyProps struct {
	MarkUnread            string `json:"mark_unread"`              // "all" | "mention"
	Desktop               string `json:"desktop"`                  // "default" | "all" | "mention" | "none"
	Push                  string `json:"push"`                     // "default" | "all" | "mention" | "none"
	IgnoreChannelMentions string `json:"ignore_channel_mentions"`  // "default" | "on" | "off"
}

// ChannelMember holds a user's membership info for a specific channel.
type ChannelMember struct {
	ChannelID    string      `json:"channel_id"`
	UserID       string      `json:"user_id"`
	MsgCount     int64       `json:"msg_count"`
	MentionCount int64       `json:"mention_count"`
	NotifyProps  NotifyProps `json:"notify_props"`
	LastViewedAt int64       `json:"last_viewed_at"`
}

// IsMuted returns true if the channel member has muted this channel.
// A channel is muted when mark_unread is set to "mention".
func (cm ChannelMember) IsMuted() bool {
	return cm.NotifyProps.MarkUnread == "mention"
}

// UnreadCount returns the number of unread messages for this member.
func (cm ChannelMember) UnreadCount(ch Channel) int64 {
	if ch.TotalMsgCount <= cm.MsgCount {
		return 0
	}
	return ch.TotalMsgCount - cm.MsgCount
}

// Post represents a single message/post in a channel.
type Post struct {
	ID        string   `json:"id"`
	ChannelID string   `json:"channel_id"`
	UserID    string   `json:"user_id"`
	Message   string   `json:"message"`
	CreateAt  int64    `json:"create_at"`
	UpdateAt  int64    `json:"update_at"`
	DeleteAt  int64    `json:"delete_at"`
	Type      string   `json:"type"`      // "" = normal, "system_*" = system message
	RootID    string   `json:"root_id"`   // non-empty if this is a thread reply
	EditAt    int64    `json:"edit_at"`
	IsPinned  bool     `json:"is_pinned"`
	FileIds   []string `json:"file_ids"` // IDs of attached files (may be nil)
}

// PostList is the response from the posts endpoint.
// Order is a slice of post IDs in display order (newest last).
// Posts is a map from post ID to Post.
type PostList struct {
	Order      []string        `json:"order"`
	Posts      map[string]Post `json:"posts"`
	NextPostID string          `json:"next_post_id"`
	PrevPostID string          `json:"prev_post_id"`
}

// FileInfo holds metadata for a Mattermost file attachment.
type FileInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Extension string `json:"extension"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// Preference represents a single user preference entry.
type Preference struct {
	UserID   string `json:"user_id"`
	Category string `json:"category"`
	Name     string `json:"name"`
	Value    string `json:"value"`
}

// WSEvent represents a WebSocket event from the Mattermost server.
type WSEvent struct {
	Event     string                 `json:"event"`
	Data      map[string]interface{} `json:"data"`
	Broadcast WSBroadcast            `json:"broadcast"`
	Seq       int64                  `json:"seq"`
}

// WSBroadcast holds the targeting info for a WebSocket event.
type WSBroadcast struct {
	OmitUsers map[string]bool `json:"omit_users"`
	UserID    string          `json:"user_id"`
	ChannelID string          `json:"channel_id"`
	TeamID    string          `json:"team_id"`
}
