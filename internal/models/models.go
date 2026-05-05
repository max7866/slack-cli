package models

// ChannelInfo holds display info for a channel.
type ChannelInfo struct {
	ID         string
	Name       string
	Topic      string
	Members    int
	IsPrivate  bool
	IsArchived bool
}

// MessageInfo holds display info for a message.
type MessageInfo struct {
	Timestamp string
	User      string
	Text      string
}

// UserInfo holds display info for a user.
type UserInfo struct {
	ID       string
	Name     string
	RealName string
	Email    string
	IsBot    bool
	IsAdmin  bool
}
