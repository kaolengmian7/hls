import http from '../http'

export interface TeachingOrderView {
  code: string
  title: string
  student: { id: number; name: string }
  dispatcherName: string
  teacher: { id: number; name: string } | null
  price: number
  commissionRate: number
  bonusRate: number
  bonusQualified: boolean
  totalHours: number
  remainingHours: number
  totalMinutes: number
  remainingMinutes: number
  status: 'dispatching' | 'ongoing' | 'settled'
  earnings: number
  createdAt: string
  newStudentCredentials?: { username: string; password: string }
}

export interface ScorecardView {
  id: number
  durationHours: number
  scores: Record<string, number>
  comment: string
  teacherName: string
  createdAt: string
}

export interface TimeLogView {
  deltaHours: number
  reason: 'initial' | 'topup' | 'lesson'
  priceDelta: number
  operatorName: string
  operatorRole: string
  note: string
  createdAt: string
}

export interface TeachingReviewView {
  attitudeScore: number
  technicalScore: number
  total: number
  bonusQualified: boolean
  comment: string
  reviewerName: string
  createdAt: string
}

export interface SyllabusItem { key: string; label: string; desc: string }
export interface SyllabusCategory { key: string; label: string; items: SyllabusItem[] }
export interface CustomerOption { id: number; nickname: string; username: string }

export interface CreateTeachingPayload {
  title: string
  studentId?: number
  newStudentNickname?: string
  totalHours: number
  price: number
  commissionRate: number
  bonusRate: number
}

export async function listTeachingOrders(): Promise<TeachingOrderView[]> {
  return (await http.get<TeachingOrderView[]>('/teaching-orders')).data
}
export async function getTeachingOrder(code: string): Promise<TeachingOrderView> {
  return (await http.get<TeachingOrderView>(`/teaching-orders/${code}`)).data
}
export async function createTeachingOrder(p: CreateTeachingPayload): Promise<TeachingOrderView> {
  return (await http.post<TeachingOrderView>('/teaching-orders', p)).data
}
export async function claimTeachingOrder(code: string): Promise<TeachingOrderView> {
  return (await http.post<TeachingOrderView>(`/teaching-orders/${code}/claim`)).data
}
export async function topUpTeachingOrder(code: string, addHours: number, addPrice: number): Promise<TeachingOrderView> {
  return (await http.post<TeachingOrderView>(`/teaching-orders/${code}/topup`, { addHours, addPrice })).data
}
export async function settleTeachingOrder(code: string): Promise<TeachingOrderView> {
  return (await http.post<TeachingOrderView>(`/teaching-orders/${code}/settle`)).data
}
export async function listScorecards(code: string): Promise<ScorecardView[]> {
  return (await http.get<ScorecardView[]>(`/teaching-orders/${code}/scorecards`)).data
}
export async function createScorecard(code: string, durationHours: number, scores: Record<string, number>, comment: string): Promise<ScorecardView> {
  return (await http.post<ScorecardView>(`/teaching-orders/${code}/scorecards`, { durationHours, scores, comment })).data
}
export async function listTimeLogs(code: string): Promise<TimeLogView[]> {
  return (await http.get<TimeLogView[]>(`/teaching-orders/${code}/timelogs`)).data
}
export async function getTeachingReview(code: string): Promise<TeachingReviewView | null> {
  try {
    return (await http.get<TeachingReviewView>(`/teaching-orders/${code}/review`)).data
  } catch {
    return null
  }
}
export async function upsertTeachingReview(code: string, attitudeScore: number, technicalScore: number, comment: string): Promise<TeachingReviewView> {
  return (await http.put<TeachingReviewView>(`/teaching-orders/${code}/review`, { attitudeScore, technicalScore, comment })).data
}
export async function listMyTeaching(): Promise<TeachingOrderView[]> {
  return (await http.get<TeachingOrderView[]>('/my-teaching')).data
}
export async function getSyllabus(): Promise<SyllabusCategory[]> {
  return (await http.get<SyllabusCategory[]>('/teaching/syllabus')).data
}
export async function listCustomers(): Promise<CustomerOption[]> {
  return (await http.get<CustomerOption[]>('/customers')).data
}
