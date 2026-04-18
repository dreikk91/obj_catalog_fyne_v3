package backend

import (
	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1MessageLookupBase interface {
	ListMessageProtocols() ([]int64, error)
	ListMessages(protocolID *int64, filter string) ([]contracts.AdminMessage, error)
}

type adminV1MessagesBase interface {
	adminV1MessageLookupBase
	SetMessageAdminOnly(uin int64, adminOnly bool) error
}

type adminV1Message220VBase interface {
	List220VMessageBuckets(protocolIDs []int64, filter string) (contracts.Admin220VMessageBuckets, error)
	SetMessage220VMode(uin int64, mode contracts.Admin220VMode) error
}

type adminV1EventOverrideBase interface {
	adminV1MessagesBase
	adminV1Message220VBase
	SetMessageCategory(uin int64, sc1 *int64) error
}

type adminV1EventEmulationBase interface {
	ListDisplayBlockObjects(filter string) ([]contracts.DisplayBlockObject, error)
	adminV1MessageLookupBase
	EmulateEvent(objn int64, zone int64, messageUIN int64) error
}

type adminV1MessagesAdapter struct {
	base adminV1MessagesBase
}

type adminV1EventOverrideAdapter struct {
	base adminV1EventOverrideBase
}

type adminV1EventEmulationAdapter struct {
	base adminV1EventEmulationBase
}

func NewAdminV1MessagesProvider(base adminV1MessagesBase) adminv1.MessagesProvider {
	if base == nil {
		return nil
	}
	return &adminV1MessagesAdapter{base: base}
}

func NewAdminV1EventOverrideProvider(base adminV1EventOverrideBase) adminv1.EventOverrideProvider {
	if base == nil {
		return nil
	}
	return &adminV1EventOverrideAdapter{base: base}
}

func NewAdminV1EventEmulationProvider(base adminV1EventEmulationBase) adminv1.EventEmulationProvider {
	if base == nil {
		return nil
	}
	return &adminV1EventEmulationAdapter{base: base}
}

func (a *adminV1MessagesAdapter) ListMessageProtocols() ([]int64, error) {
	return a.base.ListMessageProtocols()
}

func (a *adminV1MessagesAdapter) ListMessages(protocolID *int64, filter string) ([]adminv1.Message, error) {
	items, err := a.base.ListMessages(protocolID, filter)
	if err != nil {
		return nil, err
	}
	return adminv1.ToMessages(items), nil
}

func (a *adminV1MessagesAdapter) SetMessageAdminOnly(uin int64, adminOnly bool) error {
	return a.base.SetMessageAdminOnly(uin, adminOnly)
}

func (a *adminV1EventOverrideAdapter) ListMessageProtocols() ([]int64, error) {
	return a.base.ListMessageProtocols()
}

func (a *adminV1EventOverrideAdapter) ListMessages(protocolID *int64, filter string) ([]adminv1.Message, error) {
	items, err := a.base.ListMessages(protocolID, filter)
	if err != nil {
		return nil, err
	}
	return adminv1.ToMessages(items), nil
}

func (a *adminV1EventOverrideAdapter) SetMessageAdminOnly(uin int64, adminOnly bool) error {
	return a.base.SetMessageAdminOnly(uin, adminOnly)
}

func (a *adminV1EventOverrideAdapter) SetMessageCategory(uin int64, sc1 *int64) error {
	return a.base.SetMessageCategory(uin, sc1)
}

func (a *adminV1EventOverrideAdapter) List220VMessageBuckets(protocolIDs []int64, filter string) (adminv1.Message220VBuckets, error) {
	buckets, err := a.base.List220VMessageBuckets(protocolIDs, filter)
	if err != nil {
		return adminv1.Message220VBuckets{}, err
	}
	return adminv1.ToMessage220VBuckets(buckets), nil
}

func (a *adminV1EventOverrideAdapter) SetMessage220VMode(uin int64, mode adminv1.Message220VMode) error {
	return a.base.SetMessage220VMode(uin, adminv1.ToContractsMessage220VMode(mode))
}

func (a *adminV1EventEmulationAdapter) ListDisplayBlockObjects(filter string) ([]adminv1.DisplayBlockObject, error) {
	items, err := a.base.ListDisplayBlockObjects(filter)
	if err != nil {
		return nil, err
	}
	return adminv1.ToDisplayBlockObjects(items), nil
}

func (a *adminV1EventEmulationAdapter) ListMessageProtocols() ([]int64, error) {
	return a.base.ListMessageProtocols()
}

func (a *adminV1EventEmulationAdapter) ListMessages(protocolID *int64, filter string) ([]adminv1.Message, error) {
	items, err := a.base.ListMessages(protocolID, filter)
	if err != nil {
		return nil, err
	}
	return adminv1.ToMessages(items), nil
}

func (a *adminV1EventEmulationAdapter) EmulateEvent(objn int64, zone int64, messageUIN int64) error {
	return a.base.EmulateEvent(objn, zone, messageUIN)
}
