import { useCallback, useEffect, useReducer, useRef } from 'react'
import type { FrontendClient } from '../shared/api/frontend-client'
import type { FrontendEventItem } from '../shared/api/types'
import { OBJECT_EVENTS_PAGE_SIZE, OBJECT_EVENTS_REFRESH_MS } from '../features/operator/constants'
import { compareEventItemsDesc, mergeRecentEvents } from '../features/operator/utils'

type ObjectEventsState = {
  events: FrontendEventItem[]
  totalCount: number
  hasMore: boolean
  isInitialLoading: boolean
  isLoadingMore: boolean
}

type ObjectEventsAction =
  | { type: 'reset' }
  | { type: 'startInitial'; preserveLoaded: boolean }
  | { type: 'applyHeadPage'; items: FrontendEventItem[]; totalCount: number; preserveLoaded: boolean }
  | { type: 'startLoadingMore' }
  | { type: 'applyMorePage'; items: FrontendEventItem[]; totalCount: number }
  | { type: 'finishLoadingMore' }

const initialState: ObjectEventsState = {
  events: [],
  totalCount: 0,
  hasMore: false,
  isInitialLoading: false,
  isLoadingMore: false,
}

export function useObjectEventsFeed(
  api: FrontendClient,
  objectID: number | null,
  enabled: boolean,
): {
  events: FrontendEventItem[]
  totalCount: number
  hasMore: boolean
  isInitialLoading: boolean
  isLoadingMore: boolean
  loadMore: () => void
} {
  const [state, dispatch] = useReducer(objectEventsReducer, initialState)
  const requestVersionRef = useRef(0)
  const cachedObjectIDRef = useRef<number | null>(null)

  useEffect(() => {
    if (objectID != null && cachedObjectIDRef.current === objectID) {
      return
    }

    requestVersionRef.current += 1
    cachedObjectIDRef.current = objectID
    dispatch({ type: 'reset' })
  }, [objectID])

  const loadHeadPage = useCallback(
    async (requestVersion: number, preserveLoaded: boolean) => {
      if (objectID == null) {
        return
      }

      const page = await api.listObjectEvents(objectID, 0, OBJECT_EVENTS_PAGE_SIZE)
      if (requestVersionRef.current !== requestVersion) {
        return
      }

      dispatch({
        type: 'applyHeadPage',
        items: page.items,
        totalCount: page.totalCount,
        preserveLoaded,
      })
    },
    [api, objectID],
  )

  useEffect(() => {
    if (!enabled || objectID == null) {
      dispatch({ type: 'startInitial', preserveLoaded: true })
      return
    }

    const requestVersion = requestVersionRef.current + 1
    requestVersionRef.current = requestVersion
    const preserveLoaded = cachedObjectIDRef.current === objectID && state.events.length > 0
    cachedObjectIDRef.current = objectID

    dispatch({ type: 'startInitial', preserveLoaded })
    void loadHeadPage(requestVersion, preserveLoaded)

    const refreshTimer = window.setInterval(() => {
      void loadHeadPage(requestVersion, true)
    }, OBJECT_EVENTS_REFRESH_MS)

    return () => {
      window.clearInterval(refreshTimer)
    }
  }, [enabled, loadHeadPage, objectID, state.events.length])

  const loadMore = useCallback(() => {
    if (objectID == null || state.isLoadingMore || !state.hasMore) {
      return
    }

    const requestVersion = requestVersionRef.current
    dispatch({ type: 'startLoadingMore' })

    void api
      .listObjectEvents(objectID, state.events.length, OBJECT_EVENTS_PAGE_SIZE)
      .then((page) => {
        if (requestVersionRef.current !== requestVersion) {
          return
        }
        dispatch({ type: 'applyMorePage', items: page.items, totalCount: page.totalCount })
      })
      .finally(() => {
        if (requestVersionRef.current === requestVersion) {
          dispatch({ type: 'finishLoadingMore' })
        }
      })
  }, [api, objectID, state.events.length, state.hasMore, state.isLoadingMore])

  return { ...state, loadMore }
}

function objectEventsReducer(state: ObjectEventsState, action: ObjectEventsAction): ObjectEventsState {
  switch (action.type) {
    case 'reset':
      return initialState
    case 'startInitial':
      return {
        ...state,
        isInitialLoading: !action.preserveLoaded,
      }
    case 'applyHeadPage': {
      const nextEvents = action.preserveLoaded
        ? mergeRecentEvents(state.events, action.items)
        : [...action.items].sort((left, right) => compareEventItemsDesc(left, right))
      return buildObjectEventsState(state, nextEvents, action.totalCount, false)
    }
    case 'startLoadingMore':
      return {
        ...state,
        isLoadingMore: true,
      }
    case 'applyMorePage': {
      const nextEvents = mergeRecentEvents(state.events, action.items)
      return buildObjectEventsState(state, nextEvents, action.totalCount, false)
    }
    case 'finishLoadingMore':
      return {
        ...state,
        isLoadingMore: false,
      }
    default:
      return state
  }
}

function buildObjectEventsState(
  state: ObjectEventsState,
  events: FrontendEventItem[],
  totalCount: number,
  isInitialLoading: boolean,
): ObjectEventsState {
  const nextTotalCount = Math.max(totalCount, events.length)
  return {
    ...state,
    events,
    totalCount: nextTotalCount,
    hasMore: events.length < nextTotalCount,
    isInitialLoading,
    isLoadingMore: false,
  }
}
