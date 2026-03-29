package eventbus

// ObjectSavedEvent описує факт створення/редагування об'єкта.
type ObjectSavedEvent struct {
	ObjectID int64
}

// ObjectDeletedEvent описує факт видалення об'єкта.
type ObjectDeletedEvent struct {
	ObjectID int64
}

// DataRefreshEvent описує, які секції потрібно оновити у UI.
type DataRefreshEvent struct {
	RefreshObjects bool
	RefreshAlarms  bool
	RefreshEvents  bool
}
