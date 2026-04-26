import { createHTTPFrontendClient } from './http-client'
import { createWailsFrontendClient } from './wails-client'
import type { FrontendClient } from './frontend-client'

export function resolveFrontendClient(): FrontendClient {
  const wailsClient = createWailsFrontendClient()
  if (wailsClient != null) {
    return wailsClient
  }
  return createHTTPFrontendClient()
}

let _client: FrontendClient | null = null

export function getClient(): FrontendClient {
  if (_client == null) {
    _client = resolveFrontendClient()
  }
  return _client
}
