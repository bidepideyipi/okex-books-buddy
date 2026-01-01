/**
 * Analysis data types from backend API
 */

export interface SupportResistanceData {
  instrument_id: string
  analysis_time: string
  support_level_1?: string
  support_level_2?: string
  resistance_level_1?: string
  resistance_level_2?: string
}

export interface LargeOrderData {
  instrument_id: string
  analysis_time: string
  large_buy_orders?: string
  large_sell_orders?: string
  large_order_trend?: 'bullish' | 'bearish' | 'neutral'
}

export interface AnalysisResponse {
  instrument_id: string
  support_resistance?: SupportResistanceData
  large_orders?: LargeOrderData
}

export interface TradingPair {
  instId: string
  displayName: string
}
