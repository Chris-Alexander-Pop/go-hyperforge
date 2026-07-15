package kubo_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3/adapters/kubo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKuboStore_AddGetPin(t *testing.T) {
	storeData := map[string][]byte{}
	pins := map[string]struct{}{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v0/add"):
			_, _ = io.Copy(io.Discard, r.Body)
			cid := "QmTestCID123"
			storeData[cid] = []byte("hello")
			_ = json.NewEncoder(w).Encode(map[string]string{"Hash": cid})
		case strings.HasPrefix(r.URL.Path, "/api/v0/cat"):
			cid := r.URL.Query().Get("arg")
			data, ok := storeData[cid]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write(data)
		case strings.HasPrefix(r.URL.Path, "/api/v0/pin/add"):
			cid := r.URL.Query().Get("arg")
			pins[cid] = struct{}{}
			_, _ = io.WriteString(w, `{}`)
		case strings.HasPrefix(r.URL.Path, "/api/v0/pin/rm"):
			cid := r.URL.Query().Get("arg")
			delete(pins, cid)
			_, _ = io.WriteString(w, `{}`)
		case strings.HasPrefix(r.URL.Path, "/api/v0/pin/ls"):
			keys := map[string]map[string]string{}
			for cid := range pins {
				keys[cid] = map[string]string{"Type": "recursive"}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"Keys": keys})
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	s, err := kubo.New(kubo.Config{APIURL: srv.URL, GatewayURL: "https://gateway.test"})
	require.NoError(t, err)
	s.WithHTTPClient(srv.Client())

	var _ web3.Store = s

	ctx := context.Background()
	cid, err := s.Add(ctx, []byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, "QmTestCID123", cid)

	data, err := s.Get(ctx, cid)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), data)

	require.NoError(t, s.Pin(ctx, cid))
	list, err := s.ListPins(ctx)
	require.NoError(t, err)
	assert.Contains(t, list, cid)
	require.NoError(t, s.Unpin(ctx, cid))

	assert.Equal(t, "https://gateway.test/ipfs/"+cid, s.GetURL(cid))
}
