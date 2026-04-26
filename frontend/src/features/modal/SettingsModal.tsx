import type { FrontendAMISettings, FrontendDBSettings } from '../../shared/api/types'
import type { ThemeMode } from '../../shared/state/theme-store'
import type { LogLevel } from '../../shared/state/log-store'

type SettingsModalProps = {
  isOpen: boolean
  settingsDraft: FrontendDBSettings | null
  settingsBusy: boolean
  settingsError: string
  settingsSuccess: string
  themeMode: ThemeMode
  logLevel: LogLevel
  amiDraft: FrontendAMISettings | null
  amiError: string
  amiSuccess: string
  amiBusy: boolean
  amiConnected: boolean | null
  onClose: () => void
  onSave: () => void
  onUpdateDraft: (patch: Partial<FrontendDBSettings>) => void
  onThemeChange: (theme: ThemeMode) => void
  onLogLevelChange: (level: LogLevel) => void
  onUpdateAMI: (patch: Partial<FrontendAMISettings>) => void
  onSaveAMI: () => void
}

export function SettingsModal({
  isOpen,
  settingsDraft,
  settingsBusy,
  settingsError,
  settingsSuccess,
  themeMode,
  logLevel,
  amiDraft,
  amiError,
  amiSuccess,
  amiBusy,
  amiConnected,
  onClose,
  onSave,
  onUpdateDraft,
  onThemeChange,
  onLogLevelChange,
  onUpdateAMI,
  onSaveAMI,
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
                  <div className="irow" style={{ borderTop: '1px solid var(--bd)', marginTop: 6 }}>
                    <label>Рівень логування</label>
                    <select value={logLevel} onChange={(e) => onLogLevelChange(e.target.value as import('../../shared/state/log-store').LogLevel)}>
                      <option value="debug">debug — усі повідомлення</option>
                      <option value="info">info — інформація+</option>
                      <option value="warn">warn — попередження+ (стандарт)</option>
                      <option value="error">error — лише помилки</option>
                      <option value="off">off — вимкнено</option>
                    </select>
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

            <div className="settings-section-sep" />

            <div className="isect-title" style={{ padding: '8px 0 4px' }}>
              Asterisk (Click-to-Call)
              {amiConnected != null && (
                <span className={`ami-status-dot ${amiConnected ? 'ami-status-dot--ok' : 'ami-status-dot--off'}`}>
                  {amiConnected ? '● підключено' : '● відключено'}
                </span>
              )}
            </div>
            {amiError !== '' && <div className="settings-banner settings-banner-warn">{amiError}</div>}
            {amiSuccess !== '' && <div className="settings-banner settings-banner-success">{amiSuccess}</div>}
            {amiDraft == null ? (
              <div className="settings-loading">Завантаження налаштувань AMI...</div>
            ) : (
              <div className="igrid" style={{ gridTemplateColumns: '1fr 1fr' }}>
                <div className="isection">
                  <CheckboxRow label="Активувати" checked={amiDraft.enabled} onChange={(v) => onUpdateAMI({ enabled: v })} />
                  <TextRow label="Host" value={amiDraft.host} onChange={(v) => onUpdateAMI({ host: v })} />
                  <NumberRow label="Port" value={amiDraft.port} onChange={(v) => onUpdateAMI({ port: v })} />
                  <TextRow label="Username" value={amiDraft.username} onChange={(v) => onUpdateAMI({ username: v })} />
                  <PasswordRow label="Secret" value={amiDraft.secret} onChange={(v) => onUpdateAMI({ secret: v })} />
                </div>
                <div className="isection">
                  <TextRow label="Extension (лінія оператора)" value={amiDraft.extension} onChange={(v) => onUpdateAMI({ extension: v })} />
                  <TextRow label="Context" value={amiDraft.context} onChange={(v) => onUpdateAMI({ context: v })} />
                  <div style={{ marginTop: 8 }}>
                    <button className="btn btn-blue" style={{ height: 28 }} onClick={onSaveAMI} disabled={amiBusy}>
                      {amiBusy ? 'ЗБЕРЕЖЕННЯ...' : 'ЗБЕРЕГТИ AMI'}
                    </button>
                  </div>
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
