import http from '../http'

export type PayrollStatus = 'frozen' | 'pending' | 'paid'

export interface PayrollRow {
  id: number
  orderCode: string
  orderTitle: string
  enforcerName: string
  status: PayrollStatus
  gross: number
  depositDeducted: number
  fineDeducted: number
  netPaid: number
  createdAt: string
  paidAt: string | null
  unlockAt: string | null
}

export async function listPayroll(status: PayrollStatus): Promise<PayrollRow[]> {
  const res = await http.get<PayrollRow[]>('/payroll', { params: { status } })
  return res.data
}

export async function payPayroll(id: number): Promise<PayrollRow> {
  const res = await http.post<PayrollRow>(`/payroll/${id}/pay`)
  return res.data
}
