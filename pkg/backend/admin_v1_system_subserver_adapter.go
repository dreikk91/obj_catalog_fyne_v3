package backend

import (
	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1SystemControlBase interface {
	GetAdminAccessStatus() (contracts.AdminAccessStatus, error)
	RunDataIntegrityChecks(limit int) ([]contracts.AdminDataCheckIssue, error)
}

type adminV1SubServerObjectsBase interface {
	ListSubServers() ([]contracts.AdminSubServer, error)
	ListSubServerObjects(filter string) ([]contracts.AdminSubServerObject, error)
	SetObjectSubServer(objn int64, channel int, bind string) error
	ClearObjectSubServer(objn int64, channel int) error
}

type adminV1SystemControlAdapter struct {
	base adminV1SystemControlBase
}

type adminV1SubServerObjectsAdapter struct {
	base adminV1SubServerObjectsBase
}

func NewAdminV1SystemControlProvider(base adminV1SystemControlBase) adminv1.SystemControlProvider {
	if base == nil {
		return nil
	}
	return &adminV1SystemControlAdapter{base: base}
}

func NewAdminV1SubServerObjectsProvider(base adminV1SubServerObjectsBase) adminv1.SubServerObjectsProvider {
	if base == nil {
		return nil
	}
	return &adminV1SubServerObjectsAdapter{base: base}
}

func (a *adminV1SystemControlAdapter) GetAdminAccessStatus() (adminv1.AccessStatus, error) {
	status, err := a.base.GetAdminAccessStatus()
	if err != nil {
		return adminv1.AccessStatus{}, err
	}
	return adminv1.ToAccessStatus(status), nil
}

func (a *adminV1SystemControlAdapter) RunDataIntegrityChecks(limit int) ([]adminv1.DataCheckIssue, error) {
	items, err := a.base.RunDataIntegrityChecks(limit)
	if err != nil {
		return nil, err
	}
	return adminv1.ToDataCheckIssues(items), nil
}

func (a *adminV1SubServerObjectsAdapter) ListSubServers() ([]adminv1.SubServer, error) {
	items, err := a.base.ListSubServers()
	if err != nil {
		return nil, err
	}
	return adminv1.ToSubServers(items), nil
}

func (a *adminV1SubServerObjectsAdapter) ListSubServerObjects(filter string) ([]adminv1.SubServerObject, error) {
	items, err := a.base.ListSubServerObjects(filter)
	if err != nil {
		return nil, err
	}
	return adminv1.ToSubServerObjects(items), nil
}

func (a *adminV1SubServerObjectsAdapter) SetObjectSubServer(objn int64, channel int, bind string) error {
	return a.base.SetObjectSubServer(objn, channel, bind)
}

func (a *adminV1SubServerObjectsAdapter) ClearObjectSubServer(objn int64, channel int) error {
	return a.base.ClearObjectSubServer(objn, channel)
}
