package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

func (b *Bridge) startProvisioner(ctx context.Context) {
	go b.runProvisionLoop(ctx)
}

func (b *Bridge) runProvisionLoop(ctx context.Context) {
	select {
	case <-time.After(10 * time.Second):
	case <-ctx.Done():
		return
	}
	b.provisionAll(ctx)

	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.provisionAll(ctx)
		}
	}
}

type provEntry struct {
	ppkNum    int
	name      string
	address   string
	isPhoenix bool
	objectID  string // Phoenix: DisplayNumber (e.g. "L00001")
	sim1      string
	sim2      string
	timeout   int // device offline timeout, seconds
}

type phoenixProvisionGroup struct {
	number int
	name   string
	zones  []models.Zone
}

func (b *Bridge) provisionAll(ctx context.Context) {
	var ppkNums []any
	var entries []provEntry

	deviceTimeoutSec := int(b.cfg.DeviceTimeout.Duration().Seconds())

	if b.bridge != nil {
		for _, obj := range b.bridge.GetObjects() {
			n, ok := ids.BridgePPKNum(obj.DisplayNumber)
			if !ok {
				continue
			}
			name, addr := normNameAddr(obj.Name, obj.Address, obj.DisplayNumber)
			ppkNums = append(ppkNums, n)
			entries = append(entries, provEntry{
				ppkNum:  n,
				name:    name,
				address: addr,
			})
		}
	}
	if b.phoenix != nil {
		for _, obj := range b.phoenix.GetObjects() {
			n, ok := ids.PhoenixPPKNum(obj.DisplayNumber)
			if !ok {
				continue
			}
			name, addr := normNameAddr(obj.Name, obj.Address, obj.DisplayNumber)
			ppkNums = append(ppkNums, n)
			entries = append(entries, provEntry{
				ppkNum:    n,
				name:      name,
				address:   addr,
				isPhoenix: true,
				objectID:  obj.DisplayNumber,
				sim1:      obj.SIM1,
				sim2:      obj.SIM2,
				timeout:   deviceTimeoutSec,
			})
		}
	}

	if len(ppkNums) == 0 {
		return
	}

	pultID, err := b.readPultID(ctx)
	if err != nil {
		log.Error().Err(err).Msg("provisioner: read_pult")
		return
	}
	log.Info().Int("pult_id", pultID).Int("total", len(entries)).Msg("provisioner: syncing objects")

	linked, err := b.casl.GetGrdObjectLastActByDevice(ctx, pultID, ppkNums)
	if err != nil {
		log.Error().Err(err).Msg("provisioner: get_grd_object_last_act_by_device")
		return
	}

	for _, e := range entries {
		if info, ok := linked[e.ppkNum]; ok {
			if err := b.syncLinkedObject(ctx, pultID, e, info); err != nil {
				log.Error().Err(err).Int("ppk_num", e.ppkNum).Str("obj_id", string(info.ObjectID)).Msg("provisioner: update linked object failed")
			}
			if err := b.syncLinkedDevice(ctx, pultID, e); err != nil {
				log.Error().Err(err).Int("ppk_num", e.ppkNum).Msg("provisioner: update linked device failed")
			}

			// Read device info to get its CASL device_id
			dev, err := b.casl.ReadOneDevice(ctx, pultID, e.ppkNum)
			if err != nil {
				log.Error().Err(err).Int("ppk_num", e.ppkNum).Msg("provisioner: failed to read device for room/line sync")
				continue
			}

			if err := b.syncRoomsAndLines(ctx, pultID, e, string(dev.DeviceID), string(info.ObjectID)); err != nil {
				log.Error().Err(err).Int("ppk_num", e.ppkNum).Str("obj_id", string(info.ObjectID)).Msg("provisioner: sync rooms/lines failed")
			}
			continue
		}

		if err := b.provisionObject(ctx, pultID, e); err != nil {
			log.Error().Err(err).Int("ppk_num", e.ppkNum).Str("name", e.name).Msg("provisioner: failed")
		} else {
			log.Info().Int("ppk_num", e.ppkNum).Str("name", e.name).Msg("provisioner: object created and synced")
		}
	}
}

func (b *Bridge) syncLinkedObject(ctx context.Context, pultID int, e provEntry, info *LinkedObjectItem) error {
	if b.cfg.DisableDeviceSync || info.ObjectID == "" {
		return nil
	}
	pultStr := strconv.Itoa(pultID)
	payload := map[string]any{
		"type":     "update_grd_object",
		"obj_id":   string(info.ObjectID),
		"_user_id": "0",
		"_pult_id": pultStr,
	}
	changed := false
	if e.name != "" && e.name != string(info.ObjectName) {
		payload["name"] = e.name
		payload["description"] = e.name
		changed = true
	}
	if e.address != "" && e.address != string(info.Address) {
		payload["address"] = e.address
		changed = true
	}
	if !changed {
		return nil
	}

	r, err := b.apiRequest(ctx, payload)
	if err != nil {
		return fmt.Errorf("update_grd_object: %w", err)
	}
	if s, _ := r["status"].(string); s != "ok" {
		return fmt.Errorf("update_grd_object: %s", caslErrorCode(r))
	}
	log.Info().Int("ppk_num", e.ppkNum).Str("obj_id", string(info.ObjectID)).Msg("provisioner: updated linked grd_object info")
	return nil
}

func (b *Bridge) syncLinkedDevice(ctx context.Context, pultID int, e provEntry) error {
	if b.cfg.DisableDeviceSync {
		return nil
	}
	dev, err := b.casl.ReadOneDevice(ctx, pultID, e.ppkNum)
	if err != nil {
		return fmt.Errorf("read_one_device: %w", err)
	}

	updates := make(map[string]any)
	changed := false
	name := strings.TrimSpace(e.name)
	if name != "" && name != string(dev.Name) {
		updates["name"] = name
		changed = true
	}
	if e.timeout > 0 && e.timeout != int(dev.Timeout) {
		updates["timeout"] = e.timeout
		changed = true
	}
	sim1 := strings.TrimSpace(e.sim1)
	if sim1 != "" && sim1 != string(dev.Sim1) {
		updates["sim1"] = sim1
		changed = true
	}
	sim2 := strings.TrimSpace(e.sim2)
	if sim2 != "" && sim2 != string(dev.Sim2) {
		updates["sim2"] = sim2
		changed = true
	}

	if !changed {
		return nil
	}

	log.Info().Int("ppk_num", e.ppkNum).Interface("updates", updates).Msg("provisioner: updating linked device")
	if err := b.casl.UpdateDevice(ctx, pultID, string(dev.DeviceID), updates); err != nil {
		return fmt.Errorf("update_device: %w", err)
	}
	return nil
}

func (b *Bridge) readPultID(ctx context.Context) (int, error) {
	if b.cfg.PultID > 0 {
		return b.cfg.PultID, nil
	}

	resp, err := b.apiRequest(ctx, map[string]any{
		"type":     "read_pult",
		"skip":     0,
		"limit":    1,
		"_user_id": "0",
		"_pult_id": "0",
	})
	if err != nil {
		return 0, err
	}
	if s, _ := resp["status"].(string); s != "ok" {
		return 0, fmt.Errorf("status=%v error=%v", resp["status"], resp["error"])
	}
	data, _ := resp["data"].([]any)
	if len(data) == 0 {
		return 0, fmt.Errorf("no pults in CASL")
	}
	pult, _ := data[0].(map[string]any)
	pultIDStr, _ := pult["pult_id"].(string)
	pultID, err := strconv.Atoi(pultIDStr)
	if err != nil || pultID <= 0 {
		return 0, fmt.Errorf("invalid pult_id: %q", pultIDStr)
	}
	return pultID, nil
}

func (b *Bridge) provisionObject(ctx context.Context, pultID int, e provEntry) error {
	deviceID, err := b.ensureDevice(ctx, pultID, e.ppkNum, e.name, e.sim1, e.sim2, e.timeout)
	if err != nil {
		return fmt.Errorf("ensure_device: %w", err)
	}

	source := "bridge"
	if e.isPhoenix {
		source = "phoenix"
	}
	objID, err := b.casl.CreateGrdObject(ctx, pultID, e.ppkNum, e.name, e.address, source)
	if err != nil {
		return fmt.Errorf("create_grd_object: %w", err)
	}

	if err := b.syncRoomsAndLines(ctx, pultID, e, deviceID, objID); err != nil {
		return fmt.Errorf("sync_rooms_and_lines: %w", err)
	}
	return nil
}

// ensureDevice creates the device if it doesn't exist, or returns the existing device_id.
func (b *Bridge) ensureDevice(ctx context.Context, pultID int, ppkNum int, name, sim1, sim2 string, timeout int) (string, error) {
	deviceID, err := b.casl.CreateDevice(ctx, pultID, ppkNum, name, sim1, sim2, timeout)
	if err == nil {
		log.Info().Int("ppk_num", ppkNum).Str("device_id", deviceID).Msg("provisioner: created device")
		return deviceID, nil
	}

	if !strings.Contains(err.Error(), "NUMBER_IN_USE") {
		return "", err
	}

	// device already exists — look it up
	dev, err := b.casl.ReadOneDevice(ctx, pultID, ppkNum)
	if err != nil {
		return "", fmt.Errorf("read_one_device: %w", err)
	}

	deviceID = string(dev.DeviceID)
	if !b.cfg.DisableDeviceSync {
		updates := make(map[string]any)
		changed := false
		name = strings.TrimSpace(name)
		if name != "" && name != string(dev.Name) {
			updates["name"] = name
			changed = true
		}
		if timeout > 0 && timeout != int(dev.Timeout) {
			updates["timeout"] = timeout
			changed = true
		}
		sim1 = strings.TrimSpace(sim1)
		if sim1 != "" && sim1 != string(dev.Sim1) {
			updates["sim1"] = sim1
			changed = true
		}
		sim2 = strings.TrimSpace(sim2)
		if sim2 != "" && sim2 != string(dev.Sim2) {
			updates["sim2"] = sim2
			changed = true
		}

		if changed {
			log.Info().Int("ppk_num", ppkNum).Interface("updates", updates).Msg("provisioner: syncing existing device")
			if err := b.casl.UpdateDevice(ctx, pultID, deviceID, updates); err != nil {
				log.Error().Err(err).Int("ppk_num", ppkNum).Msg("provisioner: update existing device failed")
			}
		}
	}
	return deviceID, nil
}

func (b *Bridge) syncRoomsAndLines(ctx context.Context, pultID int, e provEntry, deviceID string, objID string) error {
	// 1. Get zones to sync (either from Phoenix or virtual for Bridge)
	var zones []models.Zone
	if e.isPhoenix {
		if b.phoenix != nil {
			zones = b.phoenix.GetZones(e.objectID)
		}
	} else {
		zones = []models.Zone{
			{
				Number:      1,
				GroupNumber: 1,
				GroupName:   "Головна",
				Name:        "Основна лінія",
			},
		}
	}

	// Group zones by group number
	groupByNum := make(map[int]*phoenixProvisionGroup)
	var groupOrder []int
	for _, z := range zones {
		gn := z.GroupNumber
		if gn <= 0 {
			gn = 1
		}
		if _, exists := groupByNum[gn]; !exists {
			gName := z.GroupName
			if gName == "" {
				gName = fmt.Sprintf("Група %d", gn)
			}
			groupByNum[gn] = &phoenixProvisionGroup{number: gn, name: gName}
			groupOrder = append(groupOrder, gn)
		}
		groupByNum[gn].zones = append(groupByNum[gn].zones, z)
	}
	if len(groupByNum) == 0 {
		groupByNum[1] = &phoenixProvisionGroup{number: 1, name: "Головна"}
		groupOrder = []int{1}
	}
	sort.Ints(groupOrder)

	// 2. Read existing rooms from CASL
	existingRooms, err := b.casl.ReadGrdRoom(ctx, pultID, objID)
	if err != nil {
		return fmt.Errorf("read existing rooms: %w", err)
	}

	roomIDByGroup := make(map[int]string)
	for _, gn := range groupOrder {
		g := groupByNum[gn]
		var matchedRoom *CASLRoom
		for _, r := range existingRooms {
			desc := r.GetDescription()
			// Match by description prefix or name
			if strings.HasPrefix(desc, fmt.Sprintf("Phoenix група %d", gn)) ||
				strings.HasPrefix(string(r.Name), fmt.Sprintf("Група %d", gn)) ||
				(gn == 1 && string(r.Name) == "Головна") {
				matchedRoom = r
				break
			}
		}

		var roomID string
		targetDesc := phoenixRoomDescription(g)
		if !e.isPhoenix {
			targetDesc = "Автоматично створено casl-bridge"
		}

		if matchedRoom != nil {
			roomID = string(matchedRoom.RoomID)
			// Check if name or description needs update
			if string(matchedRoom.Name) != g.name || matchedRoom.GetDescription() != targetDesc {
				log.Info().Int("ppk_num", e.ppkNum).Str("room_id", roomID).Str("old_name", string(matchedRoom.Name)).Str("new_name", g.name).Msg("provisioner: updating room name/description")
				if err := b.casl.UpdateGrdRoom(ctx, pultID, roomID, g.name, targetDesc); err != nil {
					log.Error().Err(err).Int("ppk_num", e.ppkNum).Str("room_id", roomID).Msg("provisioner: failed to update room")
				}
			} else {
				log.Debug().Int("ppk_num", e.ppkNum).Str("room_id", roomID).Msg("provisioner: room unchanged, skipping update")
			}
		} else {
			log.Info().Int("ppk_num", e.ppkNum).Str("room_name", g.name).Msg("provisioner: creating room")
			newRoomID, err := b.casl.CreateGrdRoom(ctx, pultID, objID, g.name, targetDesc)
			if err != nil {
				return fmt.Errorf("create room %s: %w", g.name, err)
			}
			roomID = newRoomID
		}
		roomIDByGroup[gn] = roomID
	}

	// 3. Read device to check current lines
	dev, err := b.casl.ReadOneDevice(ctx, pultID, e.ppkNum)
	if err != nil {
		return fmt.Errorf("read device lines: %w", err)
	}

	// 4. Ensure lines exist and are linked to their respective rooms
	for _, z := range zones {
		lineNum := z.Number
		if lineNum <= 0 {
			continue
		}
		gn := z.GroupNumber
		if gn <= 0 {
			gn = 1
		}
		roomID, ok := roomIDByGroup[gn]
		if !ok || roomID == "" {
			continue
		}

		targetDesc := phoenixLineDescription(z)
		if !e.isPhoenix {
			targetDesc = "Основна лінія"
		}

		var existingLine *CASLDeviceLine
		if dev.Lines != nil {
			existingLine = dev.Lines[strconv.Itoa(lineNum)]
		}

		if existingLine != nil {
			// Compare line description, group number, and room ID
			lineChanged := false
			updates := make(map[string]any)

			if strings.TrimSpace(string(existingLine.Description)) != strings.TrimSpace(targetDesc) {
				updates["description"] = strings.TrimSpace(targetDesc)
				lineChanged = true
			}
			if int(existingLine.GroupNumber) != gn {
				updates["group_number"] = gn
				lineChanged = true
			}

			if lineChanged {
				log.Info().Int("ppk_num", e.ppkNum).Int("line_num", lineNum).Interface("updates", updates).Msg("provisioner: updating device line")
				err := b.casl.UpdateDeviceLine(ctx, pultID, string(dev.DeviceID), lineNum, string(existingLine.LineID), updates)
				if err != nil {
					log.Error().Err(err).Int("ppk_num", e.ppkNum).Int("line_num", lineNum).Msg("provisioner: failed to update device line")
				}
			} else {
				log.Debug().Int("ppk_num", e.ppkNum).Int("line_num", lineNum).Msg("provisioner: device line unchanged, skipping update")
			}

			// Ensure room link exists
			if string(existingLine.RoomID) != roomID {
				log.Info().Int("ppk_num", e.ppkNum).Int("line_num", lineNum).Str("old_room", string(existingLine.RoomID)).Str("new_room", roomID).Msg("provisioner: linking device line to room")
				err := b.casl.AddLineToRoom(ctx, pultID, string(dev.DeviceID), lineNum, roomID, objID)
				if err != nil {
					if ignoreErr := b.ignoreLinkedLineError(ctx, pultID, e.ppkNum, err); ignoreErr != nil {
						log.Error().Err(ignoreErr).Int("ppk_num", e.ppkNum).Int("line_num", lineNum).Msg("provisioner: failed to link line to room")
					}
				}
			} else {
				log.Debug().Int("ppk_num", e.ppkNum).Int("line_num", lineNum).Msg("provisioner: room link unchanged, skipping link")
			}
		} else {
			// Create new line
			log.Info().Int("ppk_num", e.ppkNum).Int("line_num", lineNum).Msg("provisioner: creating device line")
			err := b.casl.EnsureDeviceLine(ctx, pultID, string(dev.DeviceID), lineNum, gn, targetDesc)
			if err != nil {
				log.Error().Err(err).Int("ppk_num", e.ppkNum).Int("line_num", lineNum).Msg("provisioner: failed to ensure device line")
				continue
			}

			// Link to room
			log.Info().Int("ppk_num", e.ppkNum).Int("line_num", lineNum).Str("room_id", roomID).Msg("provisioner: linking new device line to room")
			err = b.casl.AddLineToRoom(ctx, pultID, string(dev.DeviceID), lineNum, roomID, objID)
			if err != nil {
				if ignoreErr := b.ignoreLinkedLineError(ctx, pultID, e.ppkNum, err); ignoreErr != nil {
					log.Error().Err(ignoreErr).Int("ppk_num", e.ppkNum).Int("line_num", lineNum).Msg("provisioner: failed to link new line to room")
				}
			}
		}
	}

	return nil
}

func (b *Bridge) ignoreLinkedLineError(ctx context.Context, pultID, ppkNum int, err error) error {
	apiErr, ok := err.(caslAPIError)
	if !ok || apiErr.code != "LINE_NUMBER_IN_USE" {
		return err
	}
	linked, lookupErr := b.casl.GetGrdObjectLastActByDevice(ctx, pultID, []any{ppkNum})
	if lookupErr != nil {
		return fmt.Errorf("%w; linked lookup: %v", err, lookupErr)
	}
	if _, ok := linked[ppkNum]; ok {
		return nil
	}
	return err
}

func phoenixRoomDescription(g *phoenixProvisionGroup) string {
	if g == nil {
		return ""
	}
	if len(g.zones) == 0 {
		return fmt.Sprintf("Phoenix група %d", g.number)
	}
	return fmt.Sprintf("Phoenix група %d, зон: %d", g.number, len(g.zones))
}

func phoenixLineDescription(z models.Zone) string {
	parts := []string{}
	if z.Name != "" {
		parts = append(parts, z.Name)
	}
	if z.SensorType != "" {
		parts = append(parts, z.SensorType)
	}
	if z.IsBypassed {
		parts = append(parts, "відключено")
	}
	return strings.Join(parts, " / ")
}

func normNameAddr(name, address, fallback string) (string, string) {
	if name == "" {
		name = fallback
	}
	if address == "" {
		address = "-"
	}
	return name, address
}
