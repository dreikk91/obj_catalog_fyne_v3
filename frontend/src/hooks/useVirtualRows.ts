import { useCallback, useEffect, useRef, useState } from 'react'
import type { UIEvent } from 'react'
import { VIRTUAL_INITIAL_CHUNK, VIRTUAL_OVERSCAN_ROWS, VIRTUAL_STEP_CHUNK } from '../features/operator/constants'
import type { VirtualRowsOptions, VirtualRowsSlice } from '../features/operator/types'

export function useVirtualRows<T>(rows: T[], options: VirtualRowsOptions): VirtualRowsSlice<T> {
  const {
    rowHeight,
    initialCount = VIRTUAL_INITIAL_CHUNK,
    step = VIRTUAL_STEP_CHUNK,
    overscanRows = VIRTUAL_OVERSCAN_ROWS,
  } = options

  const containerRef = useRef<HTMLDivElement | null>(null)
  const [scrollTop, setScrollTop] = useState(0)
  const [viewportHeight, setViewportHeight] = useState(0)
  const [loadedCount, setLoadedCount] = useState(() => Math.min(rows.length, initialCount))

  // Автоматично збільшуємо loadedCount при появі нових рядків (loadMore)
  useEffect(() => {
    setLoadedCount((prev) => {
      if (rows.length <= prev) return prev
      // Збільшуємо на крок, але не більше реальної довжини
      return Math.min(rows.length, prev + step)
    })
  }, [rows.length, step])

  // Ініціалізація при першому завантаженні
  useEffect(() => {
    if (rows.length === 0) {
      setLoadedCount(0)
      return
    }
    setLoadedCount((prev) => Math.min(rows.length, Math.max(initialCount, prev)))
  }, [rows.length, initialCount])

  const loadMoreIfNeeded = useCallback(
    (currentScrollTop: number, currentViewportHeight: number) => {
      if (rows.length === 0 || loadedCount >= rows.length) return

      const thresholdPx = rowHeight * 8 // трохи більший поріг для стабільності
      const currentLoadedHeight = loadedCount * rowHeight

      if (currentScrollTop + currentViewportHeight >= currentLoadedHeight - thresholdPx) {
        setLoadedCount((prev) => Math.min(rows.length, prev + step))
      }
    },
    [loadedCount, rowHeight, rows.length, step],
  )

  const onScroll = useCallback(
    (event: UIEvent<HTMLDivElement>) => {
      const target = event.currentTarget
      setScrollTop(target.scrollTop)
      setViewportHeight(target.clientHeight)
      loadMoreIfNeeded(target.scrollTop, target.clientHeight)
    },
    [loadMoreIfNeeded],
  )

  // Початкова ініціалізація viewport і loadedCount
  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    setViewportHeight(container.clientHeight)
    loadMoreIfNeeded(container.scrollTop, container.clientHeight)
  }, [loadMoreIfNeeded])

  const effectiveLoadedCount = Math.min(rows.length, loadedCount)

  const virtualStart = Math.max(0, Math.floor(scrollTop / rowHeight) - overscanRows)
  const virtualEnd = Math.min(
    effectiveLoadedCount,
    Math.ceil((scrollTop + Math.max(viewportHeight, rowHeight)) / rowHeight) + overscanRows,
  )

  return {
    containerRef,
    onScroll,
    visibleRows: rows.slice(virtualStart, virtualEnd),
    startIndex: virtualStart,
    topPaddingPx: virtualStart * rowHeight,
    bottomPaddingPx: Math.max(0, (effectiveLoadedCount - virtualEnd) * rowHeight),
    loadedCount: effectiveLoadedCount,
    totalCount: rows.length,
  }
}