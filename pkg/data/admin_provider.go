package data

import "obj_catalog_fyne_v3/pkg/contracts"

// Backward-compatible aliases. Prefer pkg/contracts in new code.
type DisplayBlockMode = contracts.DisplayBlockMode

const (
	DisplayBlockNone         = contracts.DisplayBlockNone
	DisplayBlockTemporaryOff = contracts.DisplayBlockTemporaryOff
	DisplayBlockDebug        = contracts.DisplayBlockDebug
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
type Admin220VMode = contracts.Admin220VMode
type Admin220VMessageBuckets = contracts.Admin220VMessageBuckets
type DisplayBlockObject = contracts.DisplayBlockObject
type AdminAccessStatus = contracts.AdminAccessStatus
type AdminDataCheckIssue = contracts.AdminDataCheckIssue

const (
	Admin220VNone    = contracts.Admin220VNone
	Admin220VAlarm   = contracts.Admin220VAlarm
	Admin220VRestore = contracts.Admin220VRestore
)

type AdminStatisticsConnectionMode = contracts.AdminStatisticsConnectionMode

const (
	StatsConnectionAll     = contracts.StatsConnectionAll
	StatsConnectionOnline  = contracts.StatsConnectionOnline
	StatsConnectionOffline = contracts.StatsConnectionOffline
)

type AdminStatisticsProtocolFilter = contracts.AdminStatisticsProtocolFilter

const (
	StatsProtocolAll      = contracts.StatsProtocolAll
	StatsProtocolAutodial = contracts.StatsProtocolAutodial
	StatsProtocolMost     = contracts.StatsProtocolMost
	StatsProtocolNova     = contracts.StatsProtocolNova
)

type AdminStatisticsFilter = contracts.AdminStatisticsFilter
type AdminStatisticsRow = contracts.AdminStatisticsRow
type AdminProvider = contracts.AdminProvider
