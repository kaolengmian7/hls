export type OrderType = 'guarantee' | 'fun' | 'companion' | 'teaching'
export type OrderStatus = 'dispatching' | 'ongoing' | 'settled' | 'refunded'
export type ComplaintStatus = 'none' | 'complained' | 'resolved'

export interface EnforcerRef { id: number; name: string }

export interface OrderView {
  code: string
  title: string
  orderType: OrderType
  price: number
  commissionRate: number
  bonusRate: number
  playerCount: number
  dispatcherName: string
  enforcerA: EnforcerRef | null
  enforcerB: EnforcerRef | null
  remark: string
  status: OrderStatus
  complaintStatus: ComplaintStatus
  earnings: number
  createdAt: string
}

export const orderTypeLabels: Record<OrderType, string> = {
  guarantee: '保底单',
  fun: '趣味单',
  companion: '陪玩单',
  teaching: '教学单',
}

export const orderStatusLabels: Record<OrderStatus, string> = {
  dispatching: '派单中',
  ongoing: '进行中',
  settled: '结单',
  refunded: '退单',
}

export const complaintLabels: Record<ComplaintStatus, string> = {
  none: '无客诉',
  complained: '被投诉',
  resolved: '已处理',
}
