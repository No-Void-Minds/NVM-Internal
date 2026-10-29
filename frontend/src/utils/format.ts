const dateTime = new Intl.DateTimeFormat('id-ID', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

// Menghasilkan format seperti di Stitch, mis. "29 Okt 2026, 09.14".
export function formatDateTime(iso: string): string {
  return dateTime.format(new Date(iso))
}
