package main

import (
	"errors"
	"testing"
)

func TestConfigureRFSequenceNotifications(t *testing.T) {
	for _, tc := range []struct {
		name       string
		revision   int
		legacy     bool
		wantEnable bool
	}{
		{"old revision remains enabled by default", 60, false, true},
		{"old revision can opt out", 60, true, false},
		{"last old revision can opt out", 89, true, false},
		{"revision 9.0 remains enabled", 90, true, true},
		{"new revision remains enabled with legacy option", 130, true, true},
		{"new revision remains enabled by default", 130, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := configureRFSequenceNotifications(tc.revision, tc.legacy, func() error {
				calls++
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			wantCalls := 0
			if tc.wantEnable {
				wantCalls = 1
			}
			if calls != wantCalls {
				t.Fatalf("enabled sequence-number notifications %d times, want %d", calls, wantCalls)
			}
		})
	}
}

func TestConfigureRFSequenceNotificationsReturnsCommandError(t *testing.T) {
	want := errors.New("configuration rejected")
	if err := configureRFSequenceNotifications(130, true, func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("got %v, want command error", err)
	}
}
