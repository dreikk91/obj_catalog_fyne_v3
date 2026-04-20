import type { InnerTab } from '../../shared/state/ui-store'
import type { ObjectRow } from '../operator/types'

type InnerTabsProps = {
  innerTab: InnerTab
  onSelectTab: (tab: InnerTab) => void
  selectedObjectRow: ObjectRow | null
}

export function InnerTabs({ innerTab, onSelectTab, selectedObjectRow }: InnerTabsProps) {
  return (
    <div className="ps-tabs-right">
      <div className="inner-tabs">
        <div className={innerTab === 'notes' ? 'itab active' : 'itab'} onClick={() => onSelectTab('notes')}>
          Примітки
        </div>
        <div className={innerTab === 'extra' ? 'itab active' : 'itab'} onClick={() => onSelectTab('extra')}>
          Додатково
        </div>
        <div className={innerTab === 'subs' ? 'itab active' : 'itab'} onClick={() => onSelectTab('subs')}>
          Заміни
        </div>
        <div className={innerTab === 'rent' ? 'itab active' : 'itab'} onClick={() => onSelectTab('rent')}>
          Обладнання в оренді
        </div>
      </div>
      <div className={innerTab === 'notes' ? 'inner-pane active' : 'inner-pane'}>
        <textarea className="note-area" value={selectedObjectRow?.note ?? ''} readOnly />
      </div>
      <div className={innerTab === 'extra' ? 'inner-pane active' : 'inner-pane'}>
        <div style={{ padding: 10, color: 'var(--tx2)', fontSize: 12 }}>Додаткові відомості про об'єкт</div>
      </div>
      <div className={innerTab === 'subs' ? 'inner-pane active' : 'inner-pane'}>
        <div style={{ padding: 10, color: 'var(--tx2)', fontSize: 12 }}>Заміни охоронців</div>
      </div>
      <div className={innerTab === 'rent' ? 'inner-pane active' : 'inner-pane'}>
        <div style={{ padding: 10, color: 'var(--tx2)', fontSize: 12 }}>Перелік орендованого обладнання</div>
      </div>
    </div>
  )
}
