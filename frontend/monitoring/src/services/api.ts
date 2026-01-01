/**
 * API service for fetching analysis data from backend
 */

import type { AnalysisResponse } from '../types/analysis'

// API calls are proxied to backend through Vite config
// No need for VITE_API_BASE_URL since we use proxy
const API_BASE_URL = ''

export class ApiError extends Error {
  statusCode?: number
  response?: any

  constructor(
    message: string,
    statusCode?: number,
    response?: any
  ) {
    super(message)
    this.name = 'ApiError'
    this.statusCode = statusCode
    this.response = response
  }
}

/**
 * Fetch analysis data for a specific instrument
 */
export async function fetchAnalysisData(instrumentId: string): Promise<AnalysisResponse> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/analysis/${instrumentId}`)
    
    if (!response.ok) {
      if (response.status === 404) {
        throw new ApiError(
          `No analysis data found for ${instrumentId}`,
          404
        )
      }
      throw new ApiError(
        `HTTP ${response.status}: ${response.statusText}`,
        response.status
      )
    }

    const data: AnalysisResponse = await response.json()
    return data
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    throw new ApiError(
      `Failed to fetch analysis data: ${error instanceof Error ? error.message : 'Unknown error'}`
    )
  }
}

/**
 * WebSocket connection status response
 */
export interface WSConnectionStatus {
  instrument_id: string
  websocket_url: string
  status: 'connected' | 'disconnected' | 'reconnecting'
  connected_at: string
  last_message_at: string
  reconnect_count: number
  messages_received: number
  last_error?: string
}

export interface WSStatusResponse {
  code: number
  message: string
  data: {
    total_connections: number
    active_pairs: number
    connections: WSConnectionStatus[]
  }
}

export interface SingleWSStatusResponse {
  code: number
  message: string
  data: WSConnectionStatus
}

/**
 * Fetch all WebSocket connection statuses
 */
export async function fetchWebSocketStatus(): Promise<WSStatusResponse> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/websocket/status`)
    
    if (!response.ok) {
      throw new ApiError(
        `HTTP ${response.status}: ${response.statusText}`,
        response.status
      )
    }

    const data: WSStatusResponse = await response.json()
    return data
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    throw new ApiError(
      `Failed to fetch WebSocket status: ${error instanceof Error ? error.message : 'Unknown error'}`
    )
  }
}

/**
 * Fetch WebSocket connection status for a specific instrument
 */
export async function fetchWebSocketStatusByInstrument(instrumentId: string): Promise<SingleWSStatusResponse> {
  try {
    const response = await fetch(`${API_BASE_URL}/api/websocket/status/${instrumentId}`)
    
    if (!response.ok) {
      if (response.status === 404) {
        throw new ApiError(
          `No connection found for ${instrumentId}`,
          404
        )
      }
      throw new ApiError(
        `HTTP ${response.status}: ${response.statusText}`,
        response.status
      )
    }

    const data: SingleWSStatusResponse = await response.json()
    return data
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    throw new ApiError(
      `Failed to fetch WebSocket status: ${error instanceof Error ? error.message : 'Unknown error'}`
    )
  }
}

/**
 * Get list of available trading pairs (placeholder - will be implemented in Option A)
 * For now, returns hardcoded list based on common OKEx pairs
 */
export function getAvailablePairs() {
  return [
    { instId: 'BTC-USDT', displayName: 'Bitcoin / USDT' },
    { instId: 'ETH-USDT', displayName: 'Ethereum / USDT' },
    { instId: 'SOL-USDT', displayName: 'Solana / USDT' },
    { instId: 'BNB-USDT', displayName: 'BNB / USDT' },
    { instId: 'XRP-USDT', displayName: 'Ripple / USDT' },
  ]
}
