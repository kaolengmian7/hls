import axios from 'axios'
import http from '../http'
import type { OrderView, OrderType, OrderStatus, ComplaintStatus, OrderReview } from '../shared/orderLabels'

const publicHttp = axios.create({
  baseURL: ((import.meta.env.VITE_API_BASE as string | undefined)?.replace(/\/+$/, '') ?? '') + '/api',
})

export interface CreateOrderPayload {
  title: string
  orderType: OrderType
  price: number
  commissionRate: number
  bonusRate: number
  playerCount: 1 | 2
  remark?: string
}

export async function listOrders(): Promise<OrderView[]> {
  const res = await http.get<OrderView[]>('/orders')
  return res.data
}

export async function myOrders(): Promise<OrderView[]> {
  const res = await http.get<OrderView[]>('/my-orders')
  return res.data
}

export async function getOrder(code: string): Promise<OrderView> {
  const res = await http.get<OrderView>(`/orders/${code}`)
  return res.data
}

export async function createOrder(payload: CreateOrderPayload): Promise<OrderView> {
  const res = await http.post<OrderView>('/orders', payload)
  return res.data
}

export async function claimOrder(code: string): Promise<OrderView> {
  const res = await http.post<OrderView>(`/orders/${code}/claim`)
  return res.data
}

export async function removeEnforcer(code: string, slot: 'a' | 'b'): Promise<OrderView> {
  const res = await http.delete<OrderView>(`/orders/${code}/enforcer/${slot}`)
  return res.data
}

export async function updateOrderStatus(code: string, status: OrderStatus): Promise<void> {
  await http.put(`/orders/${code}/status`, { status })
}

export async function updateComplaint(code: string, complaintStatus: ComplaintStatus): Promise<void> {
  await http.put(`/orders/${code}/complaint`, { complaintStatus })
}

export interface ReviewPayload {
  attitudeScore: number
  technicalScore: number
  blameBoss: boolean
  late: boolean
  tempUnavailable: boolean
  slowReply: boolean
  noBackpack: boolean
  noChannelCard: boolean
  noSupplyCard: boolean
  noEmotionalValue: boolean
  grabContainer: boolean
  comment: string
}

export interface PublicReviewPayload extends ReviewPayload {
  reviewerName?: string
}

export interface PublicOrderReviewContext {
  order: OrderView
  review: OrderReview | null
}

export async function getOrderReview(code: string): Promise<OrderReview> {
  const res = await http.get<OrderReview>(`/orders/${code}/review`)
  return res.data
}

export async function upsertOrderReview(code: string, payload: ReviewPayload): Promise<OrderReview> {
  const res = await http.put<OrderReview>(`/orders/${code}/review`, payload)
  return res.data
}

export async function getPublicOrderReview(code: string): Promise<PublicOrderReviewContext> {
  const res = await publicHttp.get<PublicOrderReviewContext>(`/public/orders/${code}/review`)
  return res.data
}

export async function upsertPublicOrderReview(code: string, payload: PublicReviewPayload): Promise<OrderReview> {
  const res = await publicHttp.put<OrderReview>(`/public/orders/${code}/review`, payload)
  return res.data
}
