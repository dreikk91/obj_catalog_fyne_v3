package data

import (
	"context"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestFlattenCASLObjectMediaIncludesObjectRoomAndCamera(t *testing.T) {
	got := flattenCASLObjectMedia(contracts.CASLGuardObjectDetails{
		Images: []string{"10"},
		Rooms: []contracts.CASLRoomDetails{
			{RoomID: "2", Name: "Склад", Images: []string{"20"}, RTSP: "rtsp://camera"},
		},
	})
	if len(got) != 3 {
		t.Fatalf("media count = %d, want 3", len(got))
	}
	if got[0].Kind != contracts.ObjectMediaImage || got[1].RoomName != "Склад" {
		t.Fatalf("unexpected image media: %+v", got)
	}
	if got[2].Kind != contracts.ObjectMediaCamera || got[2].URL != "rtsp://camera" {
		t.Fatalf("unexpected camera media: %+v", got[2])
	}
}

func TestFetchObjectMediaDecodesDataImage(t *testing.T) {
	provider := &CASLCloudProvider{}
	body, err := provider.FetchObjectMedia(context.Background(), contracts.ObjectMedia{
		Kind: contracts.ObjectMediaImage,
		ID:   "data:image/png;base64,ZmFrZQ==",
	})
	if err != nil {
		t.Fatalf("FetchObjectMedia() error = %v", err)
	}
	if string(body) != "fake" {
		t.Fatalf("decoded body = %q", body)
	}
}
