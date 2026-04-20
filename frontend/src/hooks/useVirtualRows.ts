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

  useEffect(() => {
    setLoadedCount((current) => Math.min(rows.length, Math.max(initialCount, current)))
  }, [rows.length, initialCount])

  useEffect(() => {
    if (rows.length === 0) {
      setLoadedCount(0)
      setScrollTop(0)
      return
    }

    const container = containerRef.current
    if (container != null && container.scrollTop === 0) {
      setScrollTop(0)
    }
  }, [rows.length])

  const loadMoreIfNeeded = useCallback(
    (currentScrollTop: number, currentViewportHeight: number) => {
      if (rows.length === 0) {
        return
      }

      const thresholdPx = rowHeight * 6
      const currentLoadedHeight = loadedCount * rowHeight
      if (currentScrollTop + currentViewportHeight >= currentLoadedHeight - thresholdPx && loadedCount < rows.length) {
        setLoadedCount((current) => Math.min(rows.length, current + step))
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

  useEffect(() => {
    const container = containerRef.current
    if (container == null) {
      return
    }
    setViewportHeight(container.clientHeight)
    loadMoreIfNeeded(container.scrollTop, container.clientHeight)
  }, [loadMoreIfNeeded, rows.length])

  const effectiveLoadedCount = Math.min(rows.length, loadedCount)
  const virtualStart = Math.max(0, Math.floor(scrollTop / rowHeight) - overscanRows)
  const virtualEnd = Math.min(
    effectiveLoadedCount,
    Math.ceil((scrollTop + Math.max(viewportHeight, rowHeight)) / rowHeight) + overscanRows,
  )
  const startIndex = Math.min(virtualStart, virtualEnd)

  return {
    containerRef,
    onScroll,
    visibleRows: rows.slice(startIndex, virtualEnd),
    startIndex,
    topPaddingPx: startIndex * rowHeight,
    bottomPaddingPx: Math.max(0, (effectiveLoadedCount - virtualEnd) * rowHeight),
    loadedCount: effectiveLoadedCount,
    totalCount: rows.length,
  }
}
