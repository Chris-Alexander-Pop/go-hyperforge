package s3

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
)

func TestMapError_NoSuchKey(t *testing.T) {
	err := mapError("download", &types.NoSuchKey{Message: strPtr("The specified key does not exist.")})
	if !blob.IsNotFound(err) {
		t.Fatalf("expected NotFound, got %v", err)
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != pkgerrors.CodeNotFound {
		t.Fatalf("expected AppError NOT_FOUND, got %#v", err)
	}
}

func TestMapError_NotFoundType(t *testing.T) {
	err := mapError("download", &types.NotFound{Message: strPtr("Not Found")})
	if !blob.IsNotFound(err) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestMapError_APIErrorNoSuchKey(t *testing.T) {
	err := mapError("download", &smithy.GenericAPIError{Code: "NoSuchKey", Message: "missing"})
	if !blob.IsNotFound(err) {
		t.Fatalf("expected NotFound for APIError NoSuchKey, got %v", err)
	}
}

func TestMapError_OtherError(t *testing.T) {
	err := mapError("download", errors.New("timeout"))
	if blob.IsNotFound(err) {
		t.Fatalf("did not expect NotFound for generic error")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != pkgerrors.CodeInternal {
		t.Fatalf("expected Internal, got %#v", err)
	}
}

func TestIsNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"NoSuchKey", &types.NoSuchKey{}, true},
		{"NotFound", &types.NotFound{}, true},
		{"API NoSuchKey", &smithy.GenericAPIError{Code: "NoSuchKey"}, true},
		{"API NotFound", &smithy.GenericAPIError{Code: "NotFound"}, true},
		{"other", errors.New("boom"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNotFound(tc.err); got != tc.want {
				t.Fatalf("isNotFound(%v)=%v want %v", tc.err, got, tc.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
