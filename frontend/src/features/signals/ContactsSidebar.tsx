import type { FrontendObjectDetails } from '../../shared/api/types'
import { CallButton } from '../../shared/ui/CallButton'

type ContactsSidebarProps = {
  objectDetails: FrontendObjectDetails | null
}

export function ContactsSidebar({ objectDetails }: ContactsSidebarProps) {
  const contacts = objectDetails?.contacts ?? []
  const generalPhones = objectDetails?.phones?.trim() ?? ''
  const hasContent = contacts.length > 0 || generalPhones !== ''

  return (
    <div className="ps-groups">
      <div className="ps-groups-title">Контакти</div>

      {objectDetails == null && (
        <div className="ps-groups-empty">Оберіть тривогу або об'єкт</div>
      )}

      {objectDetails != null && !hasContent && (
        <div className="ps-groups-empty">Контакти не вказані</div>
      )}

      {hasContent && (
        <div className="contacts-list">
          {generalPhones !== '' && (
            <div className="contact-section-header">Телефони об'єкта</div>
          )}
          {generalPhones !== '' && (
            <div className="contact-card">
              <div className="contact-phone-row">
                <div className="contact-phone">{generalPhones}</div>
                <CallButton phone={generalPhones} contactName="Телефон об'єкта" />
              </div>
            </div>
          )}

          {contacts.length > 0 && <div className="contact-section-header">Відповідальні особи</div>}

          {contacts.map((c, idx) => (
            <div key={idx} className="contact-card">
              <div className="contact-name">{c.name || '—'}</div>
              {c.position && <div className="contact-position">{c.position}</div>}
              {c.phone && (
                <div className="contact-phone-row">
                  <div className="contact-phone">{c.phone}</div>
                  <CallButton phone={c.phone} contactName={c.name || c.phone} />
                </div>
              )}
              {c.groupName && (
                <div className="contact-group">
                  <span className="contact-group-name">{c.groupName}</span>
                  {c.groupStateText && (
                    <span className="contact-group-state">{c.groupStateText}</span>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
