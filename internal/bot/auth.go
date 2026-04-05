package bot

// Auth handles user authorization based on an allowlist of Telegram user IDs.
type Auth struct {
	allowedIDs map[int64]bool
	enabled    bool
}

// NewAuth creates a new Auth checker. If allowedIDs is empty, all users are allowed.
func NewAuth(allowedIDs []int64) *Auth {
	m := make(map[int64]bool, len(allowedIDs))
	for _, id := range allowedIDs {
		m[id] = true
	}
	return &Auth{
		allowedIDs: m,
		enabled:    len(allowedIDs) > 0,
	}
}

// IsAllowed returns true if the given user ID is authorized to use the bot.
// If no allowlist is configured, all users are allowed.
func (a *Auth) IsAllowed(userID int64) bool {
	if !a.enabled {
		return true
	}
	return a.allowedIDs[userID]
}
