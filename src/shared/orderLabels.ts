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
  bonusQualified?: boolean
  review?: ReviewSummary | null
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

export interface ReviewSummary {
  attitudeScore: number
  technicalScore: number
  total: number
  bonusQualified: boolean
}

export type ReviewFlagKey =
  | 'blameBoss' | 'late' | 'tempUnavailable' | 'slowReply'
  | 'noBackpack' | 'noChannelCard' | 'noSupplyCard'
  | 'noEmotionalValue' | 'grabContainer'

export interface OrderReview {
  orderCode: string
  attitudeScore: number
  technicalScore: number
  total: number
  bonusQualified: boolean
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
  reviewerName: string
  createdAt: string
}

export const reviewFlagLabels: Record<ReviewFlagKey, string> = {
  blameBoss: '压力指责老板',
  late: '迟到',
  tempUnavailable: '约好时间临时有事',
  slowReply: '群里长时间不回复老板消息',
  noBackpack: '未携带背包',
  noChannelCard: '未携带通道卡',
  noSupplyCard: '未携带物资卡',
  noEmotionalValue: '未提供情绪价值',
  grabContainer: '抢老板的容器',
}
