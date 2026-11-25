package identity

import (
	"context"
	"errors"
	"testing"

	"wesmk/internal/authz"
)

type fakeRepo struct {
	profile *StudentProfile
	err     error
}

func (f fakeRepo) StudentProfile(context.Context, string) (*StudentProfile, error) {
	return f.profile, f.err
}

func TestForViewerPolicy(t *testing.T) {
	profile := &StudentProfile{AppID: "std-budi-01", UserID: "usr-student-1", FullName: "Budi"}
	service := NewService(fakeRepo{profile: profile})

	tests := []struct {
		name    string
		viewer  Viewer
		wantErr error
	}{
		{"siswa sendiri", Viewer{UserID: "usr-student-1", Role: "student", StudentID: "std-budi-01"}, nil},
		{"orang tua anak sendiri", Viewer{UserID: "usr-parent-1", Role: "parent", StudentID: "std-budi-01"}, nil},
		{"siswa lain", Viewer{UserID: "usr-student-2", Role: "student", StudentID: "std-rahma-02"}, authz.ErrDenied},
		{"guru", Viewer{UserID: "usr-teacher-1", Role: "teacher"}, nil},
		{"admin TU", Viewer{UserID: "usr-admin-1", Role: "admin_tu"}, nil},
		{"developer", Viewer{UserID: "usr-developer-1", Role: "developer"}, authz.ErrDenied},
		{"peran kosong", Viewer{}, authz.ErrDenied},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := service.ForViewer(context.Background(), test.viewer, "std-budi-01")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("err = %v, want %v", err, test.wantErr)
			}
			if test.wantErr == nil && got != profile {
				t.Fatal("expected the domain profile")
			}
		})
	}
}

func TestForViewerMissingProfile(t *testing.T) {
	service := NewService(fakeRepo{err: ErrProfileNotFound})
	if _, err := service.ForViewer(context.Background(), Viewer{Role: "admin_tu"}, "std-x"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("err = %v, want ErrProfileNotFound", err)
	}
	if _, err := service.ForViewer(context.Background(), Viewer{Role: "admin_tu"}, ""); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("empty id err = %v, want ErrProfileNotFound", err)
	}
}
