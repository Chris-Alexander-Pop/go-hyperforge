package redis

// Export private methods for testing
func (p *MFAProvider) TestKey(userID string) string {
	return p.key(userID)
}

func (p *MFAProvider) TestUsedKey(userID string, counter uint64) string {
	return p.usedKey(userID, counter)
}
