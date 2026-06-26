package main

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

func (b *Bridge) startPhoenixEventWatcher(ctx context.Context) <-chan struct{} {
	if b.phoenix == nil {
		return nil
	}
	interval := b.cfg.PhoenixProbeInterval.Duration()
	if interval <= 0 {
		return nil
	}

	wake := make(chan struct{}, 1)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		var last int64
		if value, err := b.phoenix.GetLatestEventID(); err == nil {
			last = value
		} else {
			log.Error().Err(err).Msg("bridge: phoenix latest event probe")
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				value, err := b.phoenix.GetLatestEventID()
				if err != nil {
					log.Error().Err(err).Msg("bridge: phoenix latest event probe")
					continue
				}
				if last == 0 {
					last = value
					continue
				}
				if value <= last {
					continue
				}
				last = value
				select {
				case wake <- struct{}{}:
				default:
				}
			}
		}
	}()
	return wake
}
