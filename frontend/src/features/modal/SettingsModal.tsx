import type { FrontendDBSettings } from '../../shared/api/types'
import type { ThemeMode } from '../../shared/state/theme-store'

type SettingsModalProps = {
  isOpen: boolean
  settingsDraft: FrontendDBSettings | null
  settingsBusy: boolean
  settingsError: string
  settingsSuccess: string
  themeMode: ThemeMode
  onClose: () => void
  onSave: () => void
  onUpdateDraft: (patch: Partial<FrontendDBSettings>) => void
  onThemeChange: (theme: ThemeMode) => void
}

export function SettingsModal({
  isOpen,
  settingsDraft,
  settingsBusy,
  settingsError,
  settingsSuccess,
  themeMode,
  onClose,
  onSave,
  onUpdateDraft,
  onThemeChange,
}: SettingsModalProps) {
  return (
    <div className={isOpen ? 'modal-overlay open' : 'modal-overlay'}>
      <div className="modal wide settings-modal">
        <div className="modal-tb">
          <div className="modal-tb-icon" style={{ background: 'var(--ac2)' }}>
            <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="2.5">
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 010 2.83 2 2 0 01-2.83 0l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" />
            </svg>
          </div>
          <span className="modal-tb-title">Налаштування підключень</span>
          <div className="modal-tb-close" onClick={onClose}>✕</div>
        </div>

        <div className="modal-content">
          <div className="modal-pane active settings-pane">
            {settingsError !== '' && <div className="settings-banner settings-banner-error">{settingsError}</div>}
            {settingsSuccess !== '' && <div className="settings-banner settings-banner-success">{settingsSuccess}</div>}

            {settingsDraft == null ? (
              <div className="settings-loading">Завантаження налаштувань...</div>
            ) : (
              <div className="igrid">
                <div className="isection settings-theme-section" style={{ gridColumn: '1 / span 2' }}>
                  <div className="isect-title">Оформлення</div>
                  <div className="theme-switcher">
                    <button
                      type="button"
                      className={themeMode === 'dark' ? 'theme-option active' : 'theme-option'}
                      onClick={() => onThemeChange('dark')}
                    >
                      <span className="theme-option-title">Темна тема</span>
                      <span className="theme-option-desc">Поточний темний інтерфейс оператора</span>
                    </button>
                    <button
                      type="button"
                      className={themeMode === 'light' ? 'theme-option active' : 'theme-option'}
                      onClick={() => onThemeChange('light')}
                    >
                      <span className="theme-option-title">Світла тема</span>
                      <span className="theme-option-desc">Світлий варіант з тим самим layout і акцентами</span>
                    </button>
                  </div>
                </div>

                <div className="isection">
                  <div className="isect-title">Firebird / МІСТ</div>
                  <CheckboxRow label="Активувати" checked={settingsDraft.firebirdEnabled} onChange={(value) => onUpdateDraft({ firebirdEnabled: value })} />
                  <TextRow label="Користувач" value={settingsDraft.firebirdUser} onChange={(value) => onUpdateDraft({ firebirdUser: value })} />
                  <PasswordRow label="Пароль" value={settingsDraft.firebirdPassword} onChange={(value) => onUpdateDraft({ firebirdPassword: value })} />
                  <TextRow label="Host" value={settingsDraft.firebirdHost} onChange={(value) => onUpdateDraft({ firebirdHost: value })} />
                  <TextRow label="Port" value={settingsDraft.firebirdPort} onChange={(value) => onUpdateDraft({ firebirdPort: value })} />
                  <TextRow label="Path" value={settingsDraft.firebirdPath} onChange={(value) => onUpdateDraft({ firebirdPath: value })} />
                  <TextRow label="Params" value={settingsDraft.firebirdParams} onChange={(value) => onUpdateDraft({ firebirdParams: value })} />
                </div>

                <div className="isection">
                  <div className="isect-title">Phoenix</div>
                  <CheckboxRow label="Активувати" checked={settingsDraft.phoenixEnabled} onChange={(value) => onUpdateDraft({ phoenixEnabled: value })} />
                  <TextRow label="Користувач" value={settingsDraft.phoenixUser} onChange={(value) => onUpdateDraft({ phoenixUser: value })} />
                  <PasswordRow label="Пароль" value={settingsDraft.phoenixPassword} onChange={(value) => onUpdateDraft({ phoenixPassword: value })} />
                  <TextRow label="Host" value={settingsDraft.phoenixHost} onChange={(value) => onUpdateDraft({ phoenixHost: value })} />
                  <TextRow label="Port" value={settingsDraft.phoenixPort} onChange={(value) => onUpdateDraft({ phoenixPort: value })} />
                  <TextRow label="Instance" value={settingsDraft.phoenixInstance} onChange={(value) => onUpdateDraft({ phoenixInstance: value })} />
                  <TextRow label="Database" value={settingsDraft.phoenixDatabase} onChange={(value) => onUpdateDraft({ phoenixDatabase: value })} />
                  <TextRow label="Params" value={settingsDraft.phoenixParams} onChange={(value) => onUpdateDraft({ phoenixParams: value })} />
                </div>

                <div className="isection" style={{ gridColumn: '1 / span 2' }}>
                  <div className="isect-title">CASL Cloud</div>
                  <CheckboxRow label="Активувати" checked={settingsDraft.caslEnabled} onChange={(value) => onUpdateDraft({ caslEnabled: value })} />
                  <TextRow label="Base URL" value={settingsDraft.caslBaseURL} onChange={(value) => onUpdateDraft({ caslBaseURL: value })} />
                  <TextRow label="Token" value={settingsDraft.caslToken} onChange={(value) => onUpdateDraft({ caslToken: value })} />
                  <TextRow label="Email" value={settingsDraft.caslEmail} onChange={(value) => onUpdateDraft({ caslEmail: value })} />
                  <PasswordRow label="Password" value={settingsDraft.caslPass} onChange={(value) => onUpdateDraft({ caslPass: value })} />
                  <NumberRow label="Pult ID" value={settingsDraft.caslPultID} onChange={(value) => onUpdateDraft({ caslPultID: value })} />
                </div>
              </div>
            )}
          </div>
        </div>

        <div className="modal-footer">
          <button className="btn btn-gray" style={{ width: 140, height: 28 }} onClick={onClose}>ЗАКРИТИ</button>
          <div style={{ marginLeft: 'auto' }} />
          <button className="btn btn-green" style={{ width: 180, height: 28 }} onClick={onSave} disabled={settingsBusy}>
            {settingsBusy ? 'ЗБЕРЕЖЕННЯ...' : 'ЗБЕРЕГТИ'}
          </button>
        </div>
      </div>
    </div>
  )
}

function CheckboxRow({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) {
  return (
    <div className="irow">
      <label>{label}</label>
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
    </div>
  )
}

function TextRow({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <div className="irow">
      <label>{label}</label>
      <input value={value} onChange={(event) => onChange(event.target.value)} />
    </div>
  )
}

function PasswordRow({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <div className="irow">
      <label>{label}</label>
      <input type="password" value={value} onChange={(event) => onChange(event.target.value)} />
    </div>
  )
}

function NumberRow({ label, value, onChange }: { label: string; value: number; onChange: (value: number) => void }) {
  return (
    <div className="irow">
      <label>{label}</label>
      <input type="number" value={value} onChange={(event) => onChange(Number.parseInt(event.target.value, 10) || 0)} />
    </div>
  )
}
