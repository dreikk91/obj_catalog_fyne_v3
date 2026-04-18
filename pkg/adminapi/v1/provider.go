package v1

type StatisticsProvider interface {
	CollectObjectStatistics(filter StatisticsFilter, limit int) ([]StatisticsRow, error)
	ListObjectTypes() ([]DictionaryItem, error)
	ListObjectDistricts() ([]DictionaryItem, error)
}

type DisplayBlockingProvider interface {
	ListDisplayBlockObjects(filter string) ([]DisplayBlockObject, error)
	SetDisplayBlockMode(objn int64, mode DisplayBlockMode) error
}

type DisplayBlockObjectLookupProvider interface {
	ListDisplayBlockObjects(filter string) ([]DisplayBlockObject, error)
}

type MessageLookupProvider interface {
	ListMessageProtocols() ([]int64, error)
	ListMessages(protocolID *int64, filter string) ([]Message, error)
}

type MessagesProvider interface {
	MessageLookupProvider
	SetMessageAdminOnly(uin int64, adminOnly bool) error
}

type Message220VProvider interface {
	List220VMessageBuckets(protocolIDs []int64, filter string) (Message220VBuckets, error)
	SetMessage220VMode(uin int64, mode Message220VMode) error
}

type EventOverrideProvider interface {
	MessagesProvider
	Message220VProvider
	SetMessageCategory(uin int64, sc1 *int64) error
}

type EventEmulationProvider interface {
	DisplayBlockObjectLookupProvider
	MessageLookupProvider
	EmulateEvent(objn int64, zone int64, messageUIN int64) error
}
