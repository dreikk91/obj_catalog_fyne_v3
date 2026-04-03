package data

const phoenixObjectsListQuery = `
SELECT
	G.Panel_id AS panel_id,
	G.Group_ AS group_no,
	G.Message AS group_name,
	G.IsOpen AS is_open,
	G.TimeEvent AS group_time_event,
	G.disabled AS group_disabled,
	C.CompanyName AS company_name,
	C.Address AS company_address,
	C.Telephones AS telephones,
	CAST(NULL AS nvarchar(255)) AS type_name,
	P.Disabled AS panel_disabled,
	P.TestPanel AS test_panel,
	P.Panel_type AS panel_type,
	T.StateEvent AS state_event,
	P.CreateDate AS create_date,
	P.DateLastChange AS date_last_change,
	CAST(NULL AS nvarchar(255)) AS engineer_name
FROM Groups G WITH (NOLOCK)
LEFT JOIN Company C WITH (NOLOCK) ON C.ID = G.CompanyID
LEFT JOIN (
	SELECT Panel_id, Group_, MAX(StateEvent) AS StateEvent
	FROM Temp WITH (NOLOCK)
	GROUP BY Panel_id, Group_
) T ON T.Panel_id = G.Panel_id AND T.Group_ = G.Group_
INNER JOIN vwRealPanel P WITH (NOLOCK) ON P.Panel_id = G.Panel_id
ORDER BY G.Panel_id, G.Group_
`

const phoenixObjectDetailGroupsQuery = `
SELECT
	G.Panel_id AS panel_id,
	G.Group_ AS group_no,
	G.Message AS group_name,
	G.IsOpen AS is_open,
	G.TimeEvent AS group_time_event,
	G.disabled AS group_disabled,
	C.CompanyName AS company_name,
	C.Address AS company_address,
	C.Telephones AS telephones,
	CT.Name AS type_name,
	P.Disabled AS panel_disabled,
	P.TestPanel AS test_panel,
	P.Panel_type AS panel_type,
	T.StateEvent AS state_event,
	P.CreateDate AS create_date,
	P.DateLastChange AS date_last_change,
	E.engineer_name AS engineer_name
FROM Groups G WITH (NOLOCK)
LEFT JOIN Company C WITH (NOLOCK) ON C.ID = G.CompanyID
LEFT JOIN CompanyType CT WITH (NOLOCK) ON CT.ID = C.TypeID
LEFT JOIN (
	SELECT Panel_id, Group_, MAX(StateEvent) AS StateEvent
	FROM Temp WITH (NOLOCK)
	GROUP BY Panel_id, Group_
) T ON T.Panel_id = G.Panel_id AND T.Group_ = G.Group_
INNER JOIN vwRealPanel P WITH (NOLOCK) ON P.Panel_id = G.Panel_id
LEFT JOIN Engineers E WITH (NOLOCK) ON E.Work_Panel_id = G.Panel_id
WHERE G.Panel_id = @p1
ORDER BY G.Group_
`

const phoenixChannelInfoQuery = `
SELECT TOP (1)
	MDC.Panel_id AS panel_id,
	MDC.Message AS device_name,
	MDC.ChannelType AS channel_type,
	MDC.ChannelNo AS channel_no,
	MDC.LastTest AS last_test,
	M.SignalLevel AS signal_level,
	MDC.DeviceVersion AS device_version,
	MDC.RadioVersion AS radio_version,
	S.RealSimNumber AS sim_number,
	S.SimNumber AS fallback_sim_number
FROM vwMPhoneDeviceChannels MDC WITH (NOLOCK)
INNER JOIN MPhone M WITH (NOLOCK) ON M.Mphone_id = MDC.Mphone_id
LEFT JOIN Sim S WITH (NOLOCK) ON
	S.OnBoardDevice_ID = MDC.OnBoardDevice_ID
	AND (S.IsCurrentSim = 1 OR S.IsCurrentSim IS NULL)
WHERE
	MDC.Panel_id = @p1
	AND MDC.IsActive = 1
ORDER BY
	CASE MDC.ChannelType_id
		WHEN 2 THEN 1
		WHEN 6 THEN 2
		WHEN 4 THEN 3
		WHEN 1 THEN 4
		WHEN 5 THEN 5
		WHEN 3 THEN 6
		ELSE 100
	END,
	MDC.Channel_ID
`

const phoenixZonesQuery = `
SELECT
	Z.Panel_id AS panel_id,
	Z.Group_ AS group_no,
	G.Message AS group_name,
	G.IsOpen AS group_is_open,
	G.disabled AS group_disabled,
	P.Disabled AS panel_disabled,
	P.TestPanel AS test_panel,
	G.TimeEvent AS group_time_event,
	Z.Zone AS zone_no,
	Z.Message AS zone_name,
	Z.Status AS status,
	Z.IsPatrol AS is_patrol,
	Z.IsAlarmButton AS is_alarm_button,
	Z.IsBypass AS is_bypass,
	Z.SignalLevel AS signal_level,
	Z.RadioZoneTypeid AS zone_type_id
FROM Zones Z WITH (NOLOCK)
INNER JOIN Groups G WITH (NOLOCK) ON G.Panel_id = Z.Panel_id AND G.Group_ = Z.Group_
INNER JOIN vwRealPanel P WITH (NOLOCK) ON P.Panel_id = Z.Panel_id
WHERE Z.Panel_id = @p1
ORDER BY Z.Group_, Z.Zone
`

const phoenixResponsiblesQuery = `
SELECT
	R.panel_id AS panel_id,
	R.Group_ AS group_no,
	G.Message AS group_name,
	G.IsOpen AS group_is_open,
	G.disabled AS group_disabled,
	P.Disabled AS panel_disabled,
	P.TestPanel AS test_panel,
	R.Responsible_Number AS responsible_number,
	RL.Responsible_Name AS responsible_name,
	RL.Responsible_Address AS responsible_address,
	ISNULL(RTD.CallOrder, R.Responsible_Number) AS call_order,
	ISNULL(RTD.Description, RTT.TypeTel) AS contact_label,
	RT.PhoneNo AS contact_value,
	CAST('phone' AS nvarchar(16)) AS contact_kind
FROM Responsibles R WITH (NOLOCK)
INNER JOIN ResponsiblesList RL WITH (NOLOCK) ON RL.ResponsiblesList_id = R.ResponsiblesList_id
INNER JOIN Groups G WITH (NOLOCK) ON G.Panel_id = R.panel_id AND G.Group_ = R.Group_
INNER JOIN vwRealPanel P WITH (NOLOCK) ON P.Panel_id = R.panel_id
INNER JOIN ResponsibleTel RT WITH (NOLOCK) ON RT.ResponsiblesList_id = RL.ResponsiblesList_id
LEFT JOIN ResponsibleTypeTel RTT WITH (NOLOCK) ON RTT.TypeTel_id = RT.TypeTel_id
LEFT JOIN ResponsibleTelDescription RTD WITH (NOLOCK) ON
	RTD.Responsible_id = R.Responsible_id
	AND RTD.ResponsibleTel_id = RT.ResponsibleTel_id
WHERE
	R.panel_id = @p1
	AND LTRIM(RTRIM(ISNULL(RT.PhoneNo, ''))) <> ''
UNION ALL
SELECT
	R.panel_id AS panel_id,
	R.Group_ AS group_no,
	G.Message AS group_name,
	G.IsOpen AS group_is_open,
	G.disabled AS group_disabled,
	P.Disabled AS panel_disabled,
	P.TestPanel AS test_panel,
	R.Responsible_Number AS responsible_number,
	RL.Responsible_Name AS responsible_name,
	RL.Responsible_Address AS responsible_address,
	R.Responsible_Number AS call_order,
	CAST('email' AS nvarchar(32)) AS contact_label,
	RE.EmailAddr AS contact_value,
	CAST('email' AS nvarchar(16)) AS contact_kind
FROM Responsibles R WITH (NOLOCK)
INNER JOIN ResponsiblesList RL WITH (NOLOCK) ON RL.ResponsiblesList_id = R.ResponsiblesList_id
INNER JOIN Groups G WITH (NOLOCK) ON G.Panel_id = R.panel_id AND G.Group_ = R.Group_
INNER JOIN vwRealPanel P WITH (NOLOCK) ON P.Panel_id = R.panel_id
INNER JOIN ResponsibleEmail RE WITH (NOLOCK) ON RE.ResponsiblesList_id = RL.ResponsiblesList_id
WHERE
	R.panel_id = @p1
	AND LTRIM(RTRIM(ISNULL(RE.EmailAddr, ''))) <> ''
ORDER BY group_no, responsible_number, call_order, contact_kind, contact_value
`

const phoenixLatestEventIDQuery = `
SELECT ISNULL(MAX(Event_id), 0) AS last_event_id
FROM vwArchives WITH (NOLOCK)
`

const phoenixInitialEventsQuery = `
SELECT TOP (500)
	A.Event_id AS event_id,
	A.Panel_id AS panel_id,
	A.Group_ AS group_no,
	A.Zone AS zone_no,
	A.TimeEvent AS time_event,
	A.Code AS event_code,
	C.Message AS code_message,
	TC.idTCode AS type_code_id,
	G.Message AS group_name,
	Z.Message AS zone_name,
	Co.CompanyName AS company_name,
	Co.Address AS company_address
FROM vwArchives A WITH (NOLOCK)
LEFT JOIN Groups G WITH (NOLOCK) ON G.Panel_id = A.Panel_id AND G.Group_ = A.Group_
LEFT JOIN Code C WITH (NOLOCK) ON C.Code = A.Code AND C.CodeGroup = A.CodeGroup
LEFT JOIN TypeCode TC WITH (NOLOCK) ON TC.idTCode = C.idTCode
LEFT JOIN Zones Z WITH (NOLOCK) ON Z.Panel_id = A.Panel_id AND Z.Group_ = A.Group_ AND Z.Zone = A.Zone
LEFT JOIN Company Co WITH (NOLOCK) ON Co.ID = G.CompanyID
ORDER BY A.Event_id DESC
`

const phoenixIncrementalEventsQuery = `
SELECT
	A.Event_id AS event_id,
	A.Panel_id AS panel_id,
	A.Group_ AS group_no,
	A.Zone AS zone_no,
	A.TimeEvent AS time_event,
	A.Code AS event_code,
	C.Message AS code_message,
	TC.idTCode AS type_code_id,
	G.Message AS group_name,
	Z.Message AS zone_name,
	Co.CompanyName AS company_name,
	Co.Address AS company_address
FROM vwArchives A WITH (NOLOCK)
LEFT JOIN Groups G WITH (NOLOCK) ON G.Panel_id = A.Panel_id AND G.Group_ = A.Group_
LEFT JOIN Code C WITH (NOLOCK) ON C.Code = A.Code AND C.CodeGroup = A.CodeGroup
LEFT JOIN TypeCode TC WITH (NOLOCK) ON TC.idTCode = C.idTCode
LEFT JOIN Zones Z WITH (NOLOCK) ON Z.Panel_id = A.Panel_id AND Z.Group_ = A.Group_ AND Z.Zone = A.Zone
LEFT JOIN Company Co WITH (NOLOCK) ON Co.ID = G.CompanyID
WHERE A.Event_id > @p1
ORDER BY A.Event_id ASC
`

const phoenixObjectEventsQuery = `
SELECT TOP (500)
	A.Event_id AS event_id,
	A.Panel_id AS panel_id,
	A.Group_ AS group_no,
	A.Zone AS zone_no,
	A.TimeEvent AS time_event,
	A.Code AS event_code,
	C.Message AS code_message,
	TC.idTCode AS type_code_id,
	G.Message AS group_name,
	Z.Message AS zone_name,
	Co.CompanyName AS company_name,
	Co.Address AS company_address
FROM vwArchives A WITH (NOLOCK)
LEFT JOIN Groups G WITH (NOLOCK) ON G.Panel_id = A.Panel_id AND G.Group_ = A.Group_
LEFT JOIN Code C WITH (NOLOCK) ON C.Code = A.Code AND C.CodeGroup = A.CodeGroup
LEFT JOIN TypeCode TC WITH (NOLOCK) ON TC.idTCode = C.idTCode
LEFT JOIN Zones Z WITH (NOLOCK) ON Z.Panel_id = A.Panel_id AND Z.Group_ = A.Group_ AND Z.Zone = A.Zone
LEFT JOIN Company Co WITH (NOLOCK) ON Co.ID = G.CompanyID
WHERE A.Panel_id = @p1
ORDER BY A.Event_id DESC
`
