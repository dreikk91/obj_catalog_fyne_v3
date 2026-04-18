package casleditor

import (
	"context"
	"encoding/base64"
	"fmt"
	"image/color"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/nyaruka/phonenumbers"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		t := strings.TrimSpace(v)
		if t != "" {
			return t
		}
	}
	return ""
}

func LabelsFromTexts(texts []string) []fyne.CanvasObject {
	objs := make([]fyne.CanvasObject, 0, len(texts))
	for _, t := range texts {
		objs = append(objs, widget.NewLabel(t))
	}
	return objs
}

func PhoneNumbersText(phones []contracts.CASLPhoneNumber) string {
	values := make([]string, 0, len(phones))
	for _, phone := range phones {
		number := strings.TrimSpace(phone.Number)
		if number != "" {
			values = append(values, number)
		}
	}
	return strings.Join(values, " ")
}

func FixedMinHeightArea(height float32, content fyne.CanvasObject) fyne.CanvasObject {
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(1, height))
	return container.NewStack(spacer, content)
}

func createGhostButton(label string, positive bool) *widget.Button {
	btn := widget.NewButton(label, func() {})
	if positive {
		btn.Importance = widget.SuccessImportance
	} else {
		btn.Importance = widget.LowImportance
	}
	btn.Disable()
	return btn
}

type caslImageTapTarget struct {
	widget.BaseWidget
	content fyne.CanvasObject
	onTap   func()
}

func newCASLImageTapTarget(content fyne.CanvasObject, onTap func()) *caslImageTapTarget {
	target := &caslImageTapTarget{
		content: content,
		onTap:   onTap,
	}
	target.ExtendBaseWidget(target)
	return target
}

func (t *caslImageTapTarget) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *caslImageTapTarget) TappedSecondary(*fyne.PointEvent) {}

func (t *caslImageTapTarget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}

func SetCASLImageStrip(box *fyne.Container, images []string, emptyText string, ownerLabel string, provider contracts.CASLObjectEditorProvider, onDelete func(string), onPreview func(string, string)) {
	if box == nil {
		return
	}
	items := make([]fyne.CanvasObject, 0, max(len(images), 1))
	for idx, raw := range images {
		if tile := newCASLImageTile(raw, idx+1, ownerLabel, provider, onDelete, onPreview); tile != nil {
			items = append(items, tile)
		}
	}
	if len(items) == 0 {
		items = append(items, widget.NewCard("", "", container.NewCenter(widget.NewLabel(emptyText))))
	}
	box.Objects = items
	box.Refresh()
}

func SetCASLImageSlots(box *fyne.Container, images []string, ownerLabel string, provider contracts.CASLObjectEditorProvider, onDelete func(string), onPreview func(string, string), onAdd func()) {
	if box == nil {
		return
	}
	items := make([]fyne.CanvasObject, 0, 3)
	for idx := 0; idx < 3; idx++ {
		if idx < len(images) {
			if tile := newCASLImageTile(images[idx], idx+1, ownerLabel, provider, onDelete, onPreview); tile != nil {
				items = append(items, tile)
				continue
			}
		}
		items = append(items, newCASLEmptyImageSlot(onAdd))
	}
	box.Objects = items
	box.Refresh()
}

func newCASLEmptyImageSlot(onAdd func()) fyne.CanvasObject {
	addBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		if onAdd != nil {
			onAdd()
		}
	})
	addBtn.Importance = widget.LowImportance
	label := widget.NewLabel("Додати фото")
	label.Alignment = fyne.TextAlignCenter
	return widget.NewCard("", "", container.NewCenter(container.NewVBox(
		addBtn,
		label,
	)))
}

func newCASLImageTile(raw string, index int, ownerLabel string, provider contracts.CASLObjectEditorProvider, onDelete func(string), onPreview func(string, string)) fyne.CanvasObject {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "null") {
		return nil
	}
	imageID := caslImageID(raw)
	deleteKey := imageID
	if deleteKey == "" {
		deleteKey = raw
	}
	var deleteRow fyne.CanvasObject
	if onDelete != nil && deleteKey != "" {
		deleteRow = container.NewHBox(
			layout.NewSpacer(),
			widget.NewButton("Видалити", func() { onDelete(deleteKey) }),
		)
	}
	ownerLabel = FirstNonEmpty(ownerLabel, "Фото")
	previewTitle := fmt.Sprintf("%s, фото %d", ownerLabel, index)
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(190, 120))
	previewHolder := container.NewStack(spacer)
	resource, err := caslImageResource(raw, fmt.Sprintf("casl-image-%d", index))
	if err == nil {
		image := canvas.NewImageFromResource(resource)
		image.FillMode = canvas.ImageFillContain
		image.ScaleMode = canvas.ImageScaleFastest
		image.SetMinSize(fyne.NewSize(180, 110))
		previewHolder.Objects = []fyne.CanvasObject{
			newCASLImageTapTarget(container.NewPadded(image), func() {
				if onPreview != nil {
					onPreview(previewTitle, raw)
				}
			}),
		}
	} else if imageID != "" && provider != nil {
		status := widget.NewLabel("Завантаження прев'ю...")
		status.Wrapping = fyne.TextWrapWord
		previewHolder.Objects = []fyne.CanvasObject{
			newCASLImageTapTarget(container.NewCenter(status), func() {
				if onPreview != nil {
					onPreview(previewTitle, raw)
				}
			}),
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			body, fetchErr := provider.FetchCASLImagePreview(ctx, imageID)
			fyne.Do(func() {
				if fetchErr != nil {
					status.SetText("Не вдалося завантажити прев'ю\nID: " + imageID)
					return
				}
				resource, resourceErr := caslImageResourceFromBytes(body, fmt.Sprintf("casl-image-%d", index))
				if resourceErr != nil {
					status.SetText("Некоректне зображення\nID: " + imageID)
					return
				}
				image := canvas.NewImageFromResource(resource)
				image.FillMode = canvas.ImageFillContain
				image.ScaleMode = canvas.ImageScaleFastest
				image.SetMinSize(fyne.NewSize(180, 110))
				previewHolder.Objects = []fyne.CanvasObject{
					newCASLImageTapTarget(container.NewPadded(image), func() {
						if onPreview != nil {
							onPreview(previewTitle, raw)
						}
					}),
				}
				previewHolder.Refresh()
			})
		}()
	} else {
		label := widget.NewLabel(FirstNonEmpty(raw, "Немає фото"))
		label.Wrapping = fyne.TextWrapWord
		previewHolder.Objects = []fyne.CanvasObject{container.NewCenter(label)}
	}
	caption := widget.NewLabel(ownerLabel)
	caption.Alignment = fyne.TextAlignCenter
	caption.TextStyle = fyne.TextStyle{Bold: true}
	content := container.NewBorder(caption, deleteRow, nil, nil, previewHolder)
	return widget.NewCard("", "Клікніть для перегляду", content)
}

func caslImageResource(raw string, name string) (fyne.Resource, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty image")
	}
	if !strings.HasPrefix(strings.ToLower(raw), "data:") {
		return nil, fmt.Errorf("unsupported image reference")
	}

	commaIdx := strings.Index(raw, ",")
	if commaIdx <= 5 {
		return nil, fmt.Errorf("invalid data url")
	}

	header := raw[:commaIdx]
	payload := strings.TrimSpace(raw[commaIdx+1:])
	mimeType := "image/jpeg"
	if strings.HasPrefix(strings.ToLower(header), "data:") {
		rest := strings.TrimSpace(header[5:])
		for _, separator := range []string{";", " "} {
			if idx := strings.Index(rest, separator); idx > 0 {
				rest = rest[:idx]
				break
			}
		}
		if parsed := strings.TrimSpace(rest); parsed != "" {
			mimeType = parsed
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(payload)
		if err != nil {
			return nil, err
		}
	}

	exts, _ := mime.ExtensionsByType(mimeType)
	ext := ".img"
	if len(exts) > 0 && strings.TrimSpace(exts[0]) != "" {
		ext = exts[0]
	}
	return fyne.NewStaticResource(name+ext, decoded), nil
}

func caslImageResourceFromBytes(data []byte, name string) (fyne.Resource, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty image data")
	}
	mimeType := http.DetectContentType(data)
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	exts, _ := mime.ExtensionsByType(mimeType)
	ext := ".img"
	if len(exts) > 0 && strings.TrimSpace(exts[0]) != "" {
		ext = exts[0]
	}
	return fyne.NewStaticResource(name+ext, data), nil
}

func caslImageID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(strings.ToLower(raw), "data:") {
		return ""
	}
	return raw
}

func caslEncodeImageUpload(fileName string, data []byte) (string, string) {
	if len(data) == 0 {
		return "", ""
	}
	mimeType := strings.TrimSpace(mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName))))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	if mimeType == "" || !strings.HasPrefix(mimeType, "image/") {
		mimeType = "image/jpeg"
	}
	imageType := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(mimeType, "image/")))
	switch imageType {
	case "jpeg":
		imageType = "jpg"
	case "svg+xml":
		imageType = "svg"
	}
	return imageType, base64.StdEncoding.EncodeToString(data)
}

func minListIndex(value int, max int) int {
	if max < 0 {
		return -1
	}
	if value < 0 {
		return 0
	}
	if value > max {
		return max
	}
	return value
}

func mappedOptionValue(label string, mapping map[string]string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	if value, ok := mapping[label]; ok {
		return strings.TrimSpace(value)
	}
	return label
}

func optionLabelByValue(value string, mapping map[string]string) string {
	value = strings.TrimSpace(value)
	for label, mapped := range mapping {
		if strings.TrimSpace(mapped) == value {
			return label
		}
	}
	return value
}

func optionLabelByInt64(value int64, mapping map[string]int64) string {
	for label, mapped := range mapping {
		if mapped == value {
			return label
		}
	}
	return ""
}

func setRawOptionLabel(mapping map[string]string, raw string, label string) {
	if mapping == nil {
		return
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = caslLineTypeDisplayName(raw)
	}
	mapping[raw] = label
}

func ensureOptionMapping(mapping map[string]string, key string, fallbackName string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	if mapping == nil {
		return
	}
	for _, existing := range mapping {
		if strings.TrimSpace(existing) == key {
			return
		}
	}
	label := strings.TrimSpace(fallbackName)
	if label == "" {
		label = key
	}
	base := label
	suffix := 2
	for {
		if existing, exists := mapping[label]; !exists {
			mapping[label] = key
			return
		} else if strings.TrimSpace(existing) == key {
			return
		}
		label = fmt.Sprintf("%s [%d]", base, suffix)
		suffix++
	}
}

func caslDefaultLineTypes() map[string]string {
	return map[string]string{
		"EMPTY":              "Пустий шлейф",
		"NORMAL":             "Нормальна зона",
		"ZONE_ALARM_ON_KZ":   "Тривожний шлейф",
		"ZONE_ALARM":         "Тривожний шлейф",
		"ZONE_ALM":           "Тривожний шлейф",
		"ALM_BTN":            "Тривожна кнопка",
		"ZONE_FIRE":          "Пожежний шлейф",
		"ZONE_NORMAL":        "Нормальна зона",
		"ZONE_COMMON":        "Нормальна зона",
		"ZONE_DELAY":         "Вхідний шлейф",
		"ZONE_PANIC":         "Тривожна кнопка",
		"UNTYPED_ZONE_ALARM": "Нетипізована тривожна зона",
	}
}

func caslLineTypeDisplayName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "—"
	}
	if label, ok := caslDefaultLineTypes()[raw]; ok {
		return label
	}

	upper := strings.ToUpper(raw)
	switch {
	case upper == "EMPTY":
		return "Пустий шлейф"
	case strings.HasPrefix(strings.ToLower(raw), "fire_pipeline"), strings.Contains(upper, "FIRE"):
		return "Пожежний шлейф"
	case strings.Contains(upper, "PANIC"), strings.Contains(upper, "ALM_BTN"):
		return "Тривожна кнопка"
	case strings.Contains(upper, "ALARM"), strings.Contains(upper, "ZONE_ALM"):
		return "Тривожний шлейф"
	case strings.Contains(upper, "NORMAL"), strings.Contains(upper, "COMMON"):
		return "Нормальна зона"
	case strings.Contains(upper, "UNTYPED"):
		return "Звичайна зона"
	case strings.Contains(upper, "DELAY"), strings.Contains(upper, "ENTRY"):
		return "Вхідний шлейф"
	default:
		return humanizeCASLToken(raw)
	}
}

func caslLineTypeDisplayNameWithDict(dict map[string]any, raw string) string {
	if translated := caslDictionaryTranslateLabel(dict, raw); translated != "" {
		return translated
	}
	return caslLineTypeDisplayName(raw)
}

func humanizeCASLToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	replacer := strings.NewReplacer("_", " ", "-", " ")
	text := strings.TrimSpace(replacer.Replace(raw))
	if text == "" {
		return raw
	}
	return strings.ToUpper(text[:1]) + strings.ToLower(text[1:])
}

func humanizeCASLAdapterType(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	switch strings.ToUpper(raw) {
	case "SYS":
		return "SYS"
	default:
		return humanizeCASLToken(raw)
	}
}

func caslAdapterTypeDisplayNameWithDict(dict map[string]any, raw string) string {
	if translated := caslDictionaryTranslateLabel(dict, raw); translated != "" {
		return translated
	}
	return humanizeCASLAdapterType(raw)
}

var caslDeviceTypesDisplayNames = map[string]string{
	"TYPE_DEVICE_CASL":                    "CASL",
	"TYPE_DEVICE_DUNAY_8L":                "Дунай-8L",
	"TYPE_DEVICE_DUNAY_16L":               "Дунай-16L",
	"TYPE_DEVICE_DUNAY_4L":                "Дунай-4L",
	"TYPE_DEVICE_LUN":                     "Лунь",
	"TYPE_DEVICE_AJAX":                    "Ajax",
	"TYPE_DEVICE_AJAX_SIA":                "Ajax(SIA)",
	"TYPE_DEVICE_BRON_SIA":                "Bron(SIA)",
	"TYPE_DEVICE_CASL_PLUS":               "CASL+",
	"TYPE_DEVICE_DOZOR_4":                 "Дозор-4",
	"TYPE_DEVICE_DOZOR_8":                 "Дозор-8",
	"TYPE_DEVICE_DOZOR_8MG":               "Дозор-8MG",
	"TYPE_DEVICE_DUNAY_8_32":              "Дунай-8/32",
	"TYPE_DEVICE_DUNAY_16_32":             "Дунай-16/32",
	"TYPE_DEVICE_DUNAY_4_3":               "Дунай-4.3",
	"TYPE_DEVICE_DUNAY_4_3S":              "Дунай-4.3.1S",
	"TYPE_DEVICE_DUNAY_8(16)32_DUNAY_G1R": "128 + G1R",
	"TYPE_DEVICE_DUNAY_STK":               "Дунай-СТК",
	"TYPE_DEVICE_DUNAY_4.2":               "4.2 + G1R",
	"TYPE_DEVICE_VBD4":                    "ВБД4 + G1R",
	"TYPE_DEVICE_VBD6_2":                  "ВБД6-2 + G1R",
	"TYPE_DEVICE_DUNAY_PSPN":              "ПСПН (R.COM)",
	"TYPE_DEVICE_DUNAY_PSPN_ECOM":         "ПСПН (ECOM)",
	"TYPE_DEVICE_VBD4_ECOM":               "ВБД4",
	"TYPE_DEVICE_VBD_16":                  "ВБД6-16",
}

func caslDeviceTypeDisplayName(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if label, ok := caslDeviceTypesDisplayNames[strings.ToUpper(raw)]; ok {
		return label
	}
	trimmed := strings.TrimPrefix(raw, "TYPE_DEVICE_")
	if trimmed != raw {
		return humanizeCASLToken(trimmed)
	}
	return humanizeCASLToken(raw)
}

func caslDeviceTypeDisplayNameWithDict(dict map[string]any, raw string) string {
	if translated := caslDictionaryTranslateLabel(dict, raw); translated != "" {
		return translated
	}
	return caslDeviceTypeDisplayName(raw)
}

func labeledOptionMap(mapping map[string]string) ([]string, map[string]string) {
	if len(mapping) == 0 {
		return nil, map[string]string{}
	}
	options := make([]string, 0, len(mapping))
	result := make(map[string]string, len(mapping))
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		ensureOptionMapping(result, key, mapping[key])
	}
	for label := range result {
		options = append(options, label)
	}
	sort.Strings(options)
	return options, result
}

func caslLineTypeOptionsMap(dict map[string]any) map[string]string {
	lineTypes := caslDictionaryOptionsMap(dict, "line_types")
	if len(lineTypes) == 0 {
		lineTypes = caslDictionaryOptionsMap(dict, "zone_types")
	}
	if len(lineTypes) == 0 {
		lineTypes = caslDefaultLineTypes()
	}
	for rawType, label := range lineTypes {
		setRawOptionLabel(lineTypes, rawType, FirstNonEmpty(label, caslLineTypeDisplayNameWithDict(dict, rawType)))
	}
	return lineTypes
}

func caslAdapterTypeOptionsMap(dict map[string]any) map[string]string {
	adapterTypes := caslDictionaryOptionsMap(dict, "adapters", "adapter_types")
	if len(adapterTypes) == 0 {
		adapterTypes = map[string]string{}
	}
	for rawType, label := range adapterTypes {
		setRawOptionLabel(adapterTypes, rawType, FirstNonEmpty(label, caslAdapterTypeDisplayNameWithDict(dict, rawType)))
	}
	return adapterTypes
}

func caslDeviceTypeOptionsMap(dict map[string]any) map[string]string {
	deviceTypes := caslDictionaryOptionsMap(dict, "device_types", "user_device_types")
	if len(deviceTypes) == 0 {
		deviceTypes = map[string]string{}
	}
	if raw, ok := dict["devices"]; ok {
		if arr, ok := raw.([]any); ok {
			for _, item := range arr {
				obj, ok := item.(map[string]any)
				if !ok {
					continue
				}
				rawType := strings.TrimSpace(asStringAny(obj["type"]))
				if rawType == "" {
					continue
				}
				setRawOptionLabel(deviceTypes, rawType, caslDeviceTypeDisplayNameWithDict(dict, rawType))
			}
		}
	}
	for rawType, label := range deviceTypes {
		setRawOptionLabel(deviceTypes, rawType, FirstNonEmpty(label, caslDeviceTypeDisplayNameWithDict(dict, rawType)))
	}
	return deviceTypes
}

func caslDictionaryOptionsMap(dict map[string]any, keys ...string) map[string]string {
	for _, key := range keys {
		if raw, ok := dict[key]; ok {
			if values := normalizeCASLObjectEditorOptionMap(raw); len(values) > 0 {
				return values
			}
		}
	}
	return map[string]string{}
}

func caslDictionaryTranslateLabel(dict map[string]any, key string) string {
	key = strings.TrimSpace(key)
	if key == "" || len(dict) == 0 {
		return ""
	}
	translateMap := caslDictionaryTranslateMap(dict)
	return strings.TrimSpace(translateMap[key])
}

func caslDictionaryTranslateMap(dict map[string]any) map[string]string {
	if len(dict) == 0 {
		return map[string]string{}
	}
	if translate, ok := dict["translate"]; ok {
		if mapping := caslDictionaryLanguageMap(translate, "uk"); len(mapping) > 0 {
			return mapping
		}
	}
	if nestedRaw, ok := dict["dictionary"]; ok {
		if nested, ok := nestedRaw.(map[string]any); ok {
			if translate, exists := nested["translate"]; exists {
				if mapping := caslDictionaryLanguageMap(translate, "uk"); len(mapping) > 0 {
					return mapping
				}
			}
		}
	}
	return map[string]string{}
}

func caslDictionaryLanguageMap(raw any, lang string) map[string]string {
	root, ok := raw.(map[string]any)
	if !ok || len(root) == 0 {
		return map[string]string{}
	}
	langCandidates := []string{
		strings.TrimSpace(lang),
		strings.ToLower(strings.TrimSpace(lang)),
		strings.ToUpper(strings.TrimSpace(lang)),
		"ua",
		"UA",
		"uk-UA",
		"uk_ua",
	}
	for _, key := range langCandidates {
		if nested, exists := root[key]; exists {
			return normalizeCASLObjectEditorOptionMap(nested)
		}
	}
	return map[string]string{}
}

func asStringAny(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

func parseCASLAnyIntLocal(value any) int {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return 0
	}
	if strings.Contains(text, ".") {
		text = strings.SplitN(text, ".", 2)[0]
	}
	n, err := strconv.Atoi(text)
	if err != nil {
		return 0
	}
	return n
}

func normalizeCASLObjectEditorOptionMap(raw any) map[string]string {
	switch typed := raw.(type) {
	case map[string]string:
		result := make(map[string]string, len(typed))
		for key, value := range typed {
			result[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
		return result
	case map[string]any:
		result := make(map[string]string, len(typed))
		for key, value := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			text := strings.TrimSpace(fmt.Sprint(value))
			if text == "" || text == "<nil>" {
				text = key
			}
			result[key] = text
		}
		return result
	case []string:
		result := make(map[string]string, len(typed))
		for _, value := range typed {
			value = strings.TrimSpace(value)
			if value != "" {
				result[value] = value
			}
		}
		return result
	case []any:
		result := make(map[string]string, len(typed))
		for _, value := range typed {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text == "" || text == "<nil>" {
				continue
			}
			result[text] = text
		}
		return result
	default:
		return map[string]string{}
	}
}

func newCASLListRowTemplate() fyne.CanvasObject {
	title := widget.NewLabel("")
	title.TextStyle = fyne.TextStyle{Bold: true}
	subtitle := widget.NewLabel("")
	return container.NewVBox(title, subtitle)
}

func setCASLListRow(obj fyne.CanvasObject, title, subtitle string) {
	cont, ok := obj.(*fyne.Container)
	if !ok || len(cont.Objects) < 2 {
		return
	}
	if t, ok := cont.Objects[0].(*widget.Label); ok {
		t.SetText(title)
	}
	if s, ok := cont.Objects[1].(*widget.Label); ok {
		s.SetText(subtitle)
	}
}

func hasAnyRoomLines(rooms []contracts.CASLRoomDetails) bool {
	for _, r := range rooms {
		if len(r.Lines) > 0 {
			return true
		}
	}
	return false
}

func NormalizeCASLEditorSIM(raw string) (string, error) {
	return normalizeCASLEditorUAPhone(raw)
}

func normalizeCASLEditorUAPhone(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if strings.Contains(value, "_") {
		return "", fmt.Errorf("має бути введена повністю у форматі +38 (050) 123-45-67")
	}
	value = prepareCASLEditorPhoneForParse(value)

	parsed, err := phonenumbers.Parse(value, "UA")
	if err != nil {
		return "", fmt.Errorf("має бути у форматі +38 (050) 123-45-67")
	}
	if !phonenumbers.IsValidNumberForRegion(parsed, "UA") {
		return "", fmt.Errorf("має бути українським номером у міжнародному форматі")
	}
	e164 := phonenumbers.Format(parsed, phonenumbers.E164)
	return formatCASLEditorNormalizedUAPhone(e164)
}

func prepareCASLEditorPhoneForParse(raw string) string {
	digits := DigitsOnly(raw)
	switch {
	case len(digits) == 12 && strings.HasPrefix(digits, "380"):
		return "+" + digits
	case len(digits) == 11 && strings.HasPrefix(digits, "80"):
		return "+3" + digits
	case len(digits) == 10 && strings.HasPrefix(digits, "0"):
		return "+38" + digits
	case len(digits) == 9:
		return "+380" + digits
	default:
		return raw
	}
}

func formatCASLEditorNormalizedUAPhone(raw string) (string, error) {
	digits := DigitsOnly(raw)
	if len(digits) != 12 || !strings.HasPrefix(digits, "380") {
		return "", fmt.Errorf("має бути українським номером у міжнародному форматі")
	}
	return fmt.Sprintf("+%s (%s) %s-%s-%s", digits[:2], digits[2:5], digits[5:8], digits[8:10], digits[10:12]), nil
}

func TryFormatCASLEditorUserPhone(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	if strings.Contains(value, "_") {
		return "", false
	}
	if digits := DigitsOnly(value); len(digits) < 9 {
		return "", false
	}
	formatted, err := NormalizeCASLEditorSIM(value)
	if err != nil {
		return "", false
	}
	return formatted, true
}

func FormatCASLEditorPhoneProgressively(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}

	digits := DigitsOnly(value)
	if digits == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(digits, "380"):
	case strings.HasPrefix(digits, "80"):
		digits = "3" + digits
	case strings.HasPrefix(digits, "0"):
		digits = "38" + digits
	case len(digits) <= 9:
		digits = "380" + digits
	default:
		digits = "380" + digits
	}

	if len(digits) > 12 {
		digits = digits[:12]
	}
	if len(digits) <= 2 {
		return "+" + digits
	}

	var builder strings.Builder
	builder.WriteString("+")
	builder.WriteString(digits[:2])

	if len(digits) > 2 {
		builder.WriteString(" (")
		end := min(len(digits), 5)
		builder.WriteString(digits[2:end])
		if end == 5 {
			builder.WriteString(")")
		}
	}
	if len(digits) > 5 {
		builder.WriteString(" ")
		end := min(len(digits), 8)
		builder.WriteString(digits[5:end])
	}
	if len(digits) > 8 {
		builder.WriteString("-")
		end := min(len(digits), 10)
		builder.WriteString(digits[8:end])
	}
	if len(digits) > 10 {
		builder.WriteString("-")
		end := min(len(digits), 12)
		builder.WriteString(digits[10:end])
	}
	return builder.String()
}

func FormatCASLEditorSIMForDisplay(raw string) string {
	formatted, err := NormalizeCASLEditorSIM(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	return formatted
}

func BindDebouncedPhoneFormatter(entry *widget.Entry, delay time.Duration, apply func()) {
	if entry == nil {
		return
	}

	var seq uint64
	entry.OnChanged = func(value string) {
		nextSeq := atomic.AddUint64(&seq, 1)
		time.AfterFunc(delay, func() {
			if atomic.LoadUint64(&seq) != nextSeq {
				return
			}
			formatted := FormatCASLEditorPhoneProgressively(value)
			fyne.Do(func() {
				if atomic.LoadUint64(&seq) != nextSeq {
					return
				}
				if strings.TrimSpace(entry.Text) != strings.TrimSpace(value) {
					return
				}
				if formatted != "" && formatted != entry.Text {
					entry.SetText(formatted)
					return
				}
				if apply != nil {
					apply()
				}
			})
		})
	}
}

func FormatCASLEditorLicenceForDisplay(raw string) string {
	return strings.ReplaceAll(strings.TrimSpace(raw), ";", "-")
}

func NormalizeCASLEditorLicenceForSave(raw string) (string, error) {
	value := FormatCASLEditorLicenceForDisplay(raw)
	if value == "" {
		return "", nil
	}
	parts := strings.Split(value, "-")
	if len(parts) != 6 {
		return "", fmt.Errorf("має бути порожнім або у форматі 123-123-123-123-123-123")
	}
	for _, p := range parts {
		if len(p) != 3 {
			return "", fmt.Errorf("має бути порожнім або у форматі 123-123-123-123-123-123")
		}
	}
	return strings.ReplaceAll(value, "-", ";"), nil
}

func NormalizeCASLEditorUserPhone(raw string) (string, error) {
	return normalizeCASLEditorUAPhone(raw)
}

func ValidateCASLLineNumberUnique(lines []contracts.CASLDeviceLineDetails, num int, excludeIdx int) error {
	for i, l := range lines {
		if i == excludeIdx {
			continue
		}
		if l.LineNumber == num {
			return fmt.Errorf("номер зони %d вже використовується", num)
		}
	}
	return nil
}

func ValidateCASLLineNumberRange(num int) error {
	if num < 1 || num > 999 {
		return fmt.Errorf("номер зони має бути в межах 1..999")
	}
	return nil
}

func ValidateCASLLineDescription(value string) error {
	if len([]rune(strings.TrimSpace(value))) == 0 {
		return fmt.Errorf("назва зони не може бути пустою")
	}
	return nil
}

func NextCASLLineNumber(lines []contracts.CASLDeviceLineDetails) int {
	used := make(map[int]struct{}, len(lines))
	for _, line := range lines {
		if line.LineNumber > 0 {
			used[line.LineNumber] = struct{}{}
		}
	}
	for num := 1; num <= 999; num++ {
		if _, ok := used[num]; !ok {
			return num
		}
	}
	return 999
}

func defaultCASLDeviceTimeout(rawType string) int64 {
	normalized := strings.ToUpper(strings.TrimSpace(rawType))
	if normalized == "" {
		return 0
	}
	if strings.Contains(normalized, "DUNAY") {
		return 240
	}
	return 1000
}

var caslLineLimitByDeviceType = map[string]int{
	"TYPE_DEVICE_DUNAY_4L":                128,
	"TYPE_DEVICE_CASL":                    20,
	"TYPE_DEVICE_DUNAY_8L":                8,
	"TYPE_DEVICE_LUN":                     128,
	"TYPE_DEVICE_AJAX":                    256,
	"TYPE_DEVICE_AJAX_SIA":                256,
	"TYPE_DEVICE_BRON_SIA":                128,
	"TYPE_DEVICE_DUNAY_8(16)32_DUNAY_G1R": 128,
	"TYPE_DEVICE_DUNAY_8_32":              128,
	"TYPE_DEVICE_DUNAY_16_32":             128,
	"TYPE_DEVICE_DUNAY_4_3":               4,
	"TYPE_DEVICE_DUNAY_4_3S":              4,
	"TYPE_DEVICE_VBD4_ECOM":               4,
	"TYPE_DEVICE_VBD_16":                  6,
	"TYPE_DEVICE_DOZOR_4":                 4,
	"TYPE_DEVICE_DOZOR_8":                 8,
	"TYPE_DEVICE_DOZOR_8MG":               8,
	"TYPE_DEVICE_DUNAY_PSPN_ECOM":         128,
	"FULL_SURGARD":                        256,
	"MAKS_PRO":                            256,
	"SATEL":                               999,
	"ИНТТЕЛ":                              128,
	"\"МАКС-ПРО\"":                        256,
	"TYPE_DEVICE_DUNAY_STK":               1,
	"TYPE_DEVICE_DUNAY_4.2":               4,
	"TYPE_DEVICE_VBDB_2":                  6,
	"TYPE_DEVICE_VBD4":                    4,
	"TYPE_DEVICE_DUNAY_PSPN":              128,
	"TYPE_DEVICE_DUNAY_16L":               128,
	"TYPE_DEVICE_CASL_PLUS":               128,
}

var caslAdapterLimitByDeviceType = map[string]map[string]int{
	"TYPE_DEVICE_DUNAY_4L": {
		"SYS":   4,
		"AD3L":  3,
		"AD6L":  6,
		"AD6WL": 6,
		"UTS4":  1,
	},
	"TYPE_DEVICE_CASL": {
		"SYS":   4,
		"AD3L":  3,
		"AD6L":  6,
		"AD6WL": 6,
		"UTS4":  1,
	},
	"TYPE_DEVICE_DUNAY_8L": {
		"SYS": 8,
	},
}

func caslDeviceLineLimit(rawType string) int {
	normalized := strings.ToUpper(strings.TrimSpace(rawType))
	if limit, ok := caslLineLimitByDeviceType[normalized]; ok {
		return limit
	}
	return 999
}

func isUserCreatedCASLDeviceType(dict map[string]any, rawType string) bool {
	rawType = strings.TrimSpace(rawType)
	if rawType == "" {
		return false
	}
	options := caslDictionaryOptionsMap(dict, "user_device_types")
	if len(options) == 0 {
		return false
	}
	for candidate := range options {
		if strings.EqualFold(strings.TrimSpace(candidate), rawType) {
			return true
		}
	}
	return false
}

func canEditCASLAdapterForDevice(dict map[string]any, rawType string) bool {
	rawType = strings.TrimSpace(rawType)
	if rawType == "" {
		return false
	}
	return !isUserCreatedCASLDeviceType(dict, rawType)
}

func caslAdapterOptionLabelsForDevice(dict map[string]any, deviceType string, mapping map[string]string) []string {
	deviceType = strings.ToUpper(strings.TrimSpace(deviceType))
	if len(mapping) == 0 {
		return nil
	}

	allowed := make(map[string]struct{}, len(mapping))
	switch {
	case isUserCreatedCASLDeviceType(dict, deviceType):
		allowed["SYS"] = struct{}{}
	case deviceType != "":
		if limits, ok := caslAdapterLimitByDeviceType[deviceType]; ok {
			for adapterType := range limits {
				allowed[strings.TrimSpace(adapterType)] = struct{}{}
			}
			break
		}
		fallthrough
	default:
		for _, adapterType := range mapping {
			allowed[strings.TrimSpace(adapterType)] = struct{}{}
		}
	}

	labels := make([]string, 0, len(allowed))
	for label, adapterType := range mapping {
		if _, ok := allowed[strings.TrimSpace(adapterType)]; ok {
			labels = append(labels, label)
		}
	}
	sort.Strings(labels)
	return labels
}

func nextCASLLineDefaults(dict map[string]any, deviceType string, lines []contracts.CASLDeviceLineDetails) (int, string, int, int, string) {
	lineNumber := NextCASLLineNumber(lines)
	groupNumber := 1
	lineType := "NORMAL"
	adapterType := "SYS"
	adapterNumber := 0

	deviceType = strings.ToUpper(strings.TrimSpace(deviceType))
	if isUserCreatedCASLDeviceType(dict, deviceType) {
		return lineNumber, adapterType, adapterNumber, groupNumber, lineType
	}

	countSYS := 0
	countNonUTS4 := 0
	for _, line := range lines {
		if strings.EqualFold(strings.TrimSpace(line.AdapterType), "SYS") {
			countSYS++
		}
		if !strings.EqualFold(strings.TrimSpace(line.AdapterType), "UTS4") {
			countNonUTS4++
		}
	}

	if deviceType == "TYPE_DEVICE_DUNAY_4L" || deviceType == "TYPE_DEVICE_CASL" {
		if countSYS == 4 {
			if countNonUTS4 == 16 {
				adapterType = "UTS4"
			} else {
				adapterType = "AD3L"
			}
		}
	}

	if adapterType == "SYS" {
		return lineNumber, adapterType, 0, groupNumber, lineType
	}

	if limits, ok := caslAdapterLimitByDeviceType[deviceType]; ok {
		if limit := limits[adapterType]; limit > 0 {
			counts := map[int]int{}
			for _, line := range lines {
				if strings.EqualFold(strings.TrimSpace(line.AdapterType), adapterType) {
					counts[line.AdapterNumber]++
				}
			}
			for num, count := range counts {
				if count < limit {
					return lineNumber, adapterType, num, groupNumber, lineType
				}
			}
		}
	}

	used := make(map[int]struct{}, len(lines))
	for _, line := range lines {
		used[line.AdapterNumber] = struct{}{}
	}
	for num := 1; num <= 10; num++ {
		if _, ok := used[num]; !ok {
			return lineNumber, adapterType, num, groupNumber, lineType
		}
	}

	return lineNumber, adapterType, 1, groupNumber, lineType
}

func ValidateCASLEditorDateField(raw string) error {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	// Try UA format
	if _, err := time.Parse("02.01.2006", v); err == nil {
		return nil
	}
	// Try ISO format
	if _, err := time.Parse("2006-01-02", v); err == nil {
		return nil
	}
	// Try UA short format
	if _, err := time.Parse("2.1.2006", v); err == nil {
		return nil
	}
	return fmt.Errorf("некоректний формат дати")
}

func DigitsOnly(raw string) string {
	var builder strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func int64ToString(value int64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func float64PtrToString(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func parseCASLEditorInt64(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

func parseCASLEditorInt(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func parseCASLEditorFloatPtr(raw string) (*float64, error) {
	value := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func CaslDatePtr(raw int64) *time.Time {
	if raw <= 0 {
		return nil
	}
	value := time.UnixMilli(raw).Local()
	return &value
}

func SetCASLEditorDateEntry(entry *widget.DateEntry, value *time.Time) {
	if entry == nil {
		return
	}
	if value == nil {
		entry.SetText("")
		return
	}
	entry.SetText(value.Format("02.01.2006"))
}

func DateEntryUnixMilli(entry *widget.DateEntry) (int64, error) {
	if entry == nil || entry.Text == "" {
		return 0, nil
	}
	t, err := time.Parse("02.01.2006", entry.Text)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}

func ParseCASLEditorInt64(raw string) (int64, error) {
	return parseCASLEditorInt64(raw)
}

func ParseCASLEditorInt(raw string) (int, error) {
	return parseCASLEditorInt(raw)
}

func ParseCASLEditorFloatPtr(raw string) (*float64, error) {
	return parseCASLEditorFloatPtr(raw)
}
func TranslateRole(role string) string {
	switch role {
	case "ADMIN":
		return "Адміністратор"
	case "MANAGER":
		return "Менеджер"
	case "TECH":
		return "Інженер"
	case "TECH_REG":
		return "Регламентний інженер"
	case "OPERATOR":
		return "Оператор"
	default:
		return role
	}
}
