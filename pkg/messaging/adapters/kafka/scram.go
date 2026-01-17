package kafka

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"github.com/xdg-go/scram"
)

// Hash generator types for SCRAM authentication
var (
	SHA256 scram.HashGeneratorFcn = sha256.New
	SHA512 scram.HashGeneratorFcn = sha512.New
)

// scramClient implements sarama.SCRAMClient for SCRAM-SHA authentication.
type scramClient struct {
	*scram.Client
	*scram.ClientConversation
	HashGeneratorFcn scram.HashGeneratorFcn
}

func (c *scramClient) Begin(userName, password, authzID string) error {
	client, err := c.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	c.Client = client
	c.ClientConversation = client.NewConversation()
	return nil
}

func (c *scramClient) Step(challenge string) (string, error) {
	return c.ClientConversation.Step(challenge)
}

func (c *scramClient) Done() bool {
	return c.ClientConversation.Done()
}

// hashFunc is a helper for creating hash functions
type hashFunc func() hash.Hash
