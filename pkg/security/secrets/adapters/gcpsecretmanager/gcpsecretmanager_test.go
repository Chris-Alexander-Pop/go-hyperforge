package gcpsecretmanager_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/secrets/adapters/gcpsecretmanager"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeGCP struct {
	mu   sync.Mutex
	data map[string]string // secret resource name -> value
}

func newFakeGCP() *fakeGCP {
	return &fakeGCP{data: make(map[string]string)}
}

func (f *fakeGCP) AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	// name ends with /versions/latest
	name := req.GetName()
	parent := name
	if i := len("/versions/latest"); len(name) > i && name[len(name)-i:] == "/versions/latest" {
		parent = name[:len(name)-i]
	}
	v, ok := f.data[parent]
	if !ok {
		return nil, status.Error(codes.NotFound, "not found")
	}
	return &secretmanagerpb.AccessSecretVersionResponse{
		Payload: &secretmanagerpb.SecretPayload{Data: []byte(v)},
	}, nil
}

func (f *fakeGCP) CreateSecret(ctx context.Context, req *secretmanagerpb.CreateSecretRequest, _ ...gax.CallOption) (*secretmanagerpb.Secret, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	name := req.GetParent() + "/secrets/" + req.GetSecretId()
	if _, ok := f.data[name]; ok {
		return nil, status.Error(codes.AlreadyExists, "exists")
	}
	f.data[name] = ""
	return &secretmanagerpb.Secret{Name: name}, nil
}

func (f *fakeGCP) AddSecretVersion(ctx context.Context, req *secretmanagerpb.AddSecretVersionRequest, _ ...gax.CallOption) (*secretmanagerpb.SecretVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	parent := req.GetParent()
	if _, ok := f.data[parent]; !ok {
		return nil, status.Error(codes.NotFound, "secret not found")
	}
	f.data[parent] = string(req.GetPayload().GetData())
	return &secretmanagerpb.SecretVersion{Name: parent + "/versions/1"}, nil
}

func (f *fakeGCP) Close() error { return nil }

type GCPSMSuite struct {
	test.Suite
	mgr *gcpsecretmanager.Manager
}

func (s *GCPSMSuite) SetupTest() {
	s.Suite.SetupTest()
	var err error
	s.mgr, err = gcpsecretmanager.NewFromAPI(newFakeGCP(), "demo")
	s.Require().NoError(err)
}

func (s *GCPSMSuite) TestSetGet() {
	err := s.mgr.Set(s.Ctx, "api-key", "xyz")
	s.Require().NoError(err)
	v, err := s.mgr.Get(s.Ctx, "api-key")
	s.Require().NoError(err)
	s.Equal("xyz", v)
}

func (s *GCPSMSuite) TestGetNotFound() {
	_, err := s.mgr.Get(s.Ctx, "missing")
	s.Require().Error(err)
	s.True(errors.Is(err, secrets.ErrNotFound))
}

func (s *GCPSMSuite) TestRotate() {
	s.Require().NoError(s.mgr.Set(s.Ctx, "tok", "old"))
	nv, err := s.mgr.Rotate(s.Ctx, "tok", "new")
	s.Require().NoError(err)
	s.Equal("new", nv)
}

func (s *GCPSMSuite) TestEmptyName() {
	_, err := s.mgr.Get(s.Ctx, "")
	s.True(errors.Is(err, secrets.ErrInvalidArgument))
}

func TestGCPSMSuite(t *testing.T) {
	test.Run(t, new(GCPSMSuite))
}
