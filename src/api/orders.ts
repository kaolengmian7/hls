import http from '../http'
import type { OrderView, OrderType, OrderStatus, ComplaintStatus } from '../shared/orderLabels'

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
