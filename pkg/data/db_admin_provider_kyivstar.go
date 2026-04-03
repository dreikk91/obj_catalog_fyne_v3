package data

import (
	"errors"
	"obj_catalog_fyne_v3/pkg/contracts"
	"strings"
)

func (p *DBDataProvider) GetKyivstarAuthState() (contracts.KyivstarAuthState, error) {
	service, err := p.kyivstarService()
	if err != nil {
		return contracts.KyivstarAuthState{}, err
	}
	return service.AuthState()
}

func (p *DBDataProvider) RefreshKyivstarToken() (contracts.KyivstarAuthState, error) {
	service, err := p.kyivstarService()
	if err != nil {
		return contracts.KyivstarAuthState{}, err
	}
	return service.RefreshToken()
}

func (p *DBDataProvider) ClearKyivstarToken() error {
	service, err := p.kyivstarService()
	if err != nil {
		return err
	}
	return service.ClearToken()
}

func (p *DBDataProvider) GetKyivstarSIMStatus(msisdn string) (contracts.KyivstarSIMStatus, error) {
	service, err := p.kyivstarService()
	if err != nil {
		return contracts.KyivstarSIMStatus{}, err
	}
	return service.GetSIMStatus(msisdn)
}

func (p *DBDataProvider) PauseKyivstarSIM(msisdn string) (contracts.KyivstarSIMOperationResult, error) {
	service, err := p.kyivstarService()
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}
	return service.PauseSIM(msisdn)
}

func (p *DBDataProvider) ActivateKyivstarSIM(msisdn string) (contracts.KyivstarSIMOperationResult, error) {
	service, err := p.kyivstarService()
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}
	return service.ActivateSIM(msisdn)
}

func (p *DBDataProvider) PauseKyivstarSIMServices(msisdn string, serviceIDs []string) (contracts.KyivstarSIMOperationResult, error) {
	service, err := p.kyivstarService()
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}
	return service.PauseSIMServices(msisdn, serviceIDs)
}

func (p *DBDataProvider) ActivateKyivstarSIMServices(msisdn string, serviceIDs []string) (contracts.KyivstarSIMOperationResult, error) {
	service, err := p.kyivstarService()
	if err != nil {
		return contracts.KyivstarSIMOperationResult{}, err
	}
	return service.ActivateSIMServices(msisdn, serviceIDs)
}

func (p *DBDataProvider) RebootKyivstarSIM(msisdn string) (contracts.KyivstarSIMResetResult, error) {
	service, err := p.kyivstarService()
	if err != nil {
		return contracts.KyivstarSIMResetResult{}, err
	}
	return service.RebootSIM(msisdn)
}

func (p *DBDataProvider) UpdateKyivstarSIMMetadata(msisdn string, deviceName string, deviceID string) error {
	service, err := p.kyivstarService()
	if err != nil {
		return err
	}
	return service.UpdateSIMMetadata(msisdn, deviceName, deviceID)
}

func (p *DBDataProvider) kyivstarService() (*KyivstarService, error) {
	if p == nil || p.kyivstar == nil {
		return nil, errors.New("kyivstar: сервіс не налаштований")
	}
	if strings.TrimSpace(p.kyivstar.baseURL) == "" {
		return nil, errors.New("kyivstar: сервіс не налаштований")
	}
	return p.kyivstar, nil
}
