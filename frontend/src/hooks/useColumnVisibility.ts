import { useState, useCallback, useEffect } from 'react'
import type { VisibilityState } from '@tanstack/react-table'

const STORAGE_PREFIX = 'col-vis-'

function loadVisibility(key: string): VisibilityState {
  try {
    const raw = localStorage.getItem(STORAGE_PREFIX + key)
    if (raw != null) {
      return JSON.parse(raw) as VisibilityState
    }
  } catch {
    // ignore corrupted data
  }
  return {}
}

function saveVisibility(key: string, state: VisibilityState): void {
  try {
    localStorage.setItem(STORAGE_PREFIX + key, JSON.stringify(state))
  } catch {
    // ignore write errors
  }
}

export function useColumnVisibility(storageKey: string) {
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>(() => loadVisibility(storageKey))

  useEffect(() => {
    saveVisibility(storageKey, columnVisibility)
  }, [storageKey, columnVisibility])

  const toggleColumn = useCallback((columnId: string) => {
    setColumnVisibility((prev) => ({
      ...prev,
      [columnId]: prev[columnId] === false ? true : prev[columnId] === true ? false : false,
    }))
  }, [])

  const resetAll = useCallback(() => {
    setColumnVisibility({})
  }, [])

  return { columnVisibility, setColumnVisibility, toggleColumn, resetAll }
}
