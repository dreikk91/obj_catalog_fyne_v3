package data

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type phoenixObjectCameraRow struct {
	ID       int64  `db:"camera_id"`
	Name     string `db:"camera_name"`
	GroupNo  int    `db:"group_no"`
	ZoneNo   int    `db:"zone_no"`
	ZoneName string `db:"zone_name"`
	RTSP     string `db:"rtsp_link"`
	RTSPLow  string `db:"rtsp_link_low"`
}

const phoenixObjectCamerasQuery = `
SELECT
	c.Id AS camera_id,
	c.Name AS camera_name,
	zc.Group_ AS group_no,
	zc.Zone AS zone_no,
	COALESCE(z.Message, '') AS zone_name,
	c.RTSPLink AS rtsp_link,
	c.RTSPLinkLow AS rtsp_link_low
FROM Zones_IPCameras zc WITH (NOLOCK)
INNER JOIN IPCameras c WITH (NOLOCK) ON c.Id = zc.IPCamera_Id
LEFT JOIN Zones z WITH (NOLOCK)
	ON z.Panel_id = zc.Panel_Id AND z.Group_ = zc.Group_ AND z.Zone = zc.Zone
WHERE zc.Panel_Id = @p1
ORDER BY zc.Group_, zc.Zone, c.Name
`

func (p *PhoenixDataProvider) GetObjectMedia(ctx context.Context, objectID int) ([]contracts.ObjectMedia, error) {
	if p == nil || p.db == nil {
		return nil, fmt.Errorf("phoenix media: база не ініціалізована")
	}
	panelID, ok := p.resolvePanelID(strconv.Itoa(objectID))
	if !ok {
		return nil, fmt.Errorf("phoenix media: об'єкт %d не знайдено", objectID)
	}
	var rows []phoenixObjectCameraRow
	if err := p.db.SelectContext(ctx, &rows, phoenixObjectCamerasQuery, panelID); err != nil {
		return nil, fmt.Errorf("phoenix media %s: %w", panelID, err)
	}
	result := make([]contracts.ObjectMedia, 0, len(rows))
	for _, row := range rows {
		rtsp := strings.TrimSpace(row.RTSP)
		if rtsp == "" {
			rtsp = strings.TrimSpace(row.RTSPLow)
		}
		if rtsp == "" {
			continue
		}
		room := strings.TrimSpace(row.ZoneName)
		if room == "" {
			room = fmt.Sprintf("Група %d, зона %d", row.GroupNo, row.ZoneNo)
		}
		name := strings.TrimSpace(row.Name)
		if name == "" {
			name = "Камера"
		}
		result = append(result, contracts.ObjectMedia{
			ID:       "phoenix-camera:" + strconv.FormatInt(row.ID, 10),
			Kind:     contracts.ObjectMediaCamera,
			Title:    name,
			RoomName: room,
			URL:      rtsp,
		})
	}
	return result, nil
}

func (p *PhoenixDataProvider) FetchObjectMedia(context.Context, contracts.ObjectMedia) ([]byte, error) {
	return nil, fmt.Errorf("phoenix media: фотографії об'єкта не підтримуються")
}
