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

				sensorType := strings.TrimSpace(line.Type.String())
				if sensorType == "" {
					sensorType = "Шлейф"
				}

				zones = append(zones, models.Zone{
					Number:     number,
					Name:       name,
					SensorType: sensorType,
					Status:     models.ZoneNormal,
				})
			}
			sort.SliceStable(zones, func(i, j int) bool { return zones[i].Number < zones[j].Number })
			return zones
		}
	}

	if len(record.Rooms) == 0 {
		return []models.Zone{{Number: 1, Name: "Об'єкт", SensorType: "Приміщення", Status: models.ZoneNormal}}
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

		zones = append(zones, models.Zone{Number: number, Name: name, SensorType: sensorType, Status: models.ZoneNormal})
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

	orderedIDs := normalizeContactIDs(record.InCharge, record.ManagerID)
	if len(orderedIDs) == 0 {
		return nil
	}

	users, usersErr := p.loadUsers(ctx)
	if usersErr != nil {
		log.Debug().Err(usersErr).Msg("CASL: не вдалося завантажити read_user, повертаю fallback контакти")
	}

	contacts := make([]models.Contact, 0, len(orderedIDs))
	for idx, userID := range orderedIDs {
		user, hasUser := users[userID]
		if !hasUser {
			contacts = append(contacts, models.Contact{Name: "Користувач #" + userID, Position: "IN_CHARGE", Priority: idx + 1})
			continue
		}

		contacts = append(contacts, models.Contact{
			Name:     user.FullName(),
			Position: strings.TrimSpace(user.Role),
			Phone:    user.PrimaryPhone(),
			Priority: idx + 1,
			CodeWord: strings.TrimSpace(user.Tag.String()),
		})
	}

	return contacts
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
		strings.TrimSpace(user.Role) != "" ||
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

	return append([]caslUser(nil), resp.Data...), nil
}
