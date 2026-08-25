import http from '../http'

export interface EnforcerStat {
  userId: number
  nickname: string
  username: string
  depositRequired: number
  depositPaid: number
  depositOwed: number
  orderCount: number
  goodCount: number
  badCount: number
  reviewedCount: number
  goodRate: number | null
  pendingFine: number
}

export async function listEnforcers(): Promise<EnforcerStat[]> {
  const res = await http.get<EnforcerStat[]>('/enforcers')
  return res.data
}

// depositRequired: number sets an override; null clears it (back to global default).
export async function updateEnforcerDeposit(id: number, depositRequired: number | null): Promise<void> {
  await http.put(`/enforcers/${id}/deposit`, { depositRequired })
}

export async function updateEnforcerFine(id: number, pendingFine: number): Promise<void> {
  await http.put(`/enforcers/${id}/fine`, { pendingFine })
}
