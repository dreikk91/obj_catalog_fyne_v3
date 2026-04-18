package backend

import (
	"testing"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1SystemSubServerStub struct {
	status       contracts.AdminAccessStatus
	issues       []contracts.AdminDataCheckIssue
	subservers   []contracts.AdminSubServer
	objects      []contracts.AdminSubServerObject
	issueLimit   int
	objectFilter string
	setObjN      int64
	setChannel   int
	setBind      string
	clearObjN    int64
	clearChannel int
}

func (s *adminV1SystemSubServerStub) GetAdminAccessStatus() (contracts.AdminAccessStatus, error) {
	return s.status, nil
}

func (s *adminV1SystemSubServerStub) RunDataIntegrityChecks(limit int) ([]contracts.AdminDataCheckIssue, error) {
	s.issueLimit = limit
	return s.issues, nil
}

func (s *adminV1SystemSubServerStub) ListSubServers() ([]contracts.AdminSubServer, error) {
	return s.subservers, nil
}

func (s *adminV1SystemSubServerStub) ListSubServerObjects(filter string) ([]contracts.AdminSubServerObject, error) {
	s.objectFilter = filter
	return s.objects, nil
}

func (s *adminV1SystemSubServerStub) SetObjectSubServer(objn int64, channel int, bind string) error {
	s.setObjN = objn
	s.setChannel = channel
	s.setBind = bind
	return nil
}

func (s *adminV1SystemSubServerStub) ClearObjectSubServer(objn int64, channel int) error {
	s.clearObjN = objn
	s.clearChannel = channel
	return nil
}

func TestAdminV1SystemControlProvider(t *testing.T) {
	base := &adminV1SystemSubServerStub{
		status: contracts.AdminAccessStatus{CurrentUser: "user", HasFullAccess: true},
		issues: []contracts.AdminDataCheckIssue{{Code: "I1", ObjN: 7}},
	}
	provider := NewAdminV1SystemControlProvider(base)

	status, err := provider.GetAdminAccessStatus()
	if err != nil {
		t.Fatalf("GetAdminAccessStatus() error = %v", err)
	}
	if status.CurrentUser != "user" || !status.HasFullAccess {
		t.Fatalf("status = %+v, want mapped status", status)
	}

	issues, err := provider.RunDataIntegrityChecks(123)
	if err != nil {
		t.Fatalf("RunDataIntegrityChecks() error = %v", err)
	}
	if base.issueLimit != 123 {
		t.Fatalf("limit = %d, want 123", base.issueLimit)
	}
	if len(issues) != 1 || issues[0].Code != "I1" {
		t.Fatalf("issues = %+v, want one issue code I1", issues)
	}
}

func TestAdminV1SubServerObjectsProvider(t *testing.T) {
	base := &adminV1SystemSubServerStub{
		subservers: []contracts.AdminSubServer{{ID: 1, Bind: "bind-a"}},
		objects:    []contracts.AdminSubServerObject{{ObjN: 11, Name: "obj"}},
	}
	provider := NewAdminV1SubServerObjectsProvider(base)

	servers, err := provider.ListSubServers()
	if err != nil {
		t.Fatalf("ListSubServers() error = %v", err)
	}
	if len(servers) != 1 || servers[0].Bind != "bind-a" {
		t.Fatalf("servers = %+v, want one server bind-a", servers)
	}

	objects, err := provider.ListSubServerObjects("obj")
	if err != nil {
		t.Fatalf("ListSubServerObjects() error = %v", err)
	}
	if base.objectFilter != "obj" {
		t.Fatalf("filter = %q, want obj", base.objectFilter)
	}
	if len(objects) != 1 || objects[0].ObjN != 11 {
		t.Fatalf("objects = %+v, want one object 11", objects)
	}

	if err := provider.SetObjectSubServer(11, 1, "bind-a"); err != nil {
		t.Fatalf("SetObjectSubServer() error = %v", err)
	}
	if base.setObjN != 11 || base.setChannel != 1 || base.setBind != "bind-a" {
		t.Fatalf("set args = (%d,%d,%q), want (11,1,bind-a)", base.setObjN, base.setChannel, base.setBind)
	}

	if err := provider.ClearObjectSubServer(11, 1); err != nil {
		t.Fatalf("ClearObjectSubServer() error = %v", err)
	}
	if base.clearObjN != 11 || base.clearChannel != 1 {
		t.Fatalf("clear args = (%d,%d), want (11,1)", base.clearObjN, base.clearChannel)
	}
}

var (
	_ adminv1.SystemControlProvider    = (*adminV1SystemControlAdapter)(nil)
	_ adminv1.SubServerObjectsProvider = (*adminV1SubServerObjectsAdapter)(nil)
)
