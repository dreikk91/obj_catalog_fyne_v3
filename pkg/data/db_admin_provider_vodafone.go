package data

import (
	"errors"
	"obj_catalog_fyne_v3/pkg/contracts"
	"strings"
)

func (p *DBDataProvider) GetVodafoneAuthState() (contracts.VodafoneAuthState, error) {
	service, err := p.vodafoneService()
	if err != nil {
		return contracts.VodafoneAuthState{}, err
	}
	return service.AuthState()
}

func (p *DBDataProvider) RequestVodafoneLoginSMS(phone string) error {
	service, err := p.vodafoneService()
	if err != nil {
		return err
	}
	return service.RequestLoginSMS(phone)
}

func (p *DBDataProvider) VerifyVodafoneLogin(phone string, code string) (contracts.VodafoneAuthState, error) {
	service, err := p.vodafoneService()
	if err != nil {
		return contracts.VodafoneAuthState{}, err
	}
	return service.VerifyLogin(phone, code)
}

func (p *DBDataProvider) ClearVodafoneLogin() error {
	service, err := p.vodafoneService()
	if err != nil {
		return err
	}
	return service.ClearLogin()
}

func (p *DBDataProvider) GetVodafoneSIMStatus(msisdn string) (contracts.VodafoneSIMStatus, error) {
	service, err := p.vodafoneService()
	if err != nil {
		return contracts.VodafoneSIMStatus{}, err
	}
	return service.GetSIMStatus(msisdn)
}

func (p *DBDataProvider) RebootVodafoneSIM(msisdn string) (contracts.VodafoneSIMRebootResult, error) {
	service, err := p.vodafoneService()
	if err != nil {
		return contracts.VodafoneSIMRebootResult{}, err
	}
	return service.RebootSIM(msisdn)
}

func (p *DBDataProvider) UpdateVodafoneSIMMetadata(msisdn string, name string, comment string) error {
	service, err := p.vodafoneService()
	if err != nil {
		return err
	}
	return service.UpdateSIMMetadata(msisdn, name, comment)
}

func (p *DBDataProvider) vodafoneService() (*VodafoneService, error) {
	if p == nil || p.vodafone == nil {
		return nil, errors.New("vodafone: сервіс не налаштований")
	}
	if strings.TrimSpace(p.vodafone.baseURL) == "" {
		return nil, errors.New("vodafone: сервіс не налаштований")
	}
	return p.vodafone, nil
}
