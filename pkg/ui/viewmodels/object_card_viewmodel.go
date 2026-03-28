package viewmodels

import (
	"fmt"
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// ObjectCardInput описує стан полів форми картки об'єкта без прив'язки до Fyne-віджетів.
type ObjectCardInput struct {
	ObjNRaw       string
	ShortName     string
	FullName      string
	Address       string
	Phones        string
	Contract      string
	StartDate     string
	Location      string
	Notes         string
	GSMPhone1     string
	GSMPhone2     string
	GSMHiddenNRaw string

	ChannelCode        int64
	TestControlEnabled bool
	TestIntervalMinRaw string

	ObjTypeID  int64
	ObjRegID   int64
	PPKID      int64
	SubServerA string
	SubServerB string
}

// ObjectCardViewModel містить правила поведінки картки об'єкта.
type ObjectCardViewModel struct {
	channelCode             int64
	autoUpdatingFullName    bool
	fullNameSyncedWithShort bool
}

func NewObjectCardViewModel() *ObjectCardViewModel {
	return &ObjectCardViewModel{
		channelCode:             1,
		fullNameSyncedWithShort: true,
	}
}

func (vm *ObjectCardViewModel) SetChannelCode(channelCode int64) {
	vm.channelCode = channelCode
}

func (vm *ObjectCardViewModel) ShouldShowHiddenNumber() bool {
	return vm.channelCode == 5
}

func (vm *ObjectCardViewModel) OnFullNameChanged(fullName string, shortName string) {
	if vm.autoUpdatingFullName {
		return
	}
	vm.fullNameSyncedWithShort = strings.TrimSpace(fullName) == strings.TrimSpace(shortName)
}

func (vm *ObjectCardViewModel) OnShortNameChanged(shortName string) (string, bool) {
	if vm.autoUpdatingFullName || !vm.fullNameSyncedWithShort {
		return "", false
	}

	vm.autoUpdatingFullName = true
	defer func() {
		vm.autoUpdatingFullName = false
	}()
	return strings.TrimSpace(shortName), true
}

func (vm *ObjectCardViewModel) ResetNameSync(shortName string, fullName string) {
	vm.fullNameSyncedWithShort = strings.TrimSpace(shortName) == strings.TrimSpace(fullName)
}

func (vm *ObjectCardViewModel) ValidateAndBuildCard(input ObjectCardInput) (contracts.AdminObjectCard, error) {
	var card contracts.AdminObjectCard

	objn, err := strconv.ParseInt(strings.TrimSpace(input.ObjNRaw), 10, 64)
	if err != nil {
		return card, fmt.Errorf("некоректний об'єктовий номер")
	}
	if input.ChannelCode <= 0 {
		return card, fmt.Errorf("виберіть канал зв'язку")
	}
	if input.ObjTypeID <= 0 {
		return card, fmt.Errorf("виберіть тип об'єкта")
	}

	card.ObjN = objn
	card.GrpN = 1
	card.ShortName = strings.TrimSpace(input.ShortName)
	card.FullName = strings.TrimSpace(input.FullName)
	card.Address = strings.TrimSpace(input.Address)
	card.Phones = strings.TrimSpace(input.Phones)
	card.Contract = strings.TrimSpace(input.Contract)
	card.StartDate = strings.TrimSpace(input.StartDate)
	card.Location = strings.TrimSpace(input.Location)
	card.Notes = strings.TrimSpace(input.Notes)
	card.GSMPhone1 = strings.TrimSpace(input.GSMPhone1)
	card.GSMPhone2 = strings.TrimSpace(input.GSMPhone2)
	card.ChannelCode = input.ChannelCode
	card.TestControlEnabled = input.TestControlEnabled
	card.ObjTypeID = input.ObjTypeID
	card.ObjRegID = input.ObjRegID
	card.PPKID = input.PPKID
	card.SubServerA = strings.TrimSpace(input.SubServerA)
	card.SubServerB = strings.TrimSpace(input.SubServerB)

	if input.ChannelCode == 5 {
		hiddenN, parseErr := strconv.ParseInt(strings.TrimSpace(input.GSMHiddenNRaw), 10, 64)
		if parseErr != nil || hiddenN <= 0 {
			return card, fmt.Errorf("для каналу 5 вкажіть коректний прихований номер")
		}
		card.GSMHiddenN = hiddenN
	}

	if card.TestControlEnabled {
		testInterval, parseErr := strconv.ParseInt(strings.TrimSpace(input.TestIntervalMinRaw), 10, 64)
		if parseErr != nil || testInterval <= 0 {
			return card, fmt.Errorf("некоректний інтервал контролю тесту")
		}
		card.TestIntervalMin = testInterval
	}

	return card, nil
}
