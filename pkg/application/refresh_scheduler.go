package application

import (
	"context"
	"time"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/eventbus"

	"github.com/rs/zerolog/log"
)

type latestEventIDProvider interface {
	GetLatestEventID() (int64, error)
}

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

	uiCfg := config.LoadUIConfig(a.fyneApp.Preferences())
	eventProbeInterval := time.Duration(uiCfg.EventProbeIntervalSec) * time.Second
	eventsReconcileInterval := time.Duration(uiCfg.EventsReconcileSec) * time.Second
	alarmsReconcileInterval := time.Duration(uiCfg.AlarmsReconcileSec) * time.Second
	objectsReconcileInterval := time.Duration(uiCfg.ObjectsReconcileSec) * time.Second
	fallbackRefreshInterval := time.Duration(uiCfg.FallbackRefreshSec) * time.Second
	maxProbeBackoffInterval := time.Duration(uiCfg.MaxProbeBackoffSec) * time.Second

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
			probeBackoff     time.Duration
			nextProbeAt      time.Time
		)

		for {
			select {
			case <-ctx.Done():
				return

			case <-eventProbeTicker.C:
				now := time.Now()
				if !nextProbeAt.IsZero() && now.Before(nextProbeAt) {
					continue
				}

				provider := a.getDataProvider()
				probe, ok := provider.(latestEventIDProvider)
				if !ok {
					probeBackoff = 0
					nextProbeAt = time.Time{}
					continue
				}

				latestID, err := probe.GetLatestEventID()
				if err != nil {
					if probeBackoff == 0 {
						probeBackoff = eventProbeInterval
					} else {
						probeBackoff *= 2
						if probeBackoff > maxProbeBackoffInterval {
							probeBackoff = maxProbeBackoffInterval
						}
					}
					nextProbeAt = now.Add(probeBackoff)
					log.Debug().Err(err).Msg("Не вдалося виконати probe останнього event ID")
					continue
				}
				probeBackoff = 0
				nextProbeAt = time.Time{}

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
				if _, ok := a.getDataProvider().(latestEventIDProvider); !ok {
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
