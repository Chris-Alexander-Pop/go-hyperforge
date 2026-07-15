package iot_test

import (
	"testing"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.10.0", "1.9.0", 1},
		{"1.9.0", "1.10.0", -1},
		{"2.0.0", "1.99.99", 1},
		{"1.0.0-alpha", "1.0.0", -1},
	}
	for _, tt := range tests {
		got, err := iot.CompareVersions(tt.a, tt.b)
		if err != nil {
			t.Fatalf("CompareVersions(%q,%q): %v", tt.a, tt.b, err)
		}
		if got != tt.want {
			t.Fatalf("CompareVersions(%q,%q)=%d want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsNewerVersion(t *testing.T) {
	ok, err := iot.IsNewerVersion("1.10.0", "1.9.0")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	ok, err = iot.IsNewerVersion("1.0.0", "1.0.0")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestIsNewerVersion_Invalid(t *testing.T) {
	_, err := iot.IsNewerVersion("nope", "1.0.0")
	if !pkgerrors.IsCode(err, iot.CodeInvalidVersion) {
		t.Fatalf("err=%v", err)
	}
}

func TestNormalizeVersion(t *testing.T) {
	if got := iot.NormalizeVersion("1.2.3"); got != "v1.2.3" {
		t.Fatalf("got %q", got)
	}
	if got := iot.NormalizeVersion("v1.2.3"); got != "v1.2.3" {
		t.Fatalf("got %q", got)
	}
	if !iot.IsValidVersion("1.2.3") {
		t.Fatal("expected valid")
	}
	if iot.IsValidVersion("") {
		t.Fatal("expected invalid empty")
	}
}

func TestErrorConstructors(t *testing.T) {
	cases := []struct {
		err  *pkgerrors.AppError
		code string
	}{
		{iot.ErrConnectionFailed(nil), iot.CodeConnectionFailed},
		{iot.ErrPublishFailed(nil), iot.CodePublishFailed},
		{iot.ErrSubscribeFailed(nil), iot.CodeSubscribeFailed},
		{iot.ErrTimeout("publish", nil), iot.CodeTimeout},
		{iot.ErrNotConnected(), iot.CodeNotConnected},
		{iot.ErrInvalidConfig("x", nil), iot.CodeInvalidConfig},
		{iot.ErrManifestNotFound(nil), iot.CodeManifestNotFound},
		{iot.ErrDownloadFailed("f", nil), iot.CodeDownloadFailed},
		{iot.ErrChecksumMismatch("f", "a", "b"), iot.CodeChecksumMismatch},
		{iot.ErrInvalidVersion("x", nil), iot.CodeInvalidVersion},
		{iot.ErrUpdateFailed(nil), iot.CodeUpdateFailed},
	}
	for _, tc := range cases {
		if tc.err.Code != tc.code {
			t.Fatalf("got %s want %s", tc.err.Code, tc.code)
		}
	}
}
