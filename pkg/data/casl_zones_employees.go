package data

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/models"

	"github.com/rs/zerolog/log"
)

func (p *CASLCloudProvider) GetZones(objectID string) []models.Zone {
	internalID, ok := parseObjectID(objectID)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return nil
	}

	groups := p.resolveCASLObjectGroups(ctx, record)
	defaultGroup := firstCASLObjectGroup(groups)

	if _, devicesErr := p.loadDevices(ctx); devicesErr == nil {
		device, hasDevice := p.resolveDeviceForObject(record)
		if hasDevice && len(device.Lines) > 0 {
			zones := make([]models.Zone, 0, len(device.Lines))
			for idx, line := range device.Lines {
				number := int(line.Number.Int64())
				if number <= 0 {
					number = int(line.ID.Int64())
				}
				if number <= 0 {
					number = idx + 1
				}

				name := strings.TrimSpace(line.Name.String())
				if name == "" {
					name = fmt.Sprintf("Зона %d", number)
				}

				sensorType := p.resolveCASLDeviceLineTypeLabel(ctx, line)
				if sensorType == "" {
					sensorType = "Шлейф"
				}

				group := resolveCASLGroupForLine(line, groups, defaultGroup)

				zones = append(zones, models.Zone{
					Number:         number,
					Name:           name,
					SensorType:     sensorType,
					Status:         models.ZoneNormal,
					GroupID:        group.ID,
					GroupNumber:    group.Number,
					GroupName:      displayCASLGroupName(group),
					GroupStateText: group.StateText,
				})
			}
			sort.SliceStable(zones, func(i, j int) bool { return zones[i].Number < zones[j].Number })
			return zones
		}
	}

	if len(record.Rooms) == 0 {
		return []models.Zone{{
			Number:         1,
			Name:           "Об'єкт",
			SensorType:     "Приміщення",
			Status:         models.ZoneNormal,
			GroupID:        defaultGroup.ID,
			GroupNumber:    defaultGroup.Number,
			GroupName:      displayCASLGroupName(defaultGroup),
			GroupStateText: defaultGroup.StateText,
		}}
	}

	zones := make([]models.Zone, 0, len(record.Rooms))
	for idx, room := range record.Rooms {
		number := idx + 1
		if parsed := parseCASLID(room.RoomID); parsed > 0 {
			number = parsed
		}

		name := strings.TrimSpace(room.Name)
		if name == "" {
			name = fmt.Sprintf("Приміщення %d", idx+1)
		}

		sensorType := "Приміщення"
		if strings.TrimSpace(room.Description) != "" {
			sensorType = strings.TrimSpace(room.Description)
		}

		group := resolveCASLGroupForRoom(room, groups, idx, defaultGroup)
		zones = append(zones, models.Zone{
			Number:         number,
			Name:           name,
			SensorType:     sensorType,
			Status:         models.ZoneNormal,
			GroupID:        group.ID,
			GroupNumber:    group.Number,
			GroupName:      displayCASLGroupName(group),
			GroupStateText: group.StateText,
		})
	}

	sort.SliceStable(zones, func(i, j int) bool { return zones[i].Number < zones[j].Number })
	return zones
}

func (p *CASLCloudProvider) GetEmployees(objectID string) []models.Contact {
	internalID, ok := parseObjectID(objectID)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return nil
	}

	groups := p.resolveCASLObjectGroups(ctx, record)
	defaultGroup := firstCASLObjectGroup(groups)

	orderedIDs := normalizeContactIDs(record.InCharge, record.ManagerID)
	if len(orderedIDs) == 0 && len(record.Rooms) == 0 {
		return nil
	}

	users, usersErr := p.loadUsers(ctx)
	if usersErr != nil {
		log.Debug().Err(usersErr).Msg("CASL: не вдалося завантажити read_user, повертаю fallback контакти")
	}

	contacts := make([]models.Contact, 0, len(orderedIDs))
	if len(record.Rooms) > 0 {
		priority := 1
		seenUserIDs := make(map[string]struct{}, len(orderedIDs))
		for idx, room := range record.Rooms {
			group := resolveCASLGroupForRoom(room, groups, idx, defaultGroup)
			groupName := displayCASLGroupName(group)
			for _, roomUser := range room.Users {
				userID := strings.TrimSpace(roomUser.UserID)
				if userID != "" {
					seenUserIDs[userID] = struct{}{}
				}
				detailedUser := roomUser
				if userID != "" {
					if user, ok := users[userID]; ok {
						detailedUser = mergeCASLUsers(detailedUser, user)
					}
				}
				contact := buildCASLContact(detailedUser, priority)
				contact.GroupID = group.ID
				contact.GroupNumber = group.Number
				contact.GroupName = groupName
				contact.GroupStateText = group.StateText
				contacts = append(contacts, contact)
				priority++
			}
		}
		if len(contacts) > 0 {
			for _, userID := range orderedIDs {
				if _, ok := seenUserIDs[userID]; ok {
					continue
				}
				user, hasUser := users[userID]
				if !hasUser {
					contacts = append(contacts, models.Contact{
						Name:           "Користувач #" + userID,
						Position:       "IN_CHARGE",
						Priority:       priority,
						GroupID:        defaultGroup.ID,
						GroupNumber:    defaultGroup.Number,
						GroupName:      displayCASLGroupName(defaultGroup),
						GroupStateText: defaultGroup.StateText,
					})
					priority++
					continue
				}

				contact := buildCASLContact(user, priority)
				contact.GroupID = defaultGroup.ID
				contact.GroupNumber = defaultGroup.Number
				contact.GroupName = displayCASLGroupName(defaultGroup)
				contact.GroupStateText = defaultGroup.StateText
				contacts = append(contacts, contact)
				priority++
			}
			return contacts
		}
	}

	for idx, userID := range orderedIDs {
		user, hasUser := users[userID]
		if !hasUser {
			contacts = append(contacts, models.Contact{
				Name:           "Користувач #" + userID,
				Position:       "IN_CHARGE",
				Priority:       idx + 1,
				GroupID:        defaultGroup.ID,
				GroupNumber:    defaultGroup.Number,
				GroupName:      displayCASLGroupName(defaultGroup),
				GroupStateText: defaultGroup.StateText,
			})
			continue
		}

		contact := buildCASLContact(user, idx+1)
		contact.GroupID = defaultGroup.ID
		contact.GroupNumber = defaultGroup.Number
		contact.GroupName = displayCASLGroupName(defaultGroup)
		contact.GroupStateText = defaultGroup.StateText
		contacts = append(contacts, contact)
	}

	return contacts
}

func buildCASLContact(user caslUser, priority int) models.Contact {
	return models.Contact{
		Name:     user.FullName(),
		Position: strings.TrimSpace(user.Role),
		Phone:    user.PrimaryPhone(),
		Priority: priority,
		CodeWord: strings.TrimSpace(user.Tag.String()),
	}
}

func (p *CASLCloudProvider) resolveCASLObjectGroups(ctx context.Context, record caslGrdObject) []models.ObjectGroup {
	if state, err := p.readDeviceState(ctx, record); err == nil {
		if groups := p.buildCASLObjectGroups(ctx, record, state.Groups); len(groups) > 0 {
			return groups
		}
	}

	if len(record.Rooms) == 0 {
		return nil
	}

	groups := make([]models.ObjectGroup, 0, len(record.Rooms))
	for idx, room := range record.Rooms {
		number := idx + 1
		if parsed := parseCASLID(room.RoomID); parsed > 0 {
			number = parsed
		}
		name := strings.TrimSpace(room.Name)
		if name == "" {
			name = fmt.Sprintf("Група %d", number)
		}
		groups = append(groups, models.ObjectGroup{
			ID:          fmt.Sprintf("casl:group=%d", number),
			Source:      "casl",
			Number:      number,
			Name:        name,
			StateText:   "—",
			RoomID:      strings.TrimSpace(room.RoomID),
			RoomName:    name,
			PremiseID:   strings.TrimSpace(room.RoomID),
			PremiseName: name,
		})
	}

	return groups
}

func (p *CASLCloudProvider) buildCASLObjectGroups(
	ctx context.Context,
	record caslGrdObject,
	rawGroups any,
) []models.ObjectGroup {
	groups := mapCASLDeviceGroupsToObjectGroups(rawGroups, record.Rooms)

	if _, devicesErr := p.loadDevices(ctx); devicesErr == nil {
		if device, hasDevice := p.resolveDeviceForObject(record); hasDevice && len(device.Lines) > 0 {
			groups = alignCASLGroupsWithDeviceLines(groups, device.Lines, record.Rooms)
		}
	}

	if len(groups) == 0 {
		stats, statsErr := p.loadCASLGroupStatistics(ctx)
		if statsErr == nil {
			groups = mergeCASLGroupsWithStatistics(groups, stats[strings.TrimSpace(record.ObjID)])
		}
	}

	return groups
}

func (p *CASLCloudProvider) loadCASLGroupStatistics(ctx context.Context) (map[string]map[int]int, error) {
	p.mu.RLock()
	cacheValid := len(p.cachedGroupStats) > 0 && time.Since(p.cachedGroupStatsAt) < caslObjectsStatTTL
	if cacheValid {
		copied := cloneCASLGroupStatistics(p.cachedGroupStats)
		p.mu.RUnlock()
		return copied, nil
	}
	p.mu.RUnlock()

	raw, err := p.GetObjectsStatistic(ctx)
	if err != nil {
		return nil, err
	}

	stats := normalizeCASLGroupStatistics(raw)

	p.mu.Lock()
	p.cachedGroupStats = cloneCASLGroupStatistics(stats)
	p.cachedGroupStatsAt = time.Now()
	p.mu.Unlock()

	return cloneCASLGroupStatistics(stats), nil
}

func normalizeCASLGroupStatistics(raw map[string]any) map[string]map[int]int {
	if len(raw) == 0 {
		return map[string]map[int]int{}
	}

	source, ok := raw["groupStatistics"]
	if !ok {
		source = raw
	}

	payload, ok := source.(map[string]any)
	if !ok {
		return map[string]map[int]int{}
	}

	result := make(map[string]map[int]int, len(payload))
	for rawObjectID, rawGroups := range payload {
		objectID := strings.TrimSpace(rawObjectID)
		if objectID == "" {
			continue
		}

		groupMap, ok := rawGroups.(map[string]any)
		if !ok {
			continue
		}

		normalizedGroups := make(map[int]int, len(groupMap))
		for rawNumber, rawState := range groupMap {
			number := parseCASLID(rawNumber)
			if number <= 0 {
				continue
			}
			normalizedGroups[number] = parseCASLAnyInt(rawState)
		}
		if len(normalizedGroups) == 0 {
			continue
		}

		result[objectID] = normalizedGroups
	}

	return result
}

func cloneCASLGroupStatistics(src map[string]map[int]int) map[string]map[int]int {
	if len(src) == 0 {
		return map[string]map[int]int{}
	}

	dst := make(map[string]map[int]int, len(src))
	for objectID, groups := range src {
		clonedGroups := make(map[int]int, len(groups))
		for number, state := range groups {
			clonedGroups[number] = state
		}
		dst[objectID] = clonedGroups
	}
	return dst
}

func mergeCASLGroupsWithStatistics(groups []models.ObjectGroup, stats map[int]int) []models.ObjectGroup {
	if len(stats) == 0 {
		return groups
	}

	mergedByNumber := make(map[int]models.ObjectGroup, len(groups)+len(stats))
	order := make([]int, 0, len(groups)+len(stats))

	for _, group := range groups {
		if group.Number <= 0 {
			continue
		}
		mergedByNumber[group.Number] = group
		order = append(order, group.Number)
	}

	numbers := make([]int, 0, len(stats))
	for number := range stats {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)

	for _, number := range numbers {
		group, exists := mergedByNumber[number]
		if !exists {
			group = models.ObjectGroup{
				ID:     fmt.Sprintf("casl:group=%d", number),
				Source: "casl",
				Number: number,
			}
			order = append(order, number)
		}

		applyCASLGroupStatisticState(&group, stats[number])
		mergedByNumber[number] = group
	}

	result := make([]models.ObjectGroup, 0, len(mergedByNumber))
	seen := make(map[int]struct{}, len(order))
	for _, number := range order {
		if _, ok := seen[number]; ok {
			continue
		}
		seen[number] = struct{}{}
		result = append(result, mergedByNumber[number])
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Number == result[j].Number {
			return result[i].RoomName < result[j].RoomName
		}
		return result[i].Number < result[j].Number
	})

	return result
}

func applyCASLGroupStatisticState(group *models.ObjectGroup, state int) {
	if group == nil {
		return
	}

	switch state {
	case 0:
		group.Armed = false
		group.StateText = "ЗНЯТО"
	case 1:
		group.Armed = true
		group.StateText = "ПІД ОХОРОНОЮ"
	default:
		if state > 0 {
			group.Armed = true
		}
		if strings.TrimSpace(group.StateText) == "" || strings.TrimSpace(group.StateText) == "—" {
			if group.Armed {
				group.StateText = "ПІД ОХОРОНОЮ"
			} else {
				group.StateText = "ЗНЯТО"
			}
		}
	}
}

func firstCASLObjectGroup(groups []models.ObjectGroup) models.ObjectGroup {
	if len(groups) > 0 {
		return groups[0]
	}
	return models.ObjectGroup{
		ID:        "casl:group=1",
		Source:    "casl",
		Number:    1,
		Name:      "Основна група",
		StateText: "—",
	}
}

func resolveCASLGroupForLine(line caslDeviceLine, groups []models.ObjectGroup, fallback models.ObjectGroup) models.ObjectGroup {
	if number := caslLineGroupNumber(line); number > 0 {
		if group, ok := matchCASLGroupByNumber(groups, number); ok {
			return group
		}
	}
	if group, ok := matchCASLGroupByRoomID(groups, line.RoomID.String()); ok {
		return group
	}
	return fallback
}

func caslLineGroupNumber(line caslDeviceLine) int {
	if number := int(line.GroupNumber.Int64()); number > 0 {
		return number
	}
	if number := parseCASLID(line.GroupID.String()); number > 0 {
		return number
	}
	if number := parseCASLID(line.Group.String()); number > 0 {
		return number
	}
	return 0
}

func resolveCASLGroupForRoom(
	room caslRoom,
	groups []models.ObjectGroup,
	idx int,
	fallback models.ObjectGroup,
) models.ObjectGroup {
	if group, ok := matchCASLGroupByRoomID(groups, room.RoomID); ok {
		return group
	}
	if idx >= 0 && idx < len(groups) {
		return groups[idx]
	}
	return fallback
}

func matchCASLGroupByRoomID(groups []models.ObjectGroup, roomID string) (models.ObjectGroup, bool) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return models.ObjectGroup{}, false
	}
	for _, group := range groups {
		if strings.TrimSpace(group.RoomID) == roomID || strings.TrimSpace(group.PremiseID) == roomID {
			return group, true
		}
	}
	return models.ObjectGroup{}, false
}

func matchCASLGroupByNumber(groups []models.ObjectGroup, number int) (models.ObjectGroup, bool) {
	if number <= 0 {
		return models.ObjectGroup{}, false
	}
	for _, group := range groups {
		if group.Number == number {
			return group, true
		}
	}
	return models.ObjectGroup{}, false
}

func displayCASLGroupName(group models.ObjectGroup) string {
	if name := strings.TrimSpace(group.Name); name != "" {
		return name
	}
	if name := strings.TrimSpace(group.RoomName); name != "" {
		return name
	}
	if name := strings.TrimSpace(group.PremiseName); name != "" {
		return name
	}
	if group.Number > 0 {
		return fmt.Sprintf("Група %d", group.Number)
	}
	return "Група"
}

func (p *CASLCloudProvider) loadUsers(ctx context.Context) (map[string]caslUser, error) {
	p.mu.RLock()
	cacheValid := len(p.cachedUsers) > 0 && time.Since(p.cachedUsersAt) < caslUsersCacheTTL
	if cacheValid {
		copied := make(map[string]caslUser, len(p.cachedUsers))
		for key, value := range p.cachedUsers {
			copied[key] = value
		}
		p.mu.RUnlock()
		return copied, nil
	}
	p.mu.RUnlock()

	objects, objectsErr := p.loadObjects(ctx)
	if objectsErr == nil && len(objects) > 0 {
		usersFromObjects := collectCASLUsersFromObjects(objects)
		if len(usersFromObjects) > 0 && hasDetailedCASLUsers(usersFromObjects) {
			p.applyCASLCoreSnapshot(nil, nil, usersFromObjects)
			copied := make(map[string]caslUser, len(usersFromObjects))
			for key, value := range usersFromObjects {
				copied[key] = value
			}
			return copied, nil
		}
	}

	users, err := p.readUsers(ctx)
	if err == nil && len(users) > 0 {
		index := make(map[string]caslUser, len(users))
		for _, user := range users {
			appendCASLUserIndex(index, user)
		}
		p.applyCASLCoreSnapshot(nil, nil, index)
		copied := make(map[string]caslUser, len(index))
		for key, value := range index {
			copied[key] = value
		}
		return copied, nil
	}

	connObjects, connDevices, connUsers, connErr := p.readConnectionsCoreSnapshot(ctx)
	if connErr == nil && len(connUsers) > 0 {
		p.applyCASLCoreSnapshot(connObjects, connDevices, connUsers)
		copied := make(map[string]caslUser, len(connUsers))
		for key, value := range connUsers {
			copied[key] = value
		}
		return copied, nil
	}

	if err != nil {
		return nil, err
	}
	return nil, connErr
}

func collectCASLUsersFromObjects(records []caslGrdObject) map[string]caslUser {
	index := make(map[string]caslUser, len(records)*2)
	for _, record := range records {
		appendCASLUserIndex(index, record.Manager)
		for _, room := range record.Rooms {
			for _, roomUser := range room.Users {
				appendCASLUserIndex(index, roomUser)
			}
		}
	}
	return index
}

func hasDetailedCASLUsers(users map[string]caslUser) bool {
	for _, user := range users {
		if hasCASLUserDetails(user) {
			return true
		}
	}
	return false
}

func hasCASLUserDetails(user caslUser) bool {
	if strings.TrimSpace(user.LastName) != "" ||
		strings.TrimSpace(user.FirstName) != "" ||
		strings.TrimSpace(user.MiddleName) != "" ||
		strings.TrimSpace(user.Email) != "" ||
		strings.TrimSpace(user.Tag.String()) != "" {
		return true
	}
	for _, phone := range user.PhoneNumbers {
		if strings.TrimSpace(phone.Number) != "" {
			return true
		}
	}
	return false
}

func appendCASLUserIndex(index map[string]caslUser, user caslUser) {
	userID := strings.TrimSpace(user.UserID)
	if userID == "" {
		return
	}
	if existing, ok := index[userID]; ok {
		index[userID] = mergeCASLUsers(existing, user)
		return
	}
	index[userID] = user
}

func mergeCASLUsers(base caslUser, incoming caslUser) caslUser {
	if strings.TrimSpace(base.Email) == "" {
		base.Email = incoming.Email
	}
	if strings.TrimSpace(base.LastName) == "" {
		base.LastName = incoming.LastName
	}
	if strings.TrimSpace(base.FirstName) == "" {
		base.FirstName = incoming.FirstName
	}
	if strings.TrimSpace(base.MiddleName) == "" {
		base.MiddleName = incoming.MiddleName
	}
	if strings.TrimSpace(base.Role) == "" {
		base.Role = incoming.Role
	}
	if strings.TrimSpace(base.Tag.String()) == "" {
		base.Tag = incoming.Tag
	}
	if len(base.PhoneNumbers) == 0 && len(incoming.PhoneNumbers) > 0 {
		base.PhoneNumbers = append([]caslPhoneNumber(nil), incoming.PhoneNumbers...)
	}
	return base
}

func (p *CASLCloudProvider) readUsers(ctx context.Context) ([]caslUser, error) {
	payload := map[string]any{"type": "read_user", "skip": 0, "limit": caslReadLimit}

	var resp caslReadUserResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}
	if err := validateCASLUsers(resp.Data); err != nil {
		return nil, err
	}

	return append([]caslUser(nil), resp.Data...), nil
}
