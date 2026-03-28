package application

import (
	"context"
	"time"

	"obj_catalog_fyne_v3/pkg/eventbus"

	"github.com/rs/zerolog/log"
)

type latestEventIDProvider interface {
	GetLatestEventID() (int64, error)
}

const (
	eventProbeInterval       = 2 * time.Second
	eventsReconcileInterval  = 30 * time.Second
	alarmsReconcileInterval  = 10 * time.Second
	objectsReconcileInterval = 20 * time.Second
	fallbackRefreshInterval  = 4 * time.Second
)

func shouldRefreshForLatestEventID(latestID, lastKnownID int64, hasLastKnownID bool) (refresh bool, nextLastKnownID int64, nextHasLastKnownID bool) {
	if !hasLastKnownID {
		return false, latestID, true
	}
	if latestID != lastKnownID {
		return true, latestID, true
	}
	return false, lastKnownID, true
}

// startGettingEvents запускає фоновий scheduler оновлень:
// - швидкий probe останнього event ID (дешевий SQL), без постійного Refresh усіх панелей;
// - подієве оновлення через EventBus тільки при реальних змінах;
// - періодична reconcile-синхронізація для гарантії консистентності.
func (a *Application) startGettingEvents() {
	if a == nil {
		return
	}

	if a.refreshLoopCancel != nil {
		a.refreshLoopCancel()
		a.refreshLoopCancel = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.refreshLoopCancel = cancel

	go func() {
		eventProbeTicker := time.NewTicker(eventProbeInterval)
		eventsReconcileTicker := time.NewTicker(eventsReconcileInterval)
		alarmsReconcileTicker := time.NewTicker(alarmsReconcileInterval)
		objectsReconcileTicker := time.NewTicker(objectsReconcileInterval)
		fallbackTicker := time.NewTicker(fallbackRefreshInterval)
		defer eventProbeTicker.Stop()
		defer eventsReconcileTicker.Stop()
		defer alarmsReconcileTicker.Stop()
		defer objectsReconcileTicker.Stop()
		defer fallbackTicker.Stop()

		var (
			lastKnownEventID int64
			hasLastKnownID   bool
		)

		for {
			select {
			case <-ctx.Done():
				return

			case <-eventProbeTicker.C:
				probe, ok := a.dataProvider.(latestEventIDProvider)
				if !ok {
					continue
				}

				latestID, err := probe.GetLatestEventID()
				if err != nil {
					log.Debug().Err(err).Msg("Не вдалося виконати probe останнього event ID")
					continue
				}

				refresh, nextID, hasNext := shouldRefreshForLatestEventID(latestID, lastKnownEventID, hasLastKnownID)
				lastKnownEventID = nextID
				hasLastKnownID = hasNext
				if refresh {
					a.publishDataRefresh(eventbus.DataRefreshEvent{
						RefreshAlarms: true,
						RefreshEvents: true,
					})
				}

			case <-eventsReconcileTicker.C:
				a.publishDataRefresh(eventbus.DataRefreshEvent{RefreshEvents: true})

			case <-alarmsReconcileTicker.C:
				a.publishDataRefresh(eventbus.DataRefreshEvent{RefreshAlarms: true})

			case <-objectsReconcileTicker.C:
				a.publishDataRefresh(eventbus.DataRefreshEvent{RefreshObjects: true})

			case <-fallbackTicker.C:
				// Fallback для провайдерів без інтерфейсу probe (наприклад, мок/тест),
				// щоб не втрачати автооновлення навіть без event cursor API.
				if _, ok := a.dataProvider.(latestEventIDProvider); !ok {
					a.publishDataRefresh(eventbus.DataRefreshEvent{
						RefreshObjects: true,
						RefreshAlarms:  true,
						RefreshEvents:  true,
					})
				}
			}
		}
	}()
}
