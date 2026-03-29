package viewmodels

import "fyne.io/fyne/v2/data/binding"

// WorkAreaDeviceStateViewModel зберігає статичні поля вкладки "Стан" через binding.
type WorkAreaDeviceStateViewModel struct {
	deviceType  binding.String
	panelMark   binding.String
	groups      binding.String
	power       binding.String
	sim         binding.String
	autoTest    binding.String
	guard       binding.String
	channel     binding.String
	phone       binding.String
	akb         binding.String
	testControl binding.String
	notes       binding.String
	location    binding.String
}

func NewWorkAreaDeviceStateViewModel() *WorkAreaDeviceStateViewModel {
	vm := &WorkAreaDeviceStateViewModel{
		deviceType:  binding.NewString(),
		panelMark:   binding.NewString(),
		groups:      binding.NewString(),
		power:       binding.NewString(),
		sim:         binding.NewString(),
		autoTest:    binding.NewString(),
		guard:       binding.NewString(),
		channel:     binding.NewString(),
		phone:       binding.NewString(),
		akb:         binding.NewString(),
		testControl: binding.NewString(),
		notes:       binding.NewString(),
		location:    binding.NewString(),
	}
	vm.Reset()
	return vm
}

func (vm *WorkAreaDeviceStateViewModel) DeviceTypeBinding() binding.String { return vm.deviceType }
func (vm *WorkAreaDeviceStateViewModel) PanelMarkBinding() binding.String  { return vm.panelMark }
func (vm *WorkAreaDeviceStateViewModel) GroupsBinding() binding.String     { return vm.groups }
func (vm *WorkAreaDeviceStateViewModel) PowerBinding() binding.String      { return vm.power }
func (vm *WorkAreaDeviceStateViewModel) SIMBinding() binding.String        { return vm.sim }
func (vm *WorkAreaDeviceStateViewModel) AutoTestBinding() binding.String   { return vm.autoTest }
func (vm *WorkAreaDeviceStateViewModel) GuardBinding() binding.String      { return vm.guard }
func (vm *WorkAreaDeviceStateViewModel) ChannelBinding() binding.String    { return vm.channel }
func (vm *WorkAreaDeviceStateViewModel) PhoneBinding() binding.String      { return vm.phone }
func (vm *WorkAreaDeviceStateViewModel) AkbBinding() binding.String        { return vm.akb }
func (vm *WorkAreaDeviceStateViewModel) TestControlBinding() binding.String {
	return vm.testControl
}
func (vm *WorkAreaDeviceStateViewModel) NotesBinding() binding.String    { return vm.notes }
func (vm *WorkAreaDeviceStateViewModel) LocationBinding() binding.String { return vm.location }

func (vm *WorkAreaDeviceStateViewModel) Reset() {
	_ = vm.deviceType.Set("🔧 Тип: —")
	_ = vm.panelMark.Set("🏷️ Марка: —")
	_ = vm.groups.Set("🔐 Групи: —")
	_ = vm.power.Set("🔌 Живлення: —")
	_ = vm.sim.Set("📱 SIM: —")
	_ = vm.autoTest.Set("⏱️ Автотест: —")
	_ = vm.guard.Set("🔒 Стан: —")
	_ = vm.channel.Set("📡 Канал: —")
	_ = vm.phone.Set("☎️ Тел. об'єкта: —")
	_ = vm.akb.Set("🔋 АКБ: —")
	_ = vm.testControl.Set("⏲️ Контроль тесту: —")
	_ = vm.notes.Set("")
	_ = vm.location.Set("")
}

func (vm *WorkAreaDeviceStateViewModel) Apply(p WorkAreaDevicePresentation) {
	_ = vm.deviceType.Set(p.DeviceTypeText)
	_ = vm.panelMark.Set(p.PanelMarkText)
	_ = vm.groups.Set(p.GroupsText)
	_ = vm.power.Set(p.PowerText)
	_ = vm.sim.Set(p.SIMText)
	_ = vm.autoTest.Set(p.AutoTestText)
	_ = vm.guard.Set(p.GuardText)
	_ = vm.channel.Set(p.ChannelText)
	_ = vm.phone.Set(p.PhoneText)
	_ = vm.akb.Set(p.AkbText)
	_ = vm.testControl.Set(p.TestControlText)
	_ = vm.notes.Set(p.NotesText)
	_ = vm.location.Set(p.LocationText)
}
