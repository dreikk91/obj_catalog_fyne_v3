package data

import "obj_catalog_fyne_v3/pkg/contracts"

type DisplayBlockMode = contracts.DisplayBlockMode

const (
	DisplayBlockNone         DisplayBlockMode = contracts.DisplayBlockNone
	DisplayBlockTemporaryOff DisplayBlockMode = contracts.DisplayBlockTemporaryOff
	DisplayBlockDebug        DisplayBlockMode = contracts.DisplayBlockDebug
)

type FireMonitoringServer = contracts.FireMonitoringServer
type FireMonitoringSettings = contracts.FireMonitoringSettings
type PPKConstructorItem = contracts.PPKConstructorItem
type AdminObjectCard = contracts.AdminObjectCard
type AdminSubServer = contracts.AdminSubServer
type AdminSubServerObject = contracts.AdminSubServerObject
type AdminObjectPersonal = contracts.AdminObjectPersonal
type AdminObjectZone = contracts.AdminObjectZone
type AdminObjectCoordinates = contracts.AdminObjectCoordinates
type AdminSIMPhoneUsage = contracts.AdminSIMPhoneUsage
type DictionaryItem = contracts.DictionaryItem
type AdminMessage = contracts.AdminMessage
type DisplayBlockObject = contracts.DisplayBlockObject
type AdminAccessStatus = contracts.AdminAccessStatus
type AdminDataCheckIssue = contracts.AdminDataCheckIssue

type AdminStatisticsConnectionMode = contracts.AdminStatisticsConnectionMode

const (
	StatsConnectionAll     AdminStatisticsConnectionMode = contracts.StatsConnectionAll
	StatsConnectionOnline  AdminStatisticsConnectionMode = contracts.StatsConnectionOnline
	StatsConnectionOffline AdminStatisticsConnectionMode = contracts.StatsConnectionOffline
)

type AdminStatisticsProtocolFilter = contracts.AdminStatisticsProtocolFilter

const (
	StatsProtocolAll      AdminStatisticsProtocolFilter = contracts.StatsProtocolAll
	StatsProtocolAutodial AdminStatisticsProtocolFilter = contracts.StatsProtocolAutodial
	StatsProtocolMost     AdminStatisticsProtocolFilter = contracts.StatsProtocolMost
	StatsProtocolNova     AdminStatisticsProtocolFilter = contracts.StatsProtocolNova
)

type AdminStatisticsFilter = contracts.AdminStatisticsFilter
type AdminStatisticsRow = contracts.AdminStatisticsRow

type AdminProvider = contracts.AdminProvider
