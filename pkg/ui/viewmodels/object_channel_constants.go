package viewmodels

const (
	ObjectChannelCodeAutoDial int64 = 1
	ObjectChannelCodeGPRS     int64 = 5

	ObjectChannelLabelAutoDial = "1 - Автододзвон"
	ObjectChannelLabelGPRS     = "5 - GPRS"
)

// ObjectChannelOptions повертає стандартний список каналів для Select.
func ObjectChannelOptions() []string {
	return []string{
		ObjectChannelLabelAutoDial,
		ObjectChannelLabelGPRS,
	}
}

// DefaultObjectChannelLabelToCode повертає стандартне відображення label -> code.
func DefaultObjectChannelLabelToCode() map[string]int64 {
	return map[string]int64{
		ObjectChannelLabelAutoDial: ObjectChannelCodeAutoDial,
		ObjectChannelLabelGPRS:     ObjectChannelCodeGPRS,
	}
}

// DefaultObjectChannelCodeToLabel повертає стандартне відображення code -> label.
func DefaultObjectChannelCodeToLabel() map[int64]string {
	return map[int64]string{
		ObjectChannelCodeAutoDial: ObjectChannelLabelAutoDial,
		ObjectChannelCodeGPRS:     ObjectChannelLabelGPRS,
	}
}

// ResolveObjectChannel повертає label і code з fallback до стандартного автододзвону.
func ResolveObjectChannel(channelCode int64, channelCodeToLabel map[int64]string) (string, int64) {
	if label, ok := channelCodeToLabel[channelCode]; ok {
		return label, channelCode
	}
	if label, ok := channelCodeToLabel[ObjectChannelCodeAutoDial]; ok {
		return label, ObjectChannelCodeAutoDial
	}
	defaults := DefaultObjectChannelCodeToLabel()
	return defaults[ObjectChannelCodeAutoDial], ObjectChannelCodeAutoDial
}
