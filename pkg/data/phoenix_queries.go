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
	COALESCE(ChannelMeta.channel_type_name, MDC.ChannelType) AS channel_type,
	COALESCE(ChannelMeta.channel_no, MDC.ChannelNo) AS channel_no,
	COALESCE(ChannelMeta.last_test, MDC.LastTest) AS last_test,
	ChannelMeta.test_timeout AS test_timeout,
	ChannelMeta.open_internet_channel_id AS open_internet_channel_id,
	M.SignalLevel AS signal_level,
	MDC.DeviceVersion AS device_version,
	MDC.RadioVersion AS radio_version,
	PrimarySIM.sim_number AS sim1_number,
	PrimarySIM.operator_name AS sim1_operator_name,
	SecondarySIM.sim_number AS sim2_number,
	SecondarySIM.operator_name AS sim2_operator_name
FROM vwMPhoneDeviceChannels MDC WITH (NOLOCK)
INNER JOIN MPhone M WITH (NOLOCK) ON M.Mphone_id = MDC.Mphone_id
OUTER APPLY (
	SELECT TOP (1)
		cno.ChannelNo AS channel_no,
		CASE
			WHEN i.OpenInternetChannel_ID IS NULL THEN crch.CentralReceiverChannel_ID
			ELSE i.OpenInternetChannel_ID
		END AS open_internet_channel_id,
		c.LastTest AS last_test,
		c.TestTimeout AS test_timeout,
		ct.ChannelType AS channel_type_name
	FROM Channel AS c WITH (NOLOCK)
	INNER JOIN ChannelNo AS cno WITH (NOLOCK) ON
		cno.ChannelNo_ID = c.ChannelNo_ID
	INNER JOIN ChannelTypes AS ct WITH (NOLOCK) ON
		ct.ChannelType_id = c.ChannelType_id
	LEFT JOIN OpenInternetChannel AS i WITH (NOLOCK) ON
		i.Channel_ID = c.Channel_ID
	LEFT JOIN CentralReceiverChannel AS crch WITH (NOLOCK) ON
		crch.Channel_ID = c.Channel_ID
	WHERE c.Channel_ID = MDC.Channel_ID
) AS ChannelMeta
OUTER APPLY (
	SELECT TOP (1)
		COALESCE(NULLIF(LTRIM(RTRIM(S.RealSimNumber)), ''), NULLIF(LTRIM(RTRIM(S.SimNumber)), '')) AS sim_number,
		OP.OperatorName AS operator_name
	FROM OnBoardDevice D WITH (NOLOCK)
	INNER JOIN OnBoardDeviceType DT WITH (NOLOCK) ON DT.OnBoardDeviceType_ID = D.OnBoardDeviceType_ID
	INNER JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = D.OnBoardDevice_ID
	LEFT JOIN Operators OP WITH (NOLOCK) ON OP.Operator_ID = S.Operator_ID
	WHERE
		D.MPhone_ID = MDC.Mphone_id
		AND DT.Enum = 0
		AND COALESCE(NULLIF(LTRIM(RTRIM(S.RealSimNumber)), ''), NULLIF(LTRIM(RTRIM(S.SimNumber)), '')) IS NOT NULL
	ORDER BY
		CASE WHEN ISNULL(S.IsCurrentSim, 0) = 1 THEN 0 ELSE 1 END,
		CASE WHEN ISNULL(S.IsMainSim, 0) = 1 THEN 0 ELSE 1 END,
		S.Sim_ID
) AS PrimarySIM
OUTER APPLY (
	SELECT TOP (1)
		COALESCE(NULLIF(LTRIM(RTRIM(S.RealSimNumber)), ''), NULLIF(LTRIM(RTRIM(S.SimNumber)), '')) AS sim_number,
		OP.OperatorName AS operator_name
	FROM OnBoardDevice D WITH (NOLOCK)
	INNER JOIN OnBoardDeviceType DT WITH (NOLOCK) ON DT.OnBoardDeviceType_ID = D.OnBoardDeviceType_ID
	INNER JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = D.OnBoardDevice_ID
	LEFT JOIN Operators OP WITH (NOLOCK) ON OP.Operator_ID = S.Operator_ID
	WHERE
		D.MPhone_ID = MDC.Mphone_id
		AND DT.Enum = 1
		AND COALESCE(NULLIF(LTRIM(RTRIM(S.RealSimNumber)), ''), NULLIF(LTRIM(RTRIM(S.SimNumber)), '')) IS NOT NULL
	ORDER BY
		CASE WHEN ISNULL(S.IsCurrentSim, 0) = 1 THEN 0 ELSE 1 END,
		CASE WHEN ISNULL(S.IsMainSim, 0) = 1 THEN 0 ELSE 1 END,
		S.Sim_ID
) AS SecondarySIM
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

const phoenixActiveAlarmsQuery = `
SELECT
	T.Event_id AS event_id,
	T.Panel_id AS panel_id,
	T.Group_ AS group_no,
	T.Zone AS zone_no,
	T.TimeEvent AS time_event,
	CASE
		WHEN T.Code IN ('LIN','LOFF') THEN NULL
		WHEN COALESCE(G.Message, '') <> '' THEN G.Message
		WHEN COALESCE(GR.DESCRIPTION, '') <> '' THEN GR.DESCRIPTION
		ELSE NULL
	END AS group_message,
	T.Code AS event_code,
	CASE
		WHEN C.Code IN ('Z34','Z35') THEN C.Message
		WHEN TC.idTCode IN (1,2,14,105,126) AND COALESCE(Z.Message, '') <> '' THEN Z.Message
		WHEN TC.idTCode IN (26,27,131) THEN C.Message + COALESCE(SPACE(1) + Z.Message, '')
		WHEN COALESCE(T.Zone,0) = 0 THEN C.Message + COALESCE(SPACE(1) + T.MeterCount, '')
		WHEN TC.idTCode IN (20,36,41,42) THEN C.Message + COALESCE(SPACE(1) + T.MeterCount, '')
		WHEN (C.AccessCode = '1') AND COALESCE(U.UserName,'') <> '' THEN U.UserName
		WHEN (C.AccessCode = '0') AND COALESCE(ZGR.Message, '') <> '' THEN ZGR.Message
		WHEN TC.idTCode IN (151,152) THEN C.Message + COALESCE(SPACE(1) + (
			SELECT TOP (1) LO.OutputName
			FROM Mphone M WITH (NOLOCK)
			INNER JOIN LunOutput LO WITH (NOLOCK) ON LO.Mphone_id = M.Mphone_id
			WHERE M.Panel_id = T.Panel_id AND LO.OutputNum = T.Zone
		), '')
		ELSE C.Message + COALESCE(SPACE(1) + T.MeterCount, '')
	END AS code_message,
	TC.idTCode AS type_code_id,
	TC.Message AS type_code_message,
	G.Message AS group_name,
	Z.Message AS zone_name,
	T.Line AS line,
	T.Event_Parent_id AS event_parent_id,
	T.StateEvent AS state_event,
	T.Priority AS priority,
	Co.CompanyName AS company_name,
	Co.Address AS company_address,
	CASE
		WHEN TC.idTCode IN (1,2) AND COALESCE(Z.IsAlarmButton,'0') <> '0' THEN Z.IsAlarmButton
		WHEN COALESCE(ZGR.IsAlarmButton,'0') <> '0' THEN ZGR.IsAlarmButton
		ELSE NULL
	END AS is_alarm_button,
	CASE
		WHEN T.BitMask & 2 = 2 THEN 5
		WHEN COALESCE(T.StateEvent, 0) IN (2,3) THEN 1
		WHEN COALESCE(P.TestPanel, '0') = '1' THEN 2
		WHEN COALESCE(P.Disabled, '0') = '1' THEN 3
		ELSE 0
	END AS object_status,
	CASE
		WHEN T.BitMask & 4 = 4 THEN 1
		ELSE 0
	END AS unknown_object,
	G.disabled AS group_disabled,
	P.Disabled AS panel_disabled,
	P.TestPanel AS test_panel
FROM Temp T WITH (NOLOCK)
LEFT JOIN Groups G WITH (NOLOCK) ON G.Panel_id = T.Panel_id AND G.Group_ = T.Group_
LEFT JOIN Code C WITH (NOLOCK) ON C.Code = T.Code AND C.CodeGroup = T.CodeGroup
LEFT JOIN TypeCode TC WITH (NOLOCK) ON TC.idTCode = C.idTCode
LEFT JOIN Users U WITH (NOLOCK) ON U.Panel_id = T.Panel_id AND U.Group_ = T.Group_ AND U.UserCode = T.Zone
LEFT JOIN Zones Z WITH (NOLOCK) ON Z.Panel_id = T.Panel_id AND Z.Group_ = T.Group_ AND Z.Zone = T.Zone
LEFT JOIN GroupResponse GR WITH (NOLOCK) ON T.Panel_id = CAST(GR.Group_id AS VARCHAR(15))
LEFT JOIN Company Co WITH (NOLOCK) ON Co.ID = G.CompanyID
LEFT JOIN dbo.Zones_GroupResponse ZGR WITH (NOLOCK) ON
	GR.Group_id = ZGR.Group_id
	AND (T.Zone = ZGR.Zone AND T.Panel_id = CAST(ZGR.Group_id AS VARCHAR(15)))
INNER JOIN vwRealPanel P WITH (NOLOCK) ON P.Panel_id = T.Panel_id
WHERE
	1 = 1
	AND (COALESCE(P.Pult_Id, 0) IN (0,1,2) OR P.Pult_Id IS NULL)
ORDER BY TC.Priority, T.Event_Parent_id, T.Event_id
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
