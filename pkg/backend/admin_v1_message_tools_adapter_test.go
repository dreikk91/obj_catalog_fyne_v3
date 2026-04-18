package backend

import (
	"testing"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1MessageToolsStub struct {
	protocols         []int64
	messages          []contracts.AdminMessage
	buckets           contracts.Admin220VMessageBuckets
	displayObjects    []contracts.DisplayBlockObject
	listProtocolErr   error
	listMessagesErr   error
	listBucketsErr    error
	listObjectsErr    error
	setAdminOnlyUIN   int64
	setAdminOnlyValue bool
	setCategoryUIN    int64
	setCategorySC1    *int64
	set220VUIN        int64
	set220VMode       contracts.Admin220VMode
	emulateObjN       int64
	emulateZone       int64
	emulateMessageUIN int64
	protocolReceived  *int64
	messageFilter     string
	bucketProtocols   []int64
	bucketFilter      string
	objectFilter      string
}

func (s *adminV1MessageToolsStub) ListMessageProtocols() ([]int64, error) {
	return s.protocols, s.listProtocolErr
}

func (s *adminV1MessageToolsStub) ListMessages(protocolID *int64, filter string) ([]contracts.AdminMessage, error) {
	s.protocolReceived = protocolID
	s.messageFilter = filter
	return s.messages, s.listMessagesErr
}

func (s *adminV1MessageToolsStub) SetMessageAdminOnly(uin int64, adminOnly bool) error {
	s.setAdminOnlyUIN = uin
	s.setAdminOnlyValue = adminOnly
	return nil
}

func (s *adminV1MessageToolsStub) SetMessageCategory(uin int64, sc1 *int64) error {
	s.setCategoryUIN = uin
	s.setCategorySC1 = sc1
	return nil
}

func (s *adminV1MessageToolsStub) List220VMessageBuckets(protocolIDs []int64, filter string) (contracts.Admin220VMessageBuckets, error) {
	s.bucketProtocols = append([]int64(nil), protocolIDs...)
	s.bucketFilter = filter
	return s.buckets, s.listBucketsErr
}

func (s *adminV1MessageToolsStub) SetMessage220VMode(uin int64, mode contracts.Admin220VMode) error {
	s.set220VUIN = uin
	s.set220VMode = mode
	return nil
}

func (s *adminV1MessageToolsStub) ListDisplayBlockObjects(filter string) ([]contracts.DisplayBlockObject, error) {
	s.objectFilter = filter
	return s.displayObjects, s.listObjectsErr
}

func (s *adminV1MessageToolsStub) SetDisplayBlockMode(objn int64, mode contracts.DisplayBlockMode) error {
	return nil
}

func (s *adminV1MessageToolsStub) EmulateEvent(objn int64, zone int64, messageUIN int64) error {
	s.emulateObjN = objn
	s.emulateZone = zone
	s.emulateMessageUIN = messageUIN
	return nil
}

func TestAdminV1MessagesProviderListMessages(t *testing.T) {
	base := &adminV1MessageToolsStub{
		protocols: []int64{4, 18},
		messages:  []contracts.AdminMessage{{UIN: 7, Text: "demo"}},
	}
	provider := NewAdminV1MessagesProvider(base)
	protocolID := int64(4)

	items, err := provider.ListMessages(&protocolID, "alarm")
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}

	if base.protocolReceived == nil || *base.protocolReceived != 4 {
		t.Fatalf("protocol = %+v, want 4", base.protocolReceived)
	}
	if base.messageFilter != "alarm" {
		t.Fatalf("filter = %q, want alarm", base.messageFilter)
	}
	if len(items) != 1 || items[0].UIN != 7 {
		t.Fatalf("items = %+v, want one item with UIN=7", items)
	}
}

func TestAdminV1EventOverrideProvider220VMode(t *testing.T) {
	base := &adminV1MessageToolsStub{
		buckets: contracts.Admin220VMessageBuckets{
			Alarm: []contracts.AdminMessage{{UIN: 11, Text: "220"}},
		},
	}
	provider := NewAdminV1EventOverrideProvider(base)

	buckets, err := provider.List220VMessageBuckets([]int64{4}, "220")
	if err != nil {
		t.Fatalf("List220VMessageBuckets() error = %v", err)
	}
	if len(buckets.Alarm) != 1 || buckets.Alarm[0].UIN != 11 {
		t.Fatalf("buckets = %+v, want one alarm UIN=11", buckets)
	}

	if err := provider.SetMessage220VMode(11, adminv1.Message220VModeRestore); err != nil {
		t.Fatalf("SetMessage220VMode() error = %v", err)
	}
	if base.set220VMode != contracts.Admin220VRestore {
		t.Fatalf("mode = %v, want %v", base.set220VMode, contracts.Admin220VRestore)
	}
}

func TestAdminV1EventEmulationProvider(t *testing.T) {
	base := &adminV1MessageToolsStub{
		displayObjects: []contracts.DisplayBlockObject{{ObjN: 15, Name: "obj"}},
	}
	provider := NewAdminV1EventEmulationProvider(base)

	objects, err := provider.ListDisplayBlockObjects("obj")
	if err != nil {
		t.Fatalf("ListDisplayBlockObjects() error = %v", err)
	}
	if base.objectFilter != "obj" {
		t.Fatalf("object filter = %q, want obj", base.objectFilter)
	}
	if len(objects) != 1 || objects[0].ObjN != 15 {
		t.Fatalf("objects = %+v, want one object ObjN=15", objects)
	}

	if err := provider.EmulateEvent(15, 2, 99); err != nil {
		t.Fatalf("EmulateEvent() error = %v", err)
	}
	if base.emulateObjN != 15 || base.emulateZone != 2 || base.emulateMessageUIN != 99 {
		t.Fatalf("emulate args = (%d,%d,%d), want (15,2,99)", base.emulateObjN, base.emulateZone, base.emulateMessageUIN)
	}
}
