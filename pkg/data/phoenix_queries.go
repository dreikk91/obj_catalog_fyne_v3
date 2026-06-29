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
	COALESCE(PrimaryPhone.phone_number, C.Telephones) AS telephones,
	CAST(NULL AS nvarchar(255)) AS type_name,
	PanelMeta.device_name AS device_name,
	P.Disabled AS panel_disabled,
	P.TestPanel AS test_panel,
	P.Panel_type AS panel_type,
	T.StateEvent AS state_event,
	P.CreateDate AS create_date,
	P.DateLastChange AS date_last_change,
	PanelMeta.has_zstatus_device AS has_zstatus_device,
	PanelMeta.is_prohibited AS is_prohibited,
	PanelEngineer.has_engineer AS has_engineer,
	CAST(NULL AS nvarchar(255)) AS engineer_name,
	CAST(NULL AS nvarchar(max)) AS company_memo,
	CAST(NULL AS nvarchar(max)) AS additional_technical_information,
	PrimarySIM.sim_number AS sim1_number,
	SecondarySIM.sim_number AS sim2_number,
	P.Latitude AS latitude,
	P.Longtitude AS longitude
FROM Groups G WITH (NOLOCK)
LEFT JOIN Company C WITH (NOLOCK) ON C.ID = G.CompanyID
LEFT JOIN (
	SELECT Panel_id, Group_, MAX(StateEvent) AS StateEvent
	FROM Temp WITH (NOLOCK)
	GROUP BY Panel_id, Group_
) T ON T.Panel_id = G.Panel_id AND T.Group_ = G.Group_
INNER JOIN vwRealPanel P WITH (NOLOCK) ON P.Panel_id = G.Panel_id
OUTER APPLY (
	SELECT TOP (1)
		MRT.Message AS device_name,
		CASE WHEN ISNULL(M.IsZStatusDevice, 0) = 1 AND ISNULL(M.NotTested, 0) = 0 THEN 1 ELSE 0 END AS has_zstatus_device,
		CASE
			WHEN COALESCE(CASE
				WHEN MRT.DeviceId IN (79,107,120,128,132,134,137,138,149,151,166,177,179,191,197,199) THEN G.IsProhibited
				ELSE M.IsProhibited
			END, 0) = 1 THEN 1
			ELSE 0
		END AS is_prohibited
	FROM Mphone M WITH (NOLOCK)
	LEFT JOIN MphoneRadioType MRT WITH (NOLOCK) ON MRT.RadioType_id = M.RadioType
	WHERE M.Panel_id = G.Panel_id
	ORDER BY
		CASE WHEN ISNULL(M.IsZStatusDevice, 0) = 1 AND ISNULL(M.NotTested, 0) = 0 THEN 0 ELSE 1 END,
		M.Mphone_id
) AS PanelMeta
OUTER APPLY (
	SELECT TOP (1) 1 AS has_engineer
	FROM Engineers E WITH (NOLOCK)
	WHERE E.Work_panel_id = G.Panel_id
) AS PanelEngineer
OUTER APPLY (
	SELECT TOP (1)
		LTRIM(RTRIM(ISNULL(RT.PhoneNo, ''))) AS phone_number
	FROM Responsibles R WITH (NOLOCK)
	INNER JOIN ResponsibleTel RT WITH (NOLOCK) ON RT.ResponsiblesList_id = R.ResponsiblesList_id
	LEFT JOIN ResponsibleTelDescription RTD WITH (NOLOCK) ON
		RTD.Responsible_id = R.Responsible_id
		AND RTD.ResponsibleTel_id = RT.ResponsibleTel_id
	WHERE
		R.panel_id = G.Panel_id
		AND LTRIM(RTRIM(ISNULL(RT.PhoneNo, ''))) <> ''
	ORDER BY
		ISNULL(RTD.CallOrder, R.Responsible_Number),
		R.Responsible_Number,
		RT.ResponsibleTel_id
) AS PrimaryPhone
OUTER APPLY (
	SELECT TOP (1)
		COALESCE(NULLIF(LTRIM(RTRIM(S.SimNumber)), ''), NULLIF(LTRIM(RTRIM(S.RealSimNumber)), '')) AS sim_number
	FROM Mphone M WITH (NOLOCK)
	INNER JOIN OnBoardDevice D WITH (NOLOCK) ON D.Mphone_id = M.Mphone_id
	INNER JOIN OnBoardDeviceType DT WITH (NOLOCK) ON DT.OnBoardDeviceType_ID = D.OnBoardDeviceType_ID
	INNER JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = D.OnBoardDevice_ID
	WHERE
		M.Panel_id = G.Panel_id
		AND DT.Enum = 0
		AND COALESCE(NULLIF(LTRIM(RTRIM(S.SimNumber)), ''), NULLIF(LTRIM(RTRIM(S.RealSimNumber)), '')) IS NOT NULL
	ORDER BY
		CASE WHEN ISNULL(S.IsCurrentSim, 0) = 1 THEN 0 ELSE 1 END,
		CASE WHEN ISNULL(S.IsMainSim, 0) = 1 THEN 0 ELSE 1 END,
		S.Sim_ID
) AS PrimarySIM
OUTER APPLY (
	SELECT TOP (1)
		COALESCE(NULLIF(LTRIM(RTRIM(S.SimNumber)), ''), NULLIF(LTRIM(RTRIM(S.RealSimNumber)), '')) AS sim_number
	FROM Mphone M WITH (NOLOCK)
	INNER JOIN OnBoardDevice D WITH (NOLOCK) ON D.Mphone_id = M.Mphone_id
	INNER JOIN OnBoardDeviceType DT WITH (NOLOCK) ON DT.OnBoardDeviceType_ID = D.OnBoardDeviceType_ID
	INNER JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = D.OnBoardDevice_ID
	WHERE
		M.Panel_id = G.Panel_id
		AND DT.Enum = 1
		AND COALESCE(NULLIF(LTRIM(RTRIM(S.SimNumber)), ''), NULLIF(LTRIM(RTRIM(S.RealSimNumber)), '')) IS NOT NULL
	ORDER BY
		CASE WHEN ISNULL(S.IsCurrentSim, 0) = 1 THEN 0 ELSE 1 END,
		CASE WHEN ISNULL(S.IsMainSim, 0) = 1 THEN 0 ELSE 1 END,
		S.Sim_ID
) AS SecondarySIM
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
	COALESCE(PrimaryPhone.phone_number, C.Telephones) AS telephones,
	CT.Name AS type_name,
	PanelMeta.device_name AS device_name,
	P.Disabled AS panel_disabled,
	P.TestPanel AS test_panel,
	P.Panel_type AS panel_type,
	T.StateEvent AS state_event,
	P.CreateDate AS create_date,
	P.DateLastChange AS date_last_change,
	PanelMeta.has_zstatus_device AS has_zstatus_device,
	PanelMeta.is_prohibited AS is_prohibited,
	CASE WHEN E.engineer_name IS NULL THEN 0 ELSE 1 END AS has_engineer,
	E.engineer_name AS engineer_name,
	C.Memo AS company_memo,
	P.AdditionalTechnicalInformation AS additional_technical_information
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
OUTER APPLY (
	SELECT TOP (1)
		MRT.Message AS device_name,
		CASE WHEN ISNULL(M.IsZStatusDevice, 0) = 1 AND ISNULL(M.NotTested, 0) = 0 THEN 1 ELSE 0 END AS has_zstatus_device,
		CASE
			WHEN COALESCE(CASE
				WHEN MRT.DeviceId IN (79,107,120,128,132,134,137,138,149,151,166,177,179,191,197,199) THEN G.IsProhibited
				ELSE M.IsProhibited
			END, 0) = 1 THEN 1
			ELSE 0
		END AS is_prohibited
	FROM Mphone M WITH (NOLOCK)
	LEFT JOIN MphoneRadioType MRT WITH (NOLOCK) ON MRT.RadioType_id = M.RadioType
	WHERE M.Panel_id = G.Panel_id
	ORDER BY
		CASE WHEN ISNULL(M.IsZStatusDevice, 0) = 1 AND ISNULL(M.NotTested, 0) = 0 THEN 0 ELSE 1 END,
		M.Mphone_id
) AS PanelMeta
OUTER APPLY (
	SELECT TOP (1)
		LTRIM(RTRIM(ISNULL(RT.PhoneNo, ''))) AS phone_number
	FROM Responsibles R WITH (NOLOCK)
	INNER JOIN ResponsibleTel RT WITH (NOLOCK) ON RT.ResponsiblesList_id = R.ResponsiblesList_id
	LEFT JOIN ResponsibleTelDescription RTD WITH (NOLOCK) ON
		RTD.Responsible_id = R.Responsible_id
		AND RTD.ResponsibleTel_id = RT.ResponsibleTel_id
	WHERE
		R.panel_id = G.Panel_id
		AND LTRIM(RTRIM(ISNULL(RT.PhoneNo, ''))) <> ''
	ORDER BY
		ISNULL(RTD.CallOrder, R.Responsible_Number),
		R.Responsible_Number,
		RT.ResponsibleTel_id
) AS PrimaryPhone
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
		COALESCE(NULLIF(LTRIM(RTRIM(S.SimNumber)), ''), NULLIF(LTRIM(RTRIM(S.RealSimNumber)), '')) AS sim_number,
		OP.OperatorName AS operator_name
	FROM OnBoardDevice D WITH (NOLOCK)
	INNER JOIN OnBoardDeviceType DT WITH (NOLOCK) ON DT.OnBoardDeviceType_ID = D.OnBoardDeviceType_ID
	INNER JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = D.OnBoardDevice_ID
	LEFT JOIN Operators OP WITH (NOLOCK) ON OP.Operator_ID = S.Operator_ID
	WHERE
		D.MPhone_ID = MDC.Mphone_id
		AND DT.Enum = 0
		AND COALESCE(NULLIF(LTRIM(RTRIM(S.SimNumber)), ''), NULLIF(LTRIM(RTRIM(S.RealSimNumber)), '')) IS NOT NULL
	ORDER BY
		CASE WHEN ISNULL(S.IsCurrentSim, 0) = 1 THEN 0 ELSE 1 END,
		CASE WHEN ISNULL(S.IsMainSim, 0) = 1 THEN 0 ELSE 1 END,
		S.Sim_ID
) AS PrimarySIM
OUTER APPLY (
	SELECT TOP (1)
		COALESCE(NULLIF(LTRIM(RTRIM(S.SimNumber)), ''), NULLIF(LTRIM(RTRIM(S.RealSimNumber)), '')) AS sim_number,
		OP.OperatorName AS operator_name
	FROM OnBoardDevice D WITH (NOLOCK)
	INNER JOIN OnBoardDeviceType DT WITH (NOLOCK) ON DT.OnBoardDeviceType_ID = D.OnBoardDeviceType_ID
	INNER JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = D.OnBoardDevice_ID
	LEFT JOIN Operators OP WITH (NOLOCK) ON OP.Operator_ID = S.Operator_ID
	WHERE
		D.MPhone_ID = MDC.Mphone_id
		AND DT.Enum = 1
		AND COALESCE(NULLIF(LTRIM(RTRIM(S.SimNumber)), ''), NULLIF(LTRIM(RTRIM(S.RealSimNumber)), '')) IS NOT NULL
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

const phoenixObjectSIMListQuery = `
WITH SimBase AS (
	SELECT
		M.Panel_id AS panel_id,
		DT.Enum AS sim_slot,
		COALESCE(NULLIF(LTRIM(RTRIM(S.SimNumber)), ''), NULLIF(LTRIM(RTRIM(S.RealSimNumber)), '')) AS sim_number,
		ROW_NUMBER() OVER (
			PARTITION BY M.Panel_id, DT.Enum
			ORDER BY
				CASE WHEN ISNULL(S.IsCurrentSim, 0) = 1 THEN 0 ELSE 1 END,
				CASE WHEN ISNULL(S.IsMainSim, 0) = 1 THEN 0 ELSE 1 END,
				S.Sim_ID
		) AS rn
	FROM Mphone M WITH (NOLOCK)
	INNER JOIN OnBoardDevice D WITH (NOLOCK) ON D.Mphone_id = M.Mphone_id
	INNER JOIN OnBoardDeviceType DT WITH (NOLOCK) ON DT.OnBoardDeviceType_ID = D.OnBoardDeviceType_ID
	INNER JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = D.OnBoardDevice_ID
	WHERE COALESCE(NULLIF(LTRIM(RTRIM(S.SimNumber)), ''), NULLIF(LTRIM(RTRIM(S.RealSimNumber)), '')) IS NOT NULL
)
SELECT
	panel_id,
	MAX(CASE WHEN sim_slot = 0 AND rn = 1 THEN sim_number END) AS sim1_number,
	MAX(CASE WHEN sim_slot = 1 AND rn = 1 THEN sim_number END) AS sim2_number
FROM SimBase
GROUP BY panel_id
`

const phoenixOfflinePanelsQuery = `
SELECT DISTINCT panel_id
FROM (
	SELECT
		MDC.Panel_id AS panel_id
	FROM vwMPhoneDeviceChannels MDC WITH (NOLOCK)
	JOIN (
		SELECT
			Panel_id,
			MIN(Group_) AS Group_,
			SUM(CONVERT(int, ISNULL(IsOpen, 0))) AS IsOpen
		FROM Groups G WITH (NOLOCK)
		GROUP BY Panel_id
	) G ON G.Panel_id = MDC.Panel_id
	JOIN MPhone M WITH (NOLOCK) ON M.Mphone_id = MDC.mphone_id
	JOIN vwRealPanel RP WITH (NOLOCK) ON RP.Panel_id = MDC.Panel_id
	LEFT JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = MDC.OnBoardDevice_ID
	LEFT JOIN (
		SELECT DISTINCT
			Mphone_id,
			OnBoardDevice_ID,
			OnBoardDeviceTypeEnum,
			ChannelType_id AS ActiveChannel,
			TestTimeout AS TestTimeoutActiveChannel,
			TestTimeoutReserved AS TestTimeoutReservedActiveChannel
		FROM vwMPhoneDeviceChannels WITH (NOLOCK)
		WHERE IsActive = 1
	) AC ON AC.OnBoardDevice_ID = MDC.OnBoardDevice_ID
	WHERE
		ISNULL(MDC.IsZStatus, 0) = 0
		AND RP.Disabled = 0
		AND (RP.Movable_Object = 0 OR G.IsOpen = 0)
		AND MDC.LastTest IS NOT NULL
		AND MDC.TestTimeout IS NOT NULL
		AND M.NotTested = 0
		AND (
			(MDC.OnBoardDeviceTypeEnum IN (0,1) AND S.IsCurrentSim = 1 AND MDC.IsActive = 1 AND MDC.LastTest + CONVERT(varchar, MDC.TestTimeout, 108) + '0:2:0' < GETDATE())
			OR (MDC.OnBoardDeviceTypeEnum IN (0,1) AND S.IsCurrentSim = 0 AND MDC.IsActive = 1 AND MDC.LastTest + CONVERT(varchar, COALESCE(MDC.TestTimeoutReserved, MDC.TestTimeout), 108) + '0:2:0' < GETDATE())
			OR (MDC.OnBoardDeviceTypeEnum IN (2,3,4) AND MDC.LastTest + CONVERT(varchar, MDC.TestTimeout, 108) + '0:2:0' < GETDATE())
			OR (MDC.OnBoardDeviceTypeEnum IN (0,1) AND S.IsCurrentSim = 1 AND MDC.IsActive = 0 AND MDC.ChannelType_id IN (1, 5) AND AC.ActiveChannel IN (2, 6) AND MDC.LastTest + CONVERT(varchar, AC.TestTimeoutActiveChannel, 108) + '0:2:0' + '0:3:0' < GETDATE())
			OR (MDC.OnBoardDeviceTypeEnum IN (0,1) AND S.IsCurrentSim = 1 AND MDC.IsActive = 0 AND MDC.ChannelType_id IN (2, 6) AND AC.ActiveChannel IN (1, 5) AND MDC.LastTest + CONVERT(varchar, MDC.TestTimeout, 108) + '0:2:0' < GETDATE())
			OR (MDC.OnBoardDeviceTypeEnum IN (0,1) AND S.IsCurrentSim = 0 AND MDC.IsActive = 0 AND MDC.ChannelType_id IN (1, 5) AND AC.ActiveChannel IN (2, 6) AND MDC.LastTest + CONVERT(varchar, COALESCE(AC.TestTimeoutReservedActiveChannel, AC.TestTimeoutActiveChannel), 108) + '0:2:0' + '0:3:0' < GETDATE())
			OR (MDC.OnBoardDeviceTypeEnum IN (0,1) AND S.IsCurrentSim = 0 AND MDC.IsActive = 0 AND MDC.ChannelType_id IN (2, 6) AND AC.ActiveChannel IN (1, 5) AND MDC.LastTest + CONVERT(varchar, COALESCE(MDC.TestTimeoutReserved, MDC.TestTimeout), 108) + '0:2:0' < GETDATE())
			OR (MDC.OnBoardDeviceTypeEnum IN (0,1) AND S.IsCurrentSim = 0 AND MDC.IsActive = 0 AND AC.ActiveChannel IS NULL AND MDC.LastTest + CONVERT(varchar, COALESCE(MDC.TestTimeoutReserved, MDC.TestTimeout), 108) + '0:2:0' + '0:3:0' < GETDATE())
		)
	UNION
	SELECT
		MDC.Panel_id AS panel_id
	FROM vwMPhoneDeviceChannels MDC WITH (NOLOCK)
	JOIN (
		SELECT
			Panel_id,
			MIN(Group_) AS Group_,
			SUM(CONVERT(int, ISNULL(IsOpen, 0))) AS IsOpen
		FROM Groups G WITH (NOLOCK)
		GROUP BY Panel_id
	) G ON G.Panel_id = MDC.Panel_id
	JOIN MPhone M WITH (NOLOCK) ON M.Mphone_id = MDC.mphone_id
	JOIN vwRealPanel RP WITH (NOLOCK) ON RP.Panel_id = MDC.Panel_id
	LEFT JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = MDC.OnBoardDevice_ID
	LEFT JOIN (
		SELECT DISTINCT
			BD.Mphone_id,
			MIN(C.TestTimeout) AS TestTimeoutCurrentSimChannel,
			MAX(C.LastTest) AS LastTestCurrentSimChannel
		FROM OnBoardDevice BD WITH (NOLOCK)
		JOIN Channel C WITH (NOLOCK) ON C.OnBoardDevice_ID = BD.OnBoardDevice_ID
		JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = BD.OnBoardDevice_ID
		WHERE S.IsCurrentSim = 1
			AND C.IsActive = 1
		GROUP BY BD.MPhone_ID
	) CS ON CS.MPhone_ID = MDC.Mphone_id
	LEFT JOIN (
		SELECT DISTINCT
			BD.MPhone_ID,
			MIN(CONVERT(int, C.IsZStatus)) AS IsZStatusCurrentSim
		FROM OnBoardDevice BD WITH (NOLOCK)
		JOIN Channel C WITH (NOLOCK) ON C.OnBoardDevice_ID = BD.OnBoardDevice_ID
		JOIN Sim S WITH (NOLOCK) ON S.OnBoardDevice_ID = BD.OnBoardDevice_ID
		WHERE S.IsCurrentSim = 1
		GROUP BY BD.MPhone_ID
	) ZC ON ZC.MPhone_ID = MDC.MPhone_ID
	WHERE
		ISNULL(MDC.IsZStatus, 0) = 0
		AND RP.Disabled = 0
		AND (RP.Movable_Object = 0 OR G.IsOpen = 0)
		AND MDC.LastTest IS NOT NULL
		AND MDC.TestTimeout IS NOT NULL
		AND M.NotTested = 0
		AND M.UseAlternativeTesting = 1
		AND MDC.OnBoardDeviceTypeEnum IN (0,1)
		AND S.IsCurrentSim = 0
		AND ISNULL(ZC.IsZStatusCurrentSim, 0) = 1
		AND ISNULL(CS.LastTestCurrentSimChannel, 0) + CONVERT(varchar, CS.TestTimeoutCurrentSimChannel, 108) + '0:2:0' + '0:3:0' + '0:1:0' < GETDATE()
	UNION
	SELECT
		M.Panel_id AS panel_id
	FROM MPhone M WITH (NOLOCK)
	JOIN (
		SELECT
			Panel_id,
			MIN(Group_) AS Group_,
			SUM(CONVERT(int, ISNULL(IsOpen, 0))) AS IsOpen
		FROM Groups G WITH (NOLOCK)
		GROUP BY Panel_id
	) G ON G.Panel_id = M.Panel_id
	JOIN vwRealPanel RP WITH (NOLOCK) ON RP.Panel_id = M.Panel_id
	JOIN MphoneRadioType MRT WITH (NOLOCK) ON MRT.RadioType_id = M.RadioType
	WHERE
		ISNULL(M.IsZStatusDevice, 0) = 0
		AND NOT EXISTS (
			SELECT TOP (1) Channel_ID
			FROM vwMPhoneDeviceChannels WITH (NOLOCK)
			WHERE IsZStatus = 0
				AND Mphone_id = M.mphone_id
		)
		AND M.NotTested = 0
) offline_panels
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
	CASE WHEN COALESCE(C.AutoReset, 0) = 1 THEN 1 ELSE 0 END AS auto_reset,
	CASE WHEN COALESCE(C.groupsent, 0) = 1 THEN 1 ELSE 0 END AS group_sent,
	ISNULL(C.AccessCode, '0') AS access_code,
	C.ContactID_Code AS contact_id_code,
	CASE WHEN COALESCE(C.System, 0) = 1 THEN 1 ELSE 0 END AS system_flag,
	C.zoneno AS code_zone_no,
	G.Message AS group_name,
	Z.Message AS zone_name,
	T.Line AS line,
	T.Event_Parent_id AS event_parent_id,
	T.StateEvent AS state_event,
	T.Computer AS computer,
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
	TC.Message AS type_code_message,
	CASE WHEN COALESCE(C.AutoReset, 0) = 1 THEN 1 ELSE 0 END AS auto_reset,
	CASE WHEN COALESCE(C.groupsent, 0) = 1 THEN 1 ELSE 0 END AS group_sent,
	ISNULL(C.AccessCode, '0') AS access_code,
	C.ContactID_Code AS contact_id_code,
	CASE WHEN COALESCE(C.System, 0) = 1 THEN 1 ELSE 0 END AS system_flag,
	C.zoneno AS code_zone_no,
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

const phoenixRecentEventsQuery = `
SELECT TOP (500)
	A.Event_id AS event_id,
	A.Panel_id AS panel_id,
	A.Group_ AS group_no,
	A.Zone AS zone_no,
	A.TimeEvent AS time_event,
	A.Code AS event_code,
	C.Message AS code_message,
	TC.idTCode AS type_code_id,
	TC.Message AS type_code_message,
	CASE WHEN COALESCE(C.AutoReset, 0) = 1 THEN 1 ELSE 0 END AS auto_reset,
	CASE WHEN COALESCE(C.groupsent, 0) = 1 THEN 1 ELSE 0 END AS group_sent,
	ISNULL(C.AccessCode, '0') AS access_code,
	C.ContactID_Code AS contact_id_code,
	CASE WHEN COALESCE(C.System, 0) = 1 THEN 1 ELSE 0 END AS system_flag,
	C.zoneno AS code_zone_no,
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

const phoenixAvailableStatesQuery = `
SELECT A.AvailableState_id AS available_state_id, S.StateName AS state_name
FROM AvailableStates A WITH (NOLOCK)
INNER JOIN States S WITH (NOLOCK) ON S.State_id = A.AvailableState_id
WHERE A.State_id = @p1 AND (A.idTCode = 1 OR A.idTCode IS NULL)
ORDER BY A.AvailableState_id
`

const phoenixResponseGroupsQuery = `
SELECT
	gr.Group_id AS group_id,
	gr.Description AS description,
	gr.callsign AS callsign,
	gr.Status_id AS status_id,
	sgr.reason AS status_text,
	gr.Panel_id AS panel_id,
	COALESCE(NULLIF(LTRIM(RTRIM(mp.LastLatitude)), ''), gr.DislocationPointLat) AS latitude,
	COALESCE(NULLIF(LTRIM(RTRIM(mp.LastLongtitude)), ''), gr.DislocationPointLon) AS longitude,
	gr.TimeArriveToObject AS time_arrive_to_object
FROM GroupResponse gr WITH (NOLOCK)
LEFT JOIN StatusGroupResponse sgr WITH (NOLOCK) ON sgr.status_id = gr.Status_id
LEFT JOIN MPhone mp WITH (NOLOCK) ON mp.Mphone_id = gr.Mphone_id
WHERE COALESCE(gr.Disabled, 0) = 0
ORDER BY gr.Group_id
`

const phoenixObjectPreferredResponseGroupQuery = `
SELECT TOP (1)
	grg.Group_id AS group_id,
	gr.Description AS description
FROM GroupResponse_Group grg WITH (NOLOCK)
INNER JOIN GroupResponse gr WITH (NOLOCK) ON gr.Group_id = grg.Group_id
WHERE
	grg.Panel_id = @p1
	AND grg.Group_ = 0
ORDER BY
	CASE WHEN COALESCE(grg.MainGroup, 0) = 0 THEN 0 ELSE 1 END DESC,
	grg.Group_id
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
	TC.Message AS type_code_message,
	CASE WHEN COALESCE(C.AutoReset, 0) = 1 THEN 1 ELSE 0 END AS auto_reset,
	CASE WHEN COALESCE(C.groupsent, 0) = 1 THEN 1 ELSE 0 END AS group_sent,
	ISNULL(C.AccessCode, '0') AS access_code,
	C.ContactID_Code AS contact_id_code,
	CASE WHEN COALESCE(C.System, 0) = 1 THEN 1 ELSE 0 END AS system_flag,
	C.zoneno AS code_zone_no,
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

const phoenixObjectEventsRangeQuery = `
SELECT TOP (500)
	A.Event_id AS event_id,
	A.Panel_id AS panel_id,
	A.Group_ AS group_no,
	A.Zone AS zone_no,
	A.TimeEvent AS time_event,
	A.Code AS event_code,
	C.Message AS code_message,
	TC.idTCode AS type_code_id,
	TC.Message AS type_code_message,
	CASE WHEN COALESCE(C.AutoReset, 0) = 1 THEN 1 ELSE 0 END AS auto_reset,
	CASE WHEN COALESCE(C.groupsent, 0) = 1 THEN 1 ELSE 0 END AS group_sent,
	ISNULL(C.AccessCode, '0') AS access_code,
	C.ContactID_Code AS contact_id_code,
	CASE WHEN COALESCE(C.System, 0) = 1 THEN 1 ELSE 0 END AS system_flag,
	C.zoneno AS code_zone_no,
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
  AND A.TimeEvent >= @p2
  AND A.TimeEvent <= @p3
ORDER BY A.Event_id DESC
`
